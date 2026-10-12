// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package util

import "strings"

// TrimNames returns a copy of names with surrounding whitespace removed.
func TrimNames(names []string) []string {
	out := make([]string, 0, len(names))
	for i := 0; i < len(names); i++ {
		out = append(out, strings.TrimSpace(names[i]))
	}
	return out
}
