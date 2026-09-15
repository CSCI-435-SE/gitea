// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package misc

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gitea.dev/modules/setting"
	"gitea.dev/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRobotsTxt(t *testing.T) {
	serve := func(method string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		RobotsTxt(w, httptest.NewRequest(method, "/robots.txt", nil))
		return w
	}

	t.Run("BuiltinDefault", func(t *testing.T) {
		defer test.MockVariableValue(&setting.CustomPath, t.TempDir())()
		w := serve(http.MethodGet)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, defaultRobotsTxt, w.Body.String())
	})

	t.Run("CustomOverridesDefault", func(t *testing.T) {
		dir := t.TempDir()
		defer test.MockVariableValue(&setting.CustomPath, dir)()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "public"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "public", "robots.txt"), []byte("custom"), 0o644))
		assert.Equal(t, "custom", serve(http.MethodGet).Body.String())
	})

	t.Run("LegacyCustomStillWins", func(t *testing.T) {
		dir := t.TempDir()
		defer test.MockVariableValue(&setting.CustomPath, dir)()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "robots.txt"), []byte("legacy"), 0o644))
		assert.Equal(t, "legacy", serve(http.MethodGet).Body.String())
	})

	t.Run("HeadSendsNoBody", func(t *testing.T) {
		defer test.MockVariableValue(&setting.CustomPath, t.TempDir())()
		w := serve(http.MethodHead)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Body.String())
	})
}
