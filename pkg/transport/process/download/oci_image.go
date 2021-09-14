package download

import (
	"archive/tar"
	"context"
	"fmt"
	"io"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/oci"
	"github.com/gardener/component-cli/pkg/transport/process"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

type ociImageDownloader struct {
	client ociclient.Client
}

func NewOCIImageDownloader(client ociclient.Client) process.ResourceStreamProcessor {
	obj := ociImageDownloader{
		client: client,
	}
	return &obj
}

func (d *ociImageDownloader) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cd, res, _, err := process.ReadArchive(tar.NewReader(r))
	if err != nil {
		return fmt.Errorf("unable to read input archive: %w", err)
	}

	if res.Access.GetType() != cdv2.OCIRegistryType {
		return fmt.Errorf("unsupported acces type: %+v", res.Access)
	}

	if res.Type != cdv2.OCIImageType {
		return fmt.Errorf("unsupported resource type: %s", res.Type)
	}

	ociAccess := &cdv2.OCIRegistryAccess{}
	if err := res.Access.DecodeInto(ociAccess); err != nil {
		return fmt.Errorf("unable to decode resource access: %w", err)
	}

	ociArtifact, err := d.client.GetOCIArtifact(ctx, ociAccess.ImageReference)
	if err != nil {
		return fmt.Errorf("unable to get oci artifact: %w", err)
	}

	if ociArtifact.IsIndex() {
		handleImageIndex()
	} else {
		handleImage()
	}

	return nil
}

func handleImageIndex(index *oci.Index) {

	for _, m := range index.Manifests {

	}

	artifact.
}

func handleImage() {
	err := process.WriteArchive(ctx, cd, res, tmpfile, tar.NewWriter(w))
	if err != nil {
		return fmt.Errorf("unable to write output archive: %w", err)
	}
}
