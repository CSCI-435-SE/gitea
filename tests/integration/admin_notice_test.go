// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"gitea.dev/tests"
)

func TestAdminNoticeOperationsMenu(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user1")
	resp := session.MakeRequest(t, NewRequest(t, "GET", "/-/admin/notices"), http.StatusOK)
	doc := NewHTMLParser(t, resp.Body)

	AssertHTMLElement(t, doc, "tfoot .ui.dropdown", false)
	// the click handlers in initAdminNotice bind to ".select.action", which the menu must keep
	AssertHTMLElement(t, doc, "tfoot aria-menu[placement='top-start'] > .menu > .item.select.action[data-action]", 3)
}
