// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package organization

import (
	"context"

	auth_model "gitea.dev/models/auth"
	"gitea.dev/models/db"
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
func (org *Organization) IsTwoFactorBlocked(ctx context.Context, u *user_model.User) (bool, error) {
	if !org.RequireTwoFactor {
		return false, nil
	}
	return lacksPolicyTwoFactor(ctx, u)
}

// lacksPolicyTwoFactor reports whether a two-factor policy applies to u and u has neither TOTP nor WebAuthn
func lacksPolicyTwoFactor(ctx context.Context, u *user_model.User) (bool, error) {
	if u == nil || u.ID <= 0 || u.IsAdmin || (u.Type != user_model.UserTypeIndividual && u.Type != user_model.UserTypeBot) {
		return false, nil
	}
	if ignore, _ := ctx.Value(ignoreTwoFactorPolicyKey{}).(bool); ignore {
		return false, nil
	}
	has, err := cache.GetWithContextCache(ctx, cachegroup.UserHasTwoFactor, u.ID, auth_model.HasTwoFactorOrWebAuthn)
	if err != nil {
		return false, err
	}
	return !has, nil
}

// IsTwoFactorBlockedByAnyOrg reports whether u is a member of, or a collaborator on a repository of, an
// organization whose two-factor policy blocks them
func IsTwoFactorBlockedByAnyOrg(ctx context.Context, u *user_model.User) (bool, error) {
	if lacks, err := lacksPolicyTwoFactor(ctx, u); err != nil || !lacks {
		return false, err
	}
	isMember, err := db.GetEngine(ctx).
		Where(builder.Eq{"uid": u.ID}).
		And(builder.In("org_id", user_model.RequireTwoFactorOrgIDsBuilder())).
		Exist(new(OrgUser))
	if err != nil || isMember {
		return isMember, err
	}
	return db.GetEngine(ctx).Table("collaboration").
		Join("INNER", "repository", "`repository`.id = `collaboration`.repo_id").
		Where(builder.Eq{"`collaboration`.user_id": u.ID}).
		And(builder.In("`repository`.owner_id", user_model.RequireTwoFactorOrgIDsBuilder())).
		Exist()
}

// notTwoFactorBlockedOrgCond is the SQL twin of IsTwoFactorBlocked for a query over orgIDCol
func notTwoFactorBlockedOrgCond(orgIDCol string, userID int64) builder.Cond {
	return builder.Or(
		user_model.TwoFactorPolicyExemptCond(userID),
		builder.NotIn(orgIDCol, user_model.RequireTwoFactorOrgIDsBuilder()),
	)
}

// CountOrgMembersWithoutTwoFactor counts the organization's members enrolled in neither TOTP nor WebAuthn.
func CountOrgMembersWithoutTwoFactor(ctx context.Context, orgID int64) (int64, error) {
	return db.GetEngine(ctx).
		Where(builder.Eq{"`org_user`.org_id": orgID}).
		And(builder.Not{user_model.HasTwoFactorCond("`org_user`.uid")}).
		Count(new(OrgUser))
}

// CountOrgOutsideCollaboratorsWithoutTwoFactor counts the non-members who collaborate on any of the
// organization's repositories and are enrolled in neither TOTP nor WebAuthn.
func CountOrgOutsideCollaboratorsWithoutTwoFactor(ctx context.Context, orgID int64) (int64, error) {
	return db.GetEngine(ctx).Table("collaboration").
		Join("INNER", "repository", "`repository`.id = `collaboration`.repo_id").
		Where(builder.Eq{"`repository`.owner_id": orgID}).
		And(builder.NotIn("`collaboration`.user_id", builder.Select("uid").From("org_user").Where(builder.Eq{"org_id": orgID}))).
		And(builder.Not{user_model.HasTwoFactorCond("`collaboration`.user_id")}).
		Distinct("`collaboration`.user_id").
		Count()
}
