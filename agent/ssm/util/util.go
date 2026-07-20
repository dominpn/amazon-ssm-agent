// Copyright 2016 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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

// Package util contains helper function common for ssm service
package util

import (
	cont "context"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sdk-go-v2/aws/retry"

	"github.com/aws/amazon-ssm-agent/agent/sdkutil/retryer"

	"github.com/aws/amazon-ssm-agent/agent/appconfig"
	"github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/agent/network"
	"github.com/aws/amazon-ssm-agent/common/identity/endpoint"
	"github.com/aws/aws-sdk-go-v2/aws"
	//"github.com/aws/smithy-go/logging"
)

func AwsConfig(logger log.T, appConfig appconfig.SsmagentConfig, service, region string) aws.Config {
	endpointHelper := endpoint.NewEndpointHelper(logger, appConfig)

	//cfg, err := config.LoadDefaultConfig(context.TODO())
	svcEndpoint := endpointHelper.GetServiceEndpoint(service, region)

	// Apply service-specific endpoint override from appConfig so that it is
	// baked into LoadOptions before the SDK caches ConfigSources. Otherwise a
	// post-hoc mutation of cfg.BaseEndpoint gets silently overwritten by
	// ssm.NewFromConfig's resolveBaseEndpoint, which re-reads LoadOptions.
	if service == "ssm" && appConfig.Ssm.Endpoint != "" {
		svcEndpoint = appConfig.Ssm.Endpoint
	}

	return AwsConfigWithEndpoint(logger, appConfig, service, region, &svcEndpoint)
}

func AwsConfigWithEndpoint(logger log.T, appConfig appconfig.SsmagentConfig, service, region string, endpoint *string) aws.Config {
	conf, _ := config.LoadDefaultConfig(cont.TODO(),
		config.WithRegion(region),
		config.WithBaseEndpoint(*endpoint),
		config.WithRetryer(newRetryer),
		config.WithHTTPClient(&http.Client{
			Transport:     network.GetDefaultTransport(logger, appConfig),
			CheckRedirect: network.DisableHTTPDowngrade,
			Timeout:       60 * time.Second,
		}),
	)
	return conf
}

var newRetryer = func() aws.Retryer {
	return &retryer.SsmRetryer{
		Standard: retry.NewStandard(func(so *retry.StandardOptions) {
			so.MaxAttempts = 4
		}),
	}
}

var sleepDelay = func(d time.Duration) {
	time.Sleep(d)
}
