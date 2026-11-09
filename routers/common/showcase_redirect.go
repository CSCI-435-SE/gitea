// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"net/http"

	"gitea.dev/modules/httplib"
)

// ShowcaseRedirect sends the user on to the page named by the "next" parameter
func ShowcaseRedirect(resp http.ResponseWriter, req *http.Request) {
	next := req.FormValue("next")
	if next == "" || !httplib.IsCurrentGiteaSiteURL(req.Context(), next) {
		next = "/"
	}
	http.Redirect(resp, req, next, http.StatusSeeOther)
}
