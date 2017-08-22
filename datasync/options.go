// Copyright (c) 2017 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datasync

import (
	"github.com/ligato/cn-infra/logging"
	"time"
)

// PutOption defines options for Put operation. The particular options can be found below.
type PutOption interface {
}

// DelOption defines options for Del operation. The particular options can be found below.
type DelOption interface {
}

// WithTTLOpt defines a TTL for data being put. Once TTL elapses the data is removed from data store.
type WithTTLOpt struct {
	TTL time.Duration
}

// WithTTL creates new instance of TTL option. Once TTL elapses data is removed.
// Beware: some implementation might be using TTL with lower precision.
func WithTTL(TTL time.Duration) *WithTTLOpt {
	return &WithTTLOpt{TTL}
}

// WithPrefixOpt applies an operation to all items with the specified prefix.
type WithPrefixOpt struct {
}

// WithPrefix creates new instance of WithPrefixOpt.
func WithPrefix() *WithPrefixOpt {
	return &WithPrefixOpt{}
}

// WithTimeoutOpt defines the maximum time that is attempted to deliver notification.
type WithTimeoutOpt struct {
	Timeout time.Duration
}

// WithTimeout creates an option for ToChan function that defines a timeout for notification delivery.
func WithTimeout(timeout time.Duration) *WithTimeoutOpt {
	return &WithTimeoutOpt{timeout}
}

// WithLoggerOpt defines a logger that logs if delivery of notification is unsuccessful.
type WithLoggerOpt struct {
	Logger logging.Logger
}

// WithLogger creates an option for ToChan function that specifies a logger to be used.
func WithLogger(logger logging.Logger) *WithLoggerOpt {
	return &WithLoggerOpt{logger}
}
