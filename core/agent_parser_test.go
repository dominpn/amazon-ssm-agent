// Copyright 2026 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not
// use this file except in compliance with the License. A copy of the
// License is located at
//
// http://aws.amazon.com/apache2.0/

package main

import (
	"testing"

	"github.com/aws/amazon-ssm-agent/agent/appconfig"
	"github.com/aws/amazon-ssm-agent/agent/mocks/log"
	"github.com/aws/amazon-ssm-agent/common/identity"
	"github.com/stretchr/testify/assert"
)

func TestRegisterManagedInstance_FailsWhenProviderSetAndIMDSUnavailable(t *testing.T) {
	logMock := log.NewMockLog()
	provider = appconfig.EC2

	original := identity.IsIMDSAvailableForProvider
	identity.IsIMDSAvailableForProvider = func(p appconfig.Provider) (bool, error) { return false, nil }
	defer func() { identity.IsIMDSAvailableForProvider = original }()

	_, err := registerManagedInstance(logMock)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IMDS is not available on this host")
}

func TestRegisterManagedInstance_FailsWhenAzureProviderAndIMDSUnavailable(t *testing.T) {
	logMock := log.NewMockLog()
	provider = appconfig.Azure

	original := identity.IsIMDSAvailableForProvider
	identity.IsIMDSAvailableForProvider = func(p appconfig.Provider) (bool, error) { return false, nil }
	defer func() { identity.IsIMDSAvailableForProvider = original }()

	_, err := registerManagedInstance(logMock)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IMDS is not available on this host")
	assert.Contains(t, err.Error(), string(appconfig.Azure))
}

func TestRegisterManagedInstance_ProceedsWhenProviderSetAndIMDSAvailable(t *testing.T) {
	logMock := log.NewMockLog()
	provider = appconfig.EC2
	region = "us-east-1"

	original := identity.IsIMDSAvailableForProvider
	identity.IsIMDSAvailableForProvider = func(p appconfig.Provider) (bool, error) { return true, nil }
	defer func() { identity.IsIMDSAvailableForProvider = original }()

	_, err := registerManagedInstance(logMock)
	if err != nil {
		assert.NotContains(t, err.Error(), "IMDS is not available")
	}
}

func TestRegisterManagedInstance_SkipsIMDSCheckWhenNoProvider(t *testing.T) {
	logMock := log.NewMockLog()
	provider = ""
	region = "us-east-1"

	original := identity.IsIMDSAvailableForProvider
	identity.IsIMDSAvailableForProvider = func(p appconfig.Provider) (bool, error) { return false, nil }
	defer func() { identity.IsIMDSAvailableForProvider = original }()

	_, err := registerManagedInstance(logMock)
	if err != nil {
		assert.NotContains(t, err.Error(), "IMDS is not available")
	}
}
