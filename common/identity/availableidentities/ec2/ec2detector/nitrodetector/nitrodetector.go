// Copyright 2022 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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

//go:build !darwin
// +build !darwin

// Package nitrodetector implements logic to determine if we are running on an nitro hypervisor
package nitrodetector

import (
	"strings"

	"github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/common/identity/availableidentities/ec2/ec2detector/helper"
)

const (
	expectedNitroVendor = "amazon ec2"
	Name                = "Nitro"
)

type nitroDetector struct {
	uuidParamKey   string
	vendorParamKey string
}

func (d *nitroDetector) IsEc2(log log.T) bool {
	if strings.ToLower(helper.GetSystemInfo(log, d.vendorParamKey)) != expectedNitroVendor {
		return false
	}

	return helper.MatchUuid(log, d.uuidParamKey)
}

func (d *nitroDetector) GetName() string {
	return Name
}

func New(uuidParamKey, vendorParamKey string) *nitroDetector {
	return &nitroDetector{
		uuidParamKey:   uuidParamKey,
		vendorParamKey: vendorParamKey,
	}
}
