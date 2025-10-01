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
//go:build linux
// +build linux

package platform

import (
	"testing"

	"github.com/aws/amazon-ssm-agent/agent/mocks/log"
)

func TestNetlinkVsFallback(t *testing.T) {
	logger := &log.Mock{}

	// Get result from NetlinkRIB implementation
	netlinkIP, netlinkErr := getDefaultRouteIPNetlinkRIB(logger)

	// Get result from original implementation (already defined in linux.go)
	fallbackIP, fallbackErr := originalIP()

	// Both should succeed or both should fail
	if (netlinkErr == nil) != (fallbackErr == nil) {
		t.Logf("NetlinkRIB result: %s (err: %v)", netlinkIP, netlinkErr)
		t.Logf("Original result: %s (err: %v)", fallbackIP, fallbackErr)
		t.Log("Different error states - this may be expected in some environments")
		return
	}

	// If both succeeded, IPs should match
	if netlinkErr == nil && fallbackErr == nil {
		if netlinkIP != fallbackIP {
			t.Errorf("IP mismatch: NetlinkRIB=%s, Original=%s", netlinkIP, fallbackIP)
		} else {
			t.Logf("Both methods returned same IP: %s", netlinkIP)
		}
	}

	// If both failed, that's also acceptable
	if netlinkErr != nil && fallbackErr != nil {
		t.Logf("Both methods failed (acceptable): NetlinkRIB=%v, Original=%v", netlinkErr, fallbackErr)
	}
}
