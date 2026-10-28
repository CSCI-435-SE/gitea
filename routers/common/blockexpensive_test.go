// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlockExpensive(t *testing.T) {
	cases := []struct {
		expensive bool
		routePath string
	}{
		{false, "/user/xxx"},
		{false, "/login/xxx"},
		{true, "/{username}/{reponame}/archive/xxx"},
		{true, "/{username}/{reponame}/graph"},
		{true, "/{username}/{reponame}/src/xxx"},
		{true, "/{username}/{reponame}/wiki/xxx"},
		{true, "/{username}/{reponame}/activity/xxx"},
		{true, "/{username}/{reponame}/issues.ics"},      // the iCal handler hides this feed in expensive mode
		{false, "/{username}/{reponame}/milestones.ics"}, // not blocked, so its feed stays available
	}
	for _, c := range cases {
		assert.Equal(t, c.expensive, isRoutePathExpensive(c.routePath), "routePath: %s", c.routePath)
	}
}
