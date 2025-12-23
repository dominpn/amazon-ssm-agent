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

package azure

import (
	"testing"
	"time"
)

func TestCacheExpiration(t *testing.T) {
	// Reset cache before test
	cacheLock.Lock()
	cache = nil
	cacheLock.Unlock()

	// Create mock metadata
	mockMetadata := &azureMetadataResponse{}
	mockMetadata.Compute.VMID = "test-vm-id"
	mockMetadata.Compute.Location = "eastus"
	mockMetadata.Compute.Zone = "1"
	mockMetadata.Compute.VMSize = "Standard_D2s_v3"

	// Set cache with current time
	cacheLock.Lock()
	cache = &cachedMetadata{
		data:      mockMetadata,
		timestamp: time.Now(),
	}
	cacheLock.Unlock()

	// Verify cache is valid (less than 5 minutes old)
	cacheLock.Lock()
	isValid := cache != nil && time.Since(cache.timestamp) < cacheExpirationTime
	cacheLock.Unlock()

	if !isValid {
		t.Error("Cache should be valid immediately after setting")
	}

	// Set cache with old timestamp (6 minutes ago)
	cacheLock.Lock()
	cache = &cachedMetadata{
		data:      mockMetadata,
		timestamp: time.Now().Add(-6 * time.Minute),
	}
	cacheLock.Unlock()

	// Verify cache is expired
	cacheLock.Lock()
	isExpired := cache == nil || time.Since(cache.timestamp) >= cacheExpirationTime
	cacheLock.Unlock()

	if !isExpired {
		t.Error("Cache should be expired after 6 minutes")
	}
}

func TestCacheInvalidation(t *testing.T) {
	// Reset cache before test
	cacheLock.Lock()
	cache = nil
	cacheLock.Unlock()

	// Create mock metadata with timestamp 4 minutes ago (still valid)
	mockMetadata := &azureMetadataResponse{}
	mockMetadata.Compute.VMID = "test-vm-id-1"

	cacheLock.Lock()
	cache = &cachedMetadata{
		data:      mockMetadata,
		timestamp: time.Now().Add(-4 * time.Minute),
	}
	cacheLock.Unlock()

	// Cache should still be valid
	cacheLock.Lock()
	isValid := cache != nil && time.Since(cache.timestamp) < cacheExpirationTime
	vmID := cache.data.Compute.VMID
	cacheLock.Unlock()

	if !isValid {
		t.Error("Cache should still be valid at 4 minutes")
	}

	if vmID != "test-vm-id-1" {
		t.Errorf("Expected VM ID 'test-vm-id-1', got '%s'", vmID)
	}

	// Now set cache with timestamp 6 minutes ago (expired)
	mockMetadata2 := &azureMetadataResponse{}
	mockMetadata2.Compute.VMID = "test-vm-id-2"

	cacheLock.Lock()
	cache = &cachedMetadata{
		data:      mockMetadata2,
		timestamp: time.Now().Add(-6 * time.Minute),
	}
	cacheLock.Unlock()

	// Cache should be expired
	cacheLock.Lock()
	isExpired := cache == nil || time.Since(cache.timestamp) >= cacheExpirationTime
	cacheLock.Unlock()

	if !isExpired {
		t.Error("Cache should be expired at 6 minutes")
	}
}
