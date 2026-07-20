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

// Package sdkutil provides utilities used to call awssdk.
package sdkutil

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/aws/smithy-go"

	"github.com/aws/amazon-ssm-agent/agent/log"
)

// awserr represents an AWS error with code, message, and original error
type awserr struct {
	code    string
	message string
	origErr error
}

type Error interface {
	// Satisfy the generic error interface.
	error

	// Returns the short phrase depicting the classification of the error.
	Code() string

	// Returns the error details message.
	Message() string

	// Returns the original error if one was set.  Nil is returned if not set.
	OrigErr() error
}

// New creates a new awserr with the given code, message, and original error
func New(code, message string, origErr error) *awserr {
	return &awserr{
		code:    code,
		message: message,
		origErr: origErr,
	}
}

// Error returns the string representation of the error
func (e *awserr) Error() string {
	if e.origErr != nil {
		return fmt.Sprintf("%s: %s\ncaused by: %s", e.code, e.message, e.origErr.Error())
	}
	return fmt.Sprintf("%s: %s", e.code, e.message)
}

// Code returns the error code
func (e *awserr) Code() string {
	return e.code
}

// Message returns the error message
func (e *awserr) Message() string {
	return e.message
}

// OrigErr returns the original error
func (e *awserr) OrigErr() error {
	return e.origErr
}

type RequestFailure interface {
	Error

	// The status code of the HTTP response.
	StatusCode() int

	// The request ID returned by the service for a request failure. This will
	// be empty if no request ID is available such as the request failed due
	// to a connection error.
	RequestID() string
}

// HandleAwsError logs an AWS error.
func HandleAwsError(log log.T, err error, stopPolicy *StopPolicy) {
	if err != nil {
		// notice that we're using 1, so it will actually log the where
		// the error happened, 0 = this function, we don't want that.
		pc, fn, line, _ := runtime.Caller(1)
		log.Debugf("error in %s[%s:%d] %v", runtime.FuncForPC(pc).Name(), fn, line, err)

		// In case this is aws error, update the stop policy as well.
		// Use errors.As to unwrap *smithy.OperationError which wraps smithy.APIError.
		var aErr smithy.APIError
		if errors.As(err, &aErr) {
			// Generic AWS Error with Code, Message, and original error (if any)
			log.Debugf("AWS error. Code: %v, Message: %v, origerror: %v ", aErr.ErrorCode(), aErr.ErrorMessage(), aErr.Error())
			if aErr.ErrorCode() == "ExpiredTokenException" {
				log.Errorf("error when calling AWS APIs. error details - %v", err)
				log.Infof("ExpiredTokenException, maxing out stop policy to refresh service")
				stopPolicy.AddErrorCount(stopPolicy.MaximumErrorThreshold)
				return
			}

			// special treatment for Timeout exception - as this is expected
			if aErr.ErrorCode() == "RequestError" && aErr.Error() != "" && strings.Contains(aErr.ErrorMessage(), "Client.Timeout") {
				// resetting the error count to 0 - as these exceptions are all expected
				if stopPolicy != nil {
					resetStopPolicy(stopPolicy)
				}
				return
			}

			// special treatment for ec2messages AccessDeniedException - MDS endpoint is deprecated
			if aErr.ErrorCode() == "AccessDeniedException" && strings.Contains(err.Error(), "ec2messages:") {
				log.Debugf("ec2messages endpoint access denied. This is expected if ec2messages permissions are not configured. The ec2messages endpoint is deprecated; ssmmessages is the recommended endpoint.")
				if stopPolicy != nil {
					resetStopPolicy(stopPolicy)
				}
				return
			}
		}

		log.Errorf("error when calling AWS APIs. error details - %v", err)
		if stopPolicy != nil {
			log.Infof("increasing error count by 1")
			stopPolicy.AddErrorCount(1)
		}

	} else {
		// there is no error,
		resetStopPolicy(stopPolicy)
	}
}

// GetAwsErrorCode tries to return AwsError code
func GetAwsErrorCode(err error) string {
	errorCode := ""
	var apiErr smithy.APIError
	if ok := errors.As(err, &apiErr); ok {
		return apiErr.ErrorCode()
	}

	return errorCode
}

// GetAwsError tries to return AwsError
func GetAwsError(err error) *awserr {
	var awsErr *awserr
	if ok := errors.As(err, &awsErr); ok {
		return awsErr
	}
	return nil
}

// resetStopPolicy will reset the stoppolicy error count
func resetStopPolicy(stopPolicy *StopPolicy) {
	if stopPolicy != nil {
		stopPolicy.ResetErrorCount()
	}
}
