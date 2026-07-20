// Copyright 2021 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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

package credentialproviders

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go-v2/credentials/endpointcreds"
)

func GetRemoteCreds() aws.Credentials {
	appCreds := aws.NewCredentialsCache(ec2rolecreds.New())
	value, err := appCreds.Retrieve(context.TODO())
	if err != nil {
		// handle error
	}

	return value
}

func GetRemoteCredsProvider() aws.CredentialsProvider {
	cfg, _ := config.LoadDefaultConfig(context.TODO())
	provider := cfg.Credentials
	return provider
}

func GetDefaultCredsProvider() aws.CredentialsProvider {
	cfg, _ := config.LoadDefaultConfig(context.TODO())
	return cfg.Credentials
}

func GetECSCredsProvider() aws.CredentialsProvider {
	return endpointcreds.New("http://169.254.170.2" + os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI"))
}
