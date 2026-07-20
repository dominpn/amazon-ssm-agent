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

package config

import (
	"context"
	"strings"

	"github.com/aws/amazon-ssm-agent/agent/appconfig"
	"github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/common/identity"
	"github.com/aws/aws-sdk-go-v2/config"
)

func IsTelemetryEnabled(log log.T, identity identity.IAgentIdentity, appConfig appconfig.SsmagentConfig) bool {
	region, err := identity.Region()
	if err != nil {
		log.Warnf("Could not determine region")
		return false
	}

	isAwsPartition, err := isRegionInAwsPartition(region)
	if err != nil {
		log.Warnf("Could not determine partition for region %s: %v", region, err)
		return false
	}

	if !isAwsPartition {
		log.Debugf("Region %s is not in AWS standard partition", region)
		return false
	}

	if !appConfig.Agent.GlobalEnhancedTelemetryEnabled {
		log.Info("Agent GlobalEnhancedTelemetry is disabled")
		return false
	}

	return true
}

func isRegionInAwsPartition(region string) (bool, error) {
	// Try to load config with the region - this validates the region exists
	_, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return false, err
	}

	// Check if it's NOT in other partitions (China, GovCloud, ISO)
	if strings.HasPrefix(region, "cn-") ||
		strings.HasPrefix(region, "us-gov-") ||
		strings.HasPrefix(region, "us-iso-") ||
		strings.HasPrefix(region, "us-isob-") {
		return false, nil
	}

	return true, nil
}
