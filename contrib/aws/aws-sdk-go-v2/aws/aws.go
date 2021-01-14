// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

// Package aws provides functions to trace aws/aws-sdk-go (https://github.com/aws/aws-sdk-go).
package aws // import "gopkg.in/DataDog/dd-trace-go.v1/contrib/aws/aws-sdk-go/aws"

import (
	"math"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const (
	tagAWSAgent      = "aws.agent"
	tagAWSOperation  = "aws.operation"
	tagAWSRegion     = "aws.region"
	tagAWSRetryCount = "aws.retry_count"
)

type handlers struct {
	cfg *config
}

// WrapConfig wraps a session.Session, causing requests and responses to be traced.
func WrapConfig(s aws.Config, opts ...Option) aws.Config {
	cfg := new(config)
	defaults(cfg)
	for _, opt := range opts {
		opt(cfg)
	}
	h := &handlers{cfg: cfg}
	temp := s.Copy()
	temp.Handlers.Send.PushFrontNamed(aws.NamedHandler{
		Name: "gopkg.in/DataDog/dd-trace-go.v1/contrib/aws/aws-sdk-go-v2/aws/handlers.Send",
		Fn:   h.Send,
	})
	temp.Handlers.Complete.PushBackNamed(aws.NamedHandler{
		Name: "gopkg.in/DataDog/dd-trace-go.v1/contrib/aws/aws-sdk-go-v2/aws/handlers.Complete",
		Fn:   h.Complete,
	})
	return temp
}

func (h *handlers) Send(req *aws.Request) {
	if req.RetryCount != 0 {
		return
	}
	opts := []ddtrace.StartSpanOption{
		tracer.SpanType(ext.SpanTypeHTTP),
		tracer.ServiceName(h.serviceName(req)),
		tracer.ResourceName(h.resourceName(req)),
		tracer.Tag(tagAWSAgent, h.awsAgent(req)),
		tracer.Tag(tagAWSOperation, h.awsOperation(req)),
		tracer.Tag(tagAWSRegion, h.awsRegion(req)),
		tracer.Tag(ext.HTTPMethod, req.Operation.HTTPMethod),
		tracer.Tag(ext.HTTPURL, req.HTTPRequest.URL.String()),
	}
	if !math.IsNaN(h.cfg.analyticsRate) {
		opts = append(opts, tracer.Tag(ext.EventSampleRate, h.cfg.analyticsRate))
	}
	_, ctx := tracer.StartSpanFromContext(req.Context(), h.operationName(req), opts...)
	req.SetContext(ctx)
}

func (h *handlers) Complete(req *aws.Request) {
	span, ok := tracer.SpanFromContext(req.Context())
	if !ok {
		return
	}
	span.SetTag(tagAWSRetryCount, req.RetryCount)
	if req.HTTPResponse != nil {
		span.SetTag(ext.HTTPCode, strconv.Itoa(req.HTTPResponse.StatusCode))
	}
	span.Finish(tracer.WithError(req.Error))
}

func (h *handlers) operationName(req *aws.Request) string {
	return h.awsService(req) + ".command"
}

func (h *handlers) resourceName(req *aws.Request) string {
	return h.awsService(req) + "." + req.Operation.Name
}

func (h *handlers) serviceName(req *aws.Request) string {
	if h.cfg.serviceName != "" {
		return h.cfg.serviceName
	}
	return "aws." + h.awsService(req)
}

func (h *handlers) awsAgent(req *aws.Request) string {
	if agent := req.HTTPRequest.Header.Get("User-Agent"); agent != "" {
		return agent
	}
	return "aws-sdk-go"
}

func (h *handlers) awsOperation(req *aws.Request) string {
	return req.Operation.Name
}

func (h *handlers) awsRegion(req *aws.Request) string {
	return req.Metadata.SigningRegion
}

func (h *handlers) awsService(req *aws.Request) string {
	return req.Metadata.ServiceName
}
