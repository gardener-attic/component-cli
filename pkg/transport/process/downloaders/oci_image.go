// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier
package downloaders

import (
	"context"
	"fmt"
	"io"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/serialize"
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
	cd, res, _, err := process.ReadProcessorMessage(r)
	if err != nil {
		return fmt.Errorf("unable to read input archive: %w", err)
	}

	if res.Access.GetType() != cdv2.OCIRegistryType {
		return fmt.Errorf("unsupported access type: %+v", res.Access)
	}

	if res.Type != cdv2.OCIImageType {
		return fmt.Errorf("unsupported resource type: %s", res.Type)
	}

	ociAccess := &cdv2.OCIRegistryAccess{}
	if err := res.Access.DecodeInto(ociAccess); err != nil {
		return fmt.Errorf("unable to decode resource access: %w", err)
	}

	blobReader, err := serialize.SerializeOCIArtifact(ctx, d.client, ociAccess.ImageReference)
	if err != nil {
		return fmt.Errorf("unable to serialize oci artifact: %w", err)
	}
	defer blobReader.Close()

	if err := process.WriteProcessorMessage(*cd, res, blobReader, w); err != nil {
		return fmt.Errorf("unable to write processor message: %w", err)
	}

	return nil
}
