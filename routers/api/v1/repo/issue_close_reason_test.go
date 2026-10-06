// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"strings"
	"testing"

	"gitea.dev/models/unittest"
	"gitea.dev/services/contexttest"

	"github.com/stretchr/testify/assert"
)

func TestAPICloseReasonCheck(t *testing.T) {
	unittest.PrepareTestEnv(t)

	cases := []struct {
		name   string
		reason apiCloseReason
		closes bool
		isPull bool
		index  int64
		ok     bool
	}{
		{"no fields", newAPICloseReason("", "", 0), false, false, 1, true}, // old scripts
		{"valid", newAPICloseReason("not_planned", "", 0), true, false, 1, true},
		{"duplicate", newAPICloseReason("duplicate", "", 4), true, false, 1, true},
		{"new item duplicate", newAPICloseReason("duplicate", "", 4), true, false, 0, true},
		{"not closing", newAPICloseReason("completed", "", 0), false, false, 1, false},
		{"text alone, not closing", newAPICloseReason("", "why", 0), false, false, 1, false},
		{"whitespace text alone, not closing", newAPICloseReason("", "  ", 0), false, false, 1, true}, // no reason given, so a close records the default
		{"unknown name", newAPICloseReason("wontfix", "", 0), true, false, 1, false},
		{"completed on a pull request", newAPICloseReason("completed", "", 0), true, true, 2, false},
		{"other without text", newAPICloseReason("other", "  ", 0), true, false, 1, false},
		{"other too long", newAPICloseReason("other", strings.Repeat("a", 256), 0), true, false, 1, false},
		{"text with completed", newAPICloseReason("completed", "why", 0), true, false, 1, false},
		{"duplicate of itself", newAPICloseReason("duplicate", "", 1), true, false, 1, false},
		{"duplicate missing", newAPICloseReason("duplicate", "", 999), true, false, 1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, resp := contexttest.MockAPIContext(t, "user2/repo1")
			contexttest.LoadRepo(t, ctx, 1)
			assert.Equal(t, c.ok, c.reason.check(ctx, c.closes, c.isPull, c.index))
			if !c.ok {
				assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
			}
		})
	}
}

func TestAPICloseReasonTextMessage(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, resp := contexttest.MockAPIContext(t, "user2/repo1")
	contexttest.LoadRepo(t, ctx, 1)
	newAPICloseReason("other", "", 0).check(ctx, true, false, 1)
	assert.Contains(t, resp.Body.String(), "close_reason_text: text is empty")
	assert.NotContains(t, resp.Body.String(), "[reason:") // names only, never the stored number
}

func TestAPICloseReasonNotAllowedMessage(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, resp := contexttest.MockAPIContext(t, "user2/repo1")
	contexttest.LoadRepo(t, ctx, 1)
	newAPICloseReason("completed", "", 0).check(ctx, true, true, 2)
	// scripts see names, never the stored numbers
	assert.Contains(t, resp.Body.String(), `close_reason \"completed\" is not allowed for a pull request, use one of: not_planned, duplicate, other`)
}
