package pipeline

import (
	"context"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

type ResourceProcessingPipeline interface {
	Process(context.Context, *cdv2.ComponentDescriptor, cdv2.Resource) (*cdv2.ComponentDescriptor, cdv2.Resource, error)
}

type ResourceStreamProcessor interface {
	Process(context.Context, io.Reader, io.Writer) error
}
