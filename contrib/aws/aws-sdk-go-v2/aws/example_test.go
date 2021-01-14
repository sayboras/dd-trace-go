// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package aws_test

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	awstrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/aws/aws-sdk-go-v2/aws"
)

// To start tracing requests, wrap the AWS session.Session by invoking
// awstrace.WrapConfig.
func Example() {
	cfg, err := external.LoadDefaultAWSConfig(external.WithRegion("us-west-2"))
	if err != nil {
		fmt.Println("failed to load confi")
	}
	cfg = awstrace.WrapConfig(cfg)

	s3api := s3.New(cfg)
	_, _ = s3api.CreateBucketRequest(&s3.CreateBucketInput{
		Bucket: aws.String("some-bucket-name"),
	}).Send(context.TODO())
}
