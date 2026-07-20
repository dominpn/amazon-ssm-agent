// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License").
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ec2roleprovider

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/amazon-ssm-agent/common/identity/credentialproviders"
	"github.com/aws/amazon-ssm-agent/common/identity/credentialproviders/ssmclient"

	"github.com/aws/aws-sdk-go-v2/credentials/ec2rolecreds"
)

const (
	agentName            = "amazon-ssm-agent"
	CredentialSourceNone = "None"
	CredentialSourceSSM  = "SSM"
	CredentialSourceEC2  = "EC2"
	IdentityTypeEC2      = "EC2"
)

var (
	iprEmptyCredential                                 = aws.Credentials{Source: "EC2RoleProvider"}
	newV4ServiceWithCreds        ssmclient.Initializer = ssmclient.NewV4ServiceWithCreds
	timeNowFunc                                        = time.Now
	newCredentials                                     = ec2rolecreds.New
	exceptionsForDefaultHostMgmt                       = map[string]struct{}{
		"AccessDeniedException":        {},
		"EC2RoleRequestError":          {},
		"AssumeRoleUnauthorizedAccess": {},
	}
)

type IInnerProvider interface {
	aws.CredentialsProvider
	Retrieve(ctx context.Context) (aws.Credentials, error)
	ExpiresAt() time.Time
	IsExpired() bool
}

type EC2InnerProviders struct {
	IPRProvider               IInnerProvider
	SsmEc2Provider            IInnerProvider
	SharedCredentialsProvider IInnerProvider
}

type IEC2RoleProvider interface {
	aws.CredentialsProvider
	credentialproviders.IRemoteProvider
	GetInnerProvider() IInnerProvider
	Retrieve(ctx context.Context) (aws.Credentials, error)
	ShareFile() string
	ShareProfile() string
	SharesCredentials() bool
	RemoteRetrieve(ctx context.Context) (aws.Credentials, error)
}
