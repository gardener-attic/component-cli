package upload

import (
	"archive/tar"
	"context"
	"fmt"
	"io"

	"github.com/gardener/component-cli/pkg/transport/processor"
	"github.com/gardener/component-cli/pkg/transport/util"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

type localOCIBlobUploader struct {
	targetCtx cdv2.OCIRegistryRepository
}

func NewLocalOCIBlobUploader(targetCtx cdv2.OCIRegistryRepository) processor.ResourceStreamProcessor {
	obj := localOCIBlobUploader{
		targetCtx: targetCtx,
	}
	return &obj
}

func (d *localOCIBlobUploader) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cd, res, blobreader, err := util.ReadArchive(tar.NewReader(r))
	if err != nil {
		return fmt.Errorf("unable to read input archive: %w", err)
	}
	defer blobreader.Close()

	if res.Access.GetType() != cdv2.LocalOCIBlobType {
		return fmt.Errorf("unsupported access type: %+v", res.Access)
	}

	err = uploadLocalOCIBlob(ctx, cd, res, blobreader)
	if err != nil {
		return fmt.Errorf("unable to upload blob: %w", err)
	}

	// TODO: blobreader stream will be empty here. fix somehow (TeeReader/TmpFile/...)
	err = util.WriteArchive(ctx, cd, res, blobreader, tar.NewWriter(w))
	if err != nil {
		return fmt.Errorf("unable to write output archive: %w", err)
	}

	return nil
}

func uploadLocalOCIBlob(ctx context.Context, cd *cdv2.ComponentDescriptor, res cdv2.Resource, r io.Reader) error {
	// ociClient, err := ociclient.NewClient(
	// 	logr.Discard(),
	// )
	// if err != nil {
	// 	return fmt.Errorf("unable to create oci client: %w", err)
	// }

	return nil
}
