// Copyright 2025 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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

// TestIsPrivilegedAccessOnlyUnix tests the Unix-specific implementation of IsPrivilegedAccessOnly
func TestIsPrivilegedAccessOnly(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "privileged-access-test-unix")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test cases with different permissions
	testCases := []struct {
		name        string
		permissions os.FileMode
		expected    bool
	}{
		{"Owner read only (0400)", 0400, true},
		{"Owner read-write (0600)", 0600, true},
		{"Owner read-execute (0500)", 0500, true},
		{"Owner all permissions (0700)", 0700, true},
		{"Group read (0640)", 0640, false},
		{"Group write (0620)", 0620, false},
		{"Group execute (0610)", 0610, false},
		{"Other read (0604)", 0604, false},
		{"Other write (0602)", 0602, false},
		{"Other execute (0601)", 0601, false},
		{"All read (0444)", 0444, false},
		{"All permissions (0777)", 0777, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test file with specific permissions
			testFilePath := filepath.Join(tempDir, tc.name+".txt")
			os.WriteFile(testFilePath, []byte("test content"), tc.permissions)
			os.Chmod(testFilePath, tc.permissions)

			result, err := IsPrivilegedAccessOnly(testFilePath)
			assert.Equal(t, tc.expected, result, "Expected IsPrivilegedAccessOnly to return %v for permissions %o", tc.expected, tc.permissions)

			if !tc.expected {
				assert.Error(t, err, "Expected an error for permissions %o", tc.permissions)
			} else {
				assert.NoError(t, err, "Expected no error for permissions %o", tc.permissions)
			}
		})
	}

	// Test with a file that doesn't exist
	nonExistentFile := filepath.Join(tempDir, "non-existent-file.txt")
	result, err := IsPrivilegedAccessOnly(nonExistentFile)
	assert.False(t, result, "Expected false for non-existent file")
	assert.Error(t, err, "Expected an error for non-existent file")
	assert.True(t, os.IsNotExist(err), "Expected 'file not found' error for non-existent file")
}
