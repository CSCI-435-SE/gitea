// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import "xorm.io/builder"

// RequireTwoFactorOrgIDsBuilder selects the IDs of the organizations that require two-factor authentication.
func RequireTwoFactorOrgIDsBuilder() *builder.Builder {
	return builder.Select("`user`.id").From("`user`").Where(builder.Eq{"`user`.require_two_factor": true})
}

// HasTwoFactorCond matches rows whose userIDCol (e.g. "`org_user`.uid") is a user enrolled in TOTP or WebAuthn.
func HasTwoFactorCond(userIDCol string) builder.Cond {
	return builder.Or(
		builder.In(userIDCol, builder.Select("uid").From("two_factor")),
		builder.In(userIDCol, builder.Select("user_id").From("webauthn_credential")),
	)
}

// TwoFactorPolicyExemptCond is the SQL twin of organization.IsTwoFactorBlocked: it is true when no
// organization's two-factor policy can block the user, because they are a site admin, not a person
// (an organization pushing through a deploy key) or enrolled in TOTP or WebAuthn.
func TwoFactorPolicyExemptCond(userID int64) builder.Cond {
	return builder.Or(
		builder.Exists(builder.Select("`user`.id").From("`user`").Where(builder.Eq{"`user`.id": userID}.And(builder.Or(
			builder.Eq{"`user`.is_admin": true},
			builder.NotIn("`user`.type", UserTypeIndividual, UserTypeBot),
		)))),
		builder.Exists(builder.Select("id").From("two_factor").Where(builder.Eq{"uid": userID})),
		builder.Exists(builder.Select("id").From("webauthn_credential").Where(builder.Eq{"user_id": userID})),
	)
}
