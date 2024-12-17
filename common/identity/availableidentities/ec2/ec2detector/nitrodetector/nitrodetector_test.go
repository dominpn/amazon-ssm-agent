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

package nitrodetector

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/amazon-ssm-agent/agent/log"
	logger "github.com/aws/amazon-ssm-agent/agent/mocks/log"
	"github.com/aws/amazon-ssm-agent/common/identity/availableidentities/ec2/ec2detector/helper"
	"github.com/stretchr/testify/assert"
)

func TestIsEc2(t *testing.T) {
	detector := New("", "")
	logMock := logger.NewMockLog()
	tempMatchUuid := helper.MatchUuid
	tempGetSystemInfo := helper.GetSystemInfo
	defer func() {
		helper.MatchUuid = tempMatchUuid
		helper.GetSystemInfo = tempGetSystemInfo
	}()

	helper.GetSystemInfo = func(log.T, string) string { return "someothervendor" }
	assert.False(t, detector.IsEc2(logMock))

	helper.GetSystemInfo = func(log.T, string) string { return fmt.Sprintf("%s%s", expectedNitroVendor, "-somepostfix") }
	assert.False(t, detector.IsEc2(logMock))

	helper.GetSystemInfo = func(log.T, string) string { return fmt.Sprintf("%s%s", "someprefix-", expectedNitroVendor) }
	assert.False(t, detector.IsEc2(logMock))

	helper.GetSystemInfo = func(log.T, string) string { return expectedNitroVendor }
	helper.MatchUuid = func(log.T, string) bool { return false }
	assert.False(t, detector.IsEc2(logMock))

	helper.GetSystemInfo = func(log.T, string) string { return expectedNitroVendor }
	helper.MatchUuid = func(log.T, string) bool { return true }
	assert.True(t, detector.IsEc2(logMock))

	helper.GetSystemInfo = func(log.T, string) string { return strings.ToUpper(expectedNitroVendor) }
	helper.MatchUuid = func(log.T, string) bool { return true }
	assert.True(t, detector.IsEc2(logMock))
}
