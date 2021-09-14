package process

import (
	"context"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// ResourceProcessingPipeline describes a chain of multiple processors for processing a resource.
// Each processor receives its input from the preceding processor and writes the output for the
// subsequent processor. To work correctly, a pipeline must consist of 1 downloader, 0..n processors,
// and 1..n uploaders.
type ResourceProcessingPipeline interface {
	// Process executes all processors for a resource.
	// Returns the component descriptor and resource of the last processor.
	Process(context.Context, cdv2.ComponentDescriptor, cdv2.Resource) (*cdv2.ComponentDescriptor, cdv2.Resource, error)
}

// ResourceStreamProcessor describes an individual processor for processing a resource.
// A processor can upload, modify, or download a resource.
type ResourceStreamProcessor interface {
	// Process executes the processor for a resource. Input and Output streams must be TAR 
	// archives which contain the component descriptor, resource, and resource blob.
	Process(context.Context, io.Reader, io.Writer) error
}
