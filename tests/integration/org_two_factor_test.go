// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	asymkey_model "gitea.dev/models/asymkey"
	auth_model "gitea.dev/models/auth"
	"gitea.dev/models/perm"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/private"
	api "gitea.dev/modules/structs"
	"gitea.dev/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setOrgRequireTwoFactor(t *testing.T, orgName string, require2FA bool) {
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: orgName})
	org.RequireTwoFactor = require2FA
	require.NoError(t, user_model.UpdateUserCols(t.Context(), org, "require_two_factor"))
}

// enrolTOTP gives the user a TOTP device directly, so an already signed-in session or token keeps working.
// The device is removed when the (sub)test ends.
func enrolTOTP(t *testing.T, userName string) {
	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: userName})
	tfa := &auth_model.TwoFactor{UID: u.ID}
	require.NoError(t, tfa.SetSecret("JBSWY3DPEHPK3PXP"))
	require.NoError(t, auth_model.NewTwoFactor(t.Context(), tfa))
	cleanupCtx := context.WithoutCancel(t.Context()) // t.Context() is canceled before cleanups run
	t.Cleanup(func() {
		_, _, err := auth_model.DisableTwoFactor(cleanupCtx, u.ID)
		assert.NoError(t, err)
	})
}

func repoFullNames(repos []*api.Repository) []string {
	names := make([]string, 0, len(repos))
	for _, r := range repos {
		names = append(names, r.FullName)
	}
	return names
}

// org3 fixtures: user4 is a member with write on private repo3 through team1, and user10 is an
// outside collaborator with write on public repo21. Neither has 2FA.
func TestOrgTwoFactorPolicy(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("API", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer setOrgRequireTwoFactor(t, "org3", false)

		user4Token := getUserToken(t, "user4", auth_model.AccessTokenScopeReadRepository, auth_model.AccessTokenScopeReadOrganization,
			auth_model.AccessTokenScopeReadUser, auth_model.AccessTokenScopeReadIssue)
		user10Token := getUserToken(t, "user10", auth_model.AccessTokenScopeReadRepository)

		getRepo := func(token, repo string, expectedStatus int) *api.Repository {
			resp := MakeRequest(t, NewRequest(t, "GET", "/api/v1/repos/org3/"+repo).AddTokenAuth(token), expectedStatus)
			if expectedStatus != http.StatusOK {
				return nil
			}
			return DecodeJSON(t, resp, &api.Repository{})
		}
		listMyRepos := func() []string {
			resp := MakeRequest(t, NewRequest(t, "GET", "/api/v1/user/repos").AddTokenAuth(user4Token), http.StatusOK)
			return repoFullNames(DecodeJSON(t, resp, []*api.Repository{}))
		}
		searchIssueIDs := func() []int64 {
			resp := MakeRequest(t, NewRequest(t, "GET", "/api/v1/repos/issues/search?state=all&type=issues").AddTokenAuth(user4Token), http.StatusOK)
			var ids []int64
			for _, issue := range DecodeJSON(t, resp, []*api.Issue{}) {
				ids = append(ids, issue.ID)
			}
			return ids
		}

		// before the policy: ordinary member and collaborator access
		getRepo(user4Token, "repo3", http.StatusOK)
		assert.True(t, getRepo(user10Token, "repo21", http.StatusOK).Permissions.Push)
		MakeRequest(t, NewRequest(t, "GET", "/api/v1/orgs/org3/teams").AddTokenAuth(user4Token), http.StatusOK)
		assert.Contains(t, listMyRepos(), "org3/repo3")
		assert.Contains(t, searchIssueIDs(), int64(6)) // issue6 lives in org3/repo3

		setOrgRequireTwoFactor(t, "org3", true)

		// blocked: exactly what a signed-in non-member gets
		getRepo(user4Token, "repo3", http.StatusNotFound)
		repo21 := getRepo(user10Token, "repo21", http.StatusOK)
		assert.True(t, repo21.Permissions.Pull)
		assert.False(t, repo21.Permissions.Push)
		resp := MakeRequest(t, NewRequest(t, "GET", "/api/v1/orgs/org3/teams").AddTokenAuth(user4Token), http.StatusForbidden)
		assert.Contains(t, resp.Body.String(), "requires two-factor authentication")
		assert.NotContains(t, listMyRepos(), "org3/repo3")
		assert.NotContains(t, searchIssueIDs(), int64(6))

		// enrolling restores access on the next request; nothing was removed meanwhile
		enrolTOTP(t, "user4")
		getRepo(user4Token, "repo3", http.StatusOK)
		MakeRequest(t, NewRequest(t, "GET", "/api/v1/orgs/org3/teams").AddTokenAuth(user4Token), http.StatusOK)
		assert.Contains(t, listMyRepos(), "org3/repo3")
		assert.Contains(t, searchIssueIDs(), int64(6))
	})

	t.Run("GitHTTP", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer setOrgRequireTwoFactor(t, "org3", false)

		token := getUserToken(t, "user4", auth_model.AccessTokenScopeReadRepository)
		infoRefs := func(expectedStatus int) {
			req := NewRequest(t, "GET", "/org3/repo3.git/info/refs?service=git-upload-pack").AddBasicAuth(token, "x-oauth-basic")
			MakeRequest(t, req, expectedStatus)
		}

		infoRefs(http.StatusOK)
		setOrgRequireTwoFactor(t, "org3", true)
		infoRefs(http.StatusNotFound)
		enrolTOTP(t, "user4")
		infoRefs(http.StatusOK)
	})

	t.Run("Web", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer setOrgRequireTwoFactor(t, "org3", false)

		session := loginUser(t, "user4")
		const banner = "An organization you belong to requires two-factor authentication"
		session.MakeRequest(t, NewRequest(t, "GET", "/org3/repo3"), http.StatusOK)
		session.MakeRequest(t, NewRequest(t, "GET", "/org/org3/teams"), http.StatusOK)
		assert.NotContains(t, session.MakeRequest(t, NewRequest(t, "GET", "/"), http.StatusOK).Body.String(), banner)

		setOrgRequireTwoFactor(t, "org3", true)
		session.MakeRequest(t, NewRequest(t, "GET", "/org3/repo3"), http.StatusNotFound)
		session.MakeRequest(t, NewRequest(t, "GET", "/org/org3/teams"), http.StatusNotFound)
		assert.Contains(t, session.MakeRequest(t, NewRequest(t, "GET", "/"), http.StatusOK).Body.String(), banner)
		assert.NotContains(t, session.MakeRequest(t, NewRequest(t, "GET", "/user/settings/security/two_factor/enroll"), http.StatusOK).Body.String(), banner)

		enrolTOTP(t, "user4")
		session.MakeRequest(t, NewRequest(t, "GET", "/org3/repo3"), http.StatusOK)
		session.MakeRequest(t, NewRequest(t, "GET", "/org/org3/teams"), http.StatusOK)
		assert.NotContains(t, session.MakeRequest(t, NewRequest(t, "GET", "/"), http.StatusOK).Body.String(), banner)
	})

	t.Run("Settings", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer setOrgRequireTwoFactor(t, "org3", false)

		session := loginUser(t, "user2") // org3 owner, no 2FA yet
		postSettings := func(requireTwoFactor bool) {
			values := map[string]string{"full_name": "org3"}
			if requireTwoFactor {
				values["require_two_factor"] = "on"
			}
			session.MakeRequest(t, NewRequestWithValues(t, "POST", "/org/org3/settings", values), http.StatusSeeOther)
		}
		orgRequires := func() bool {
			return unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "org3"}).RequireTwoFactor
		}

		postSettings(true)
		assert.False(t, orgRequires(), "an owner without 2FA can't turn it on")
		flash := session.GetCookieFlashMessage()
		assert.Contains(t, flash.ErrorMsg, "Set up two-factor authentication on your own account")

		enrolTOTP(t, "user2")
		postSettings(true)
		assert.True(t, orgRequires())

		// the owner can't remove their only second factor while the org requires one
		session.MakeRequest(t, NewRequest(t, "POST", "/user/settings/security/two_factor/disable"), http.StatusSeeOther)
		assert.Contains(t, session.GetCookieFlashMessage().ErrorMsg, "You own an organization that requires two-factor authentication (org3)")
		unittest.AssertExistsAndLoadBean(t, &auth_model.TwoFactor{UID: 2})

		settingsPage := session.MakeRequest(t, NewRequest(t, "GET", "/org/org3/settings"), http.StatusOK).Body.String()
		assert.Contains(t, settingsPage, `name="require_two_factor" checked`)
		assert.Contains(t, settingsPage, "Without two-factor authentication: 2 members and 1 outside collaborators")
		membersPage := session.MakeRequest(t, NewRequest(t, "GET", "/org/org3/members"), http.StatusOK).Body.String()
		assert.Contains(t, membersPage, "Members without it: 2.")
		assert.Contains(t, membersPage, "No access until 2FA is set up")
		collaboratorsPage := session.MakeRequest(t, NewRequest(t, "GET", "/org3/repo21/settings/collaboration"), http.StatusOK).Body.String()
		assert.Contains(t, collaboratorsPage, "No access until 2FA is set up", "user10 collaborates on repo21 without 2FA")

		postSettings(false) // an unticked checkbox is not posted at all, and must still turn it off
		assert.False(t, orgRequires())
	})

	t.Run("Packages", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer setOrgRequireTwoFactor(t, "org3", false)

		token := getUserToken(t, "user2", auth_model.AccessTokenScopeWritePackage)
		upload := func(version string, expectedStatus int) {
			req := NewRequestWithBody(t, "PUT", "/api/packages/org3/generic/two-factor-pkg/"+version+"/file.bin", strings.NewReader("data")).AddTokenAuth(token)
			MakeRequest(t, req, expectedStatus)
		}

		upload("1.0.0", http.StatusCreated)
		setOrgRequireTwoFactor(t, "org3", true)
		upload("1.0.1", http.StatusUnauthorized)
	})
}

func TestOrgTwoFactorPolicySSH(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, _ *url.URL) {
		defer setOrgRequireTwoFactor(t, "org3", false)
		ctx := t.Context()

		// key 1 belongs to user2, an org3 owner without 2FA
		_, extra := private.ServCommand(ctx, 1, "org3", "repo3", perm.AccessModeRead, "git-upload-pack", "")
		require.NoError(t, extra.Error)

		deployKey, err := asymkey_model.AddDeployKey(ctx, 3, "two-factor-deploy", "sk-ecdsa-sha2-nistp256@openssh.com AAAAInNrLWVjZHNhLXNoYTItbmlzdHAyNTZAb3BlbnNzaC5jb20AAAAIbmlzdHAyNTYAAABBBGXEEzWmm1dxb+57RoK5KVCL0w2eNv9cqJX2AGGVlkFsVDhOXHzsadS3LTK4VlEbbrDMJdoti9yM8vclA8IeRacAAAAEc3NoOg== nocomment", false)
		require.NoError(t, err)

		setOrgRequireTwoFactor(t, "org3", true)
		_, extra = private.ServCommand(ctx, 1, "org3", "repo3", perm.AccessModeRead, "git-upload-pack", "")
		assert.Error(t, extra.Error, "a member without 2FA can't clone over SSH")
		results, extra := private.ServCommand(ctx, deployKey.KeyID, "org3", "repo3", perm.AccessModeWrite, "git-receive-pack", "")
		require.NoError(t, extra.Error, "deploy keys are unaffected")
		assert.Equal(t, deployKey.ID, results.DeployKeyID)
	})
}
