package processors

import (
	"archive/tar"
	"context"
	"fmt"
	"io"

	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/utils"
)

type tarArchiveFileFilter struct {
	removePatterns []string
}

func (f *tarArchiveFileFilter) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cd, res, blobreader, err := process.ReadArchive(tar.NewReader(r))
	if err != nil {
		return fmt.Errorf("unable to read archive: %w", err)
	}

	if err = utils.FilterTARArchive(blobreader, tar.NewWriter(w), f.removePatterns); err != nil {
		return fmt.Errorf("unable to filter blob: %w", err)
	}

	if err = process.WriteArchive(ctx, cd, res, nil, tar.NewWriter(w)); err != nil {
		return fmt.Errorf("unable to write archive: %w", err)
	}

	return nil
}

func NewTarArchiveFileFilter(removePatterns []string) process.ResourceStreamProcessor {
	obj := tarArchiveFileFilter{
		removePatterns: removePatterns,
	}
	return &obj
}
