package download

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/pkg/transport/pipeline"
	"github.com/gardener/component-cli/pkg/transport/util"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
)

type localOCIBlobDownloader struct {
	client ociclient.Client
}

func NewLocalOCIBlobDownloader(client ociclient.Client) pipeline.ResourceStreamProcessor {
	obj := localOCIBlobDownloader{
		client: client,
	}
	return &obj
}

func (d *localOCIBlobDownloader) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cd, res, _, err := util.ReadArchive(tar.NewReader(r))
	if err != nil {
		return fmt.Errorf("unable to read input archive: %w", err)
	}

	if res.Access.GetType() != cdv2.LocalOCIBlobType {
		return fmt.Errorf("unsupported access type: %+v", res.Access)
	}

	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		return fmt.Errorf("unable to create tempfile: %w", err)
	}
	defer tmpfile.Close()

	err = d.fetchLocalOCIBlob(ctx, cd, res, tmpfile)
	if err != nil {
		return fmt.Errorf("unable to fetch blob: %w", err)
	}

	_, err = tmpfile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("unable to seek to beginning of tempfile: %w", err)
	}

	err = util.WriteArchive(ctx, cd, res, tmpfile, tar.NewWriter(w))
	if err != nil {
		return fmt.Errorf("unable to write output archive: %w", err)
	}

	return nil
}

func (d *localOCIBlobDownloader) fetchLocalOCIBlob(ctx context.Context, cd *cdv2.ComponentDescriptor, res cdv2.Resource, w io.Writer) error {
	repoctx := cdv2.OCIRegistryRepository{}
	err := cd.GetEffectiveRepositoryContext().DecodeInto(&repoctx)
	if err != nil {
		return fmt.Errorf("unable to decode repository context: %w", err)
	}

	resolver := cdoci.NewResolver(d.client)
	_, blobResolver, err := resolver.ResolveWithBlobResolver(ctx, &repoctx, cd.Name, cd.Version)
	if err != nil {
		return fmt.Errorf("unable to resolve component descriptor: %w", err)
	}

	_, err = blobResolver.Resolve(ctx, res, w)
	if err != nil {
		return fmt.Errorf("unable to to resolve blob: %w", err)
	}

	return nil
}
