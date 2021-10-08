// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier
package downloaders

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/serialize"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type ociImageDownloader struct {
	client ociclient.Client
	cache  cache.Cache
}

func NewOCIImageDownloader(client ociclient.Client, cache cache.Cache) process.ResourceStreamProcessor {
	obj := ociImageDownloader{
		client: client,
		cache:  cache,
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

	ociArtifact, err := d.client.GetOCIArtifact(ctx, ociAccess.ImageReference)
	if err != nil {
		return fmt.Errorf("unable to get oci artifact: %w", err)
	}

	// fetch config blobs which adds them to the client cache
	if ociArtifact.IsManifest() {
		if err := d.fetchConfigAndLayerBlobs(ctx, ociAccess.ImageReference, ociArtifact.GetManifest().Data); err != nil {
			return err
		}
	} else if ociArtifact.IsIndex() {
		for _, m := range ociArtifact.GetIndex().Manifests {
			if err := d.fetchConfigAndLayerBlobs(ctx, ociAccess.ImageReference, m.Data); err != nil {
				return err
			}
		}
	}

	blobReader, err := serialize.SerializeOCIArtifact(*ociArtifact, d.cache)
	if err != nil {
		return fmt.Errorf("unable to serialize oci artifact: %w", err)
	}
	defer blobReader.Close()

	if err := process.WriteProcessorMessage(*cd, res, blobReader, w); err != nil {
		return fmt.Errorf("unable to write processor message: %w", err)
	}

	return nil
}

func (d *ociImageDownloader) fetchConfigAndLayerBlobs(ctx context.Context, ref string, manifest *ocispecv1.Manifest) error {
	buf := bytes.NewBuffer([]byte{})
	if err := d.client.Fetch(ctx, ref, manifest.Config, buf); err != nil {
		return fmt.Errorf("unable to fetch config blob: %w", err)
	}
	for _, l := range manifest.Layers {
		buf := bytes.NewBuffer([]byte{})
		if err := d.client.Fetch(ctx, ref, l, buf); err != nil {
			return fmt.Errorf("unable to fetch config blob: %w", err)
		}
	}
	return nil
}
