// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package organization

import (
	"context"

	auth_model "gitea.dev/models/auth"
	"gitea.dev/models/db"
	"gitea.dev/models/perm"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/cache"
	"gitea.dev/modules/cachegroup"

	"xorm.io/builder"
)

type ignoreTwoFactorPolicyKey struct{}

// IgnoreTwoFactorPolicy returns a context in which no organization's two-factor policy blocks anyone.
// Code that writes or deletes rows based on a user's access (unassigning, unwatching, trimming allowlists)
// must use it, so a user who is only blocked until they enrol loses nothing.
func IgnoreTwoFactorPolicy(ctx context.Context) context.Context {
	return context.WithValue(ctx, ignoreTwoFactorPolicyKey{}, true)
}

// IsTwoFactorBlocked reports whether the organization requires two-factor authentication and u has
// neither TOTP nor WebAuthn, so u must be treated as a non-member: no member, team or collaborator access.
// Site admins and non-person identities (the organization itself pushing through a deploy key, Actions
// users, the ghost user) are never blocked. The enrolment check is cached per request.
//
// It does not check that u is a member or collaborator: anyone without 2FA counts as blocked. Callers
// must only use it to take away access that came from membership or collaboration.
func (org *Organization) IsTwoFactorBlocked(ctx context.Context, u *user_model.User) (bool, error) {
	if !org.RequireTwoFactor || isTwoFactorPolicyExempt(ctx, u) {
		return false, nil
	}
	return lacksTwoFactor(ctx, u)
}

// isTwoFactorPolicyExempt reports whether no two-factor policy applies to u at all
func isTwoFactorPolicyExempt(ctx context.Context, u *user_model.User) bool {
	if u == nil || u.ID <= 0 || u.IsAdmin || (u.Type != user_model.UserTypeIndividual && u.Type != user_model.UserTypeBot) {
		return true
	}
	ignore, _ := ctx.Value(ignoreTwoFactorPolicyKey{}).(bool)
	return ignore
}

// lacksTwoFactor reports whether u has neither TOTP nor WebAuthn, cached per request
func lacksTwoFactor(ctx context.Context, u *user_model.User) (bool, error) {
	has, err := cache.GetWithContextCache(ctx, cachegroup.UserHasTwoFactor, u.ID, auth_model.HasTwoFactorOrWebAuthn)
	if err != nil {
		return false, err
	}
	return !has, nil
}

// IsTwoFactorBlockedByAnyOrg reports whether u is a member of, or a collaborator on a repository of, an
// organization whose two-factor policy blocks them. It reads the database rather than the session's
// "has 2FA" flag, which goes stale when a factor is removed in another session.
func IsTwoFactorBlockedByAnyOrg(ctx context.Context, u *user_model.User) (bool, error) {
	if isTwoFactorPolicyExempt(ctx, u) {
		return false, nil
	}
	// it runs on every rendered page: most instances have no such organization, and most users are in none
	if anyOrg, err := db.GetEngine(ctx).Where(builder.Eq{"require_two_factor": true}).Exist(new(Organization)); err != nil || !anyOrg {
		return false, err
	}
	inPolicyOrg, err := db.GetEngine(ctx).
		Where(builder.Eq{"uid": u.ID}).
		And(builder.In("org_id", user_model.RequireTwoFactorOrgIDsBuilder())).
		Exist(new(OrgUser))
	if err != nil {
		return false, err
	}
	if !inPolicyOrg {
		inPolicyOrg, err = db.GetEngine(ctx).Table("collaboration").
			Join("INNER", "repository", "`repository`.id = `collaboration`.repo_id").
			Where(builder.Eq{"`collaboration`.user_id": u.ID}).
			And(builder.In("`repository`.owner_id", user_model.RequireTwoFactorOrgIDsBuilder())).
			Exist()
		if err != nil || !inPolicyOrg {
			return false, err
		}
	}
	return lacksTwoFactor(ctx, u)
}

// GetOwnedOrgsRequiringTwoFactor returns the organizations that require two-factor authentication and
// that u owns; such an owner must keep a second factor, or they would lock themselves out of the org
func GetOwnedOrgsRequiringTwoFactor(ctx context.Context, uid int64) ([]*Organization, error) {
	orgs := make([]*Organization, 0, 2)
	return orgs, db.GetEngine(ctx).
		Where(builder.Eq{"require_two_factor": true}).
		And(builder.In("id", builder.Select("`team`.org_id").From("team").
			Join("INNER", "team_user", "`team_user`.team_id = `team`.id").
			Where(builder.Eq{"`team_user`.uid": uid, "`team`.authorize": perm.AccessModeOwner}))).
		Asc("name").
		Find(&orgs)
}

// notTwoFactorBlockedOrgCond is the SQL twin of IsTwoFactorBlocked for a query over orgIDCol
func notTwoFactorBlockedOrgCond(orgIDCol string, userID int64) builder.Cond {
	return builder.Or(
		user_model.TwoFactorPolicyExemptCond(userID),
		builder.NotIn(orgIDCol, user_model.RequireTwoFactorOrgIDsBuilder()),
	)
}

// blockableWithoutTwoFactorCond matches rows whose userIDCol is a user an organization's policy would block:
// enrolled in neither TOTP nor WebAuthn, and not a site admin
func blockableWithoutTwoFactorCond(userIDCol string) builder.Cond {
	return builder.Not{user_model.HasTwoFactorCond(userIDCol)}.
		And(builder.NotIn(userIDCol, builder.Select("`user`.id").From("`user`").Where(builder.Eq{"`user`.is_admin": true})))
}

// CountOrgMembersWithoutTwoFactor counts the organization's members its two-factor policy would block:
// enrolled in neither TOTP nor WebAuthn, and not site admins, whom the policy never blocks.
func CountOrgMembersWithoutTwoFactor(ctx context.Context, orgID int64) (int64, error) {
	return db.GetEngine(ctx).
		Where(builder.Eq{"`org_user`.org_id": orgID}).
		And(blockableWithoutTwoFactorCond("`org_user`.uid")).
		Count(new(OrgUser))
}

// CountOrgOutsideCollaboratorsWithoutTwoFactor counts the non-members who collaborate on any of the
// organization's repositories and whom its two-factor policy would block, as CountOrgMembersWithoutTwoFactor.
func CountOrgOutsideCollaboratorsWithoutTwoFactor(ctx context.Context, orgID int64) (int64, error) {
	return db.GetEngine(ctx).Table("collaboration").
		Join("INNER", "repository", "`repository`.id = `collaboration`.repo_id").
		Where(builder.Eq{"`repository`.owner_id": orgID}).
		And(builder.NotIn("`collaboration`.user_id", builder.Select("uid").From("org_user").Where(builder.Eq{"org_id": orgID}))).
		And(blockableWithoutTwoFactorCond("`collaboration`.user_id")).
		Distinct("`collaboration`.user_id").
		Count()
}
