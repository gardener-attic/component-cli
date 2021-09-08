package processor

import (
	"context"
	"io"
)

type ResourceStreamProcessor interface {
	Process(context.Context, io.Reader, io.Writer) error
}
