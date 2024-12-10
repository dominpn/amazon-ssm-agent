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

	"github.com/aws/amazon-ssm-agent/common/identity/availableidentities/ec2/ec2detector/helper"
	"github.com/stretchr/testify/assert"
)

func TestIsEc2(t *testing.T) {
	detector := New()
	tempGetVendor := getVendor
	tempGetUuid := getUuid
	defer func() {
		getUuid = tempGetUuid
		getVendor = tempGetVendor
	}()

	getUuid = func() string { return "someuuid" }
	getVendor = func() string { return "someothervendor" }
	assert.False(t, detector.IsEc2())

	getUuid = func() string { return "someuuid" }
	getVendor = func() string { return fmt.Sprintf("%s%s", expectedNitroVendor, "-somepostfix") }
	assert.False(t, detector.IsEc2())

	getUuid = func() string { return "someuuid" }
	getVendor = func() string { return fmt.Sprintf("%s%s", "someprefix-", expectedNitroVendor) }
	assert.False(t, detector.IsEc2())

	getUuid = func() string { return "someuuid" }
	getVendor = func() string { return expectedNitroVendor }
	helper.MatchUuid = func(string) bool { return false }
	assert.False(t, detector.IsEc2())

	getUuid = func() string { return "someuuid" }
	getVendor = func() string { return expectedNitroVendor }
	helper.MatchUuid = func(string) bool { return true }
	assert.True(t, detector.IsEc2())

	getUuid = func() string { return "someuuid" }
	getVendor = func() string { return strings.ToUpper(expectedNitroVendor) }
	helper.MatchUuid = func(string) bool { return true }
	assert.True(t, detector.IsEc2())
}
