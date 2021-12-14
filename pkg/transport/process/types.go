// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"io"
)

// ResourceStreamProcessor describes an individual processor for processing a resource.
// A processor can upload, modify, or download a resource.
type ResourceStreamProcessor interface {
	// Process executes the processor for a resource. Input and Output streams must be
	// compliant to a specific format ("processor message"). See also ./utils/processor_message.go
	// which describes the format and provides helper functions to read/write processor messages.
	Process(context.Context, io.Reader, io.Writer) error
}
