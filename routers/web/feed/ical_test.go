// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package feed

import (
	"testing"

	"gitea.dev/modules/optional"

	"github.com/stretchr/testify/assert"
)

func TestParseICalState(t *testing.T) {
	assert.Equal(t, optional.Some(false), parseICalState(""))
	assert.Equal(t, optional.Some(false), parseICalState("open"))
	assert.Equal(t, optional.Some(true), parseICalState("closed"))
	assert.Equal(t, optional.None[bool](), parseICalState("all"))
	assert.Equal(t, optional.Some(false), parseICalState("bogus")) // unknown values fall back to the default
}

func TestICalIsPull(t *testing.T) {
	assert.Equal(t, optional.None[bool](), icalIsPull(true, true))
	assert.Equal(t, optional.Some(true), icalIsPull(false, true))
	assert.Equal(t, optional.Some(false), icalIsPull(true, false))
}
