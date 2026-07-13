// Copyright 2026 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not
// use this file except in compliance with the License. A copy of the
// License is located at
//
// http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied. See the License for the specific language governing
// permissions and limitations under the License.

//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package fileutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCreateHardenedFile_NewFile verifies that a new file is created with
// RWPermission (0600).
func TestCreateHardenedFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "harden-new-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "data")

	err = createHardenedFile(testFile)
	assert.NoError(t, err)

	fi, err := os.Stat(testFile)
	assert.NoError(t, err)
	assert.Equal(t, RWPermission, fi.Mode().Perm(),
		"expected new file to be created with %o, got %o", RWPermission, fi.Mode().Perm())
}
