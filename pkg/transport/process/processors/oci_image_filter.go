// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package processors

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/oci"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/serialize"
	"github.com/gardener/component-cli/pkg/utils"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type ociImageFilter struct {
	removePatterns []string
	cache          cache.Cache
}

func (f *ociImageFilter) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cd, res, blobreader, err := process.ReadProcessorMessage(r)
	if err != nil {
		return fmt.Errorf("unable to read archive: %w", err)
	}
	defer blobreader.Close()

	ociArtifact, err := serialize.DeserializeOCIArtifact(blobreader, f.cache)
	if err != nil {
		return fmt.Errorf("unable to deserialize oci artifact: %w", err)
	}

	if ociArtifact.IsIndex() {
		filteredImgs := []*oci.Manifest{}
		for _, m := range ociArtifact.GetIndex().Manifests {
			filteredManifest, err := f.filterImage(*m)
			if err != nil {
				return fmt.Errorf("unable to filter image %+v: %w", m, err)
			}
			filteredImgs = append(filteredImgs, filteredManifest)
		}
		filteredIndex := &oci.Index{
			Manifests:   filteredImgs,
			Annotations: ociArtifact.GetIndex().Annotations,
		}
		if err := ociArtifact.SetIndex(filteredIndex); err != nil {
			return fmt.Errorf("unable to set index: %w", err)
		}
	} else {
		filteredImg, err := f.filterImage(*ociArtifact.GetManifest())
		if err != nil {
			return fmt.Errorf("unable to filter image ")
		}
		if err := ociArtifact.SetManifest(filteredImg); err != nil {
			return fmt.Errorf("unable to set manifest: %w", err)
		}
	}

	blobReader, err := serialize.SerializeOCIArtifact(*ociArtifact, f.cache)
	if err != nil {
		return fmt.Errorf("unable to serialice oci artifact: %w", err)
	}

	if err = process.WriteProcessorMessage(*cd, res, blobReader, w); err != nil {
		return fmt.Errorf("unable to write archive: %w", err)
	}

	return nil
}

func (f *ociImageFilter) filterImage(manifest oci.Manifest) (*oci.Manifest, error) {
	filteredLayers := []ocispecv1.Descriptor{}
	for _, layer := range manifest.Data.Layers {
		layerBlobReader, err := f.cache.Get(layer)
		if err != nil {
			return nil, err
		}

		tmpfile, err := ioutil.TempFile("", "")
		if err != nil {
			return nil, fmt.Errorf("unable to create tempfile: %w", err)
		}
		defer tmpfile.Close()

		if layer.MediaType == ocispecv1.MediaTypeImageLayerGzip {
			layerBlobReader, err = gzip.NewReader(layerBlobReader)
			if err != nil {
				return nil, fmt.Errorf("unable to create gzip reader for layer: %w", err)
			}
		}

		if err = utils.FilterTARArchive(layerBlobReader, tar.NewWriter(tmpfile), f.removePatterns); err != nil {
			return nil, fmt.Errorf("unable to filter blob: %w", err)
		}

		blobDigest, err := digest.FromReader(tmpfile)
		if err != nil {
			return nil, fmt.Errorf("unable to calculate digest for layer %+v: %w", layer, err)
		}
		layer.Digest = blobDigest

		if _, err := tmpfile.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("unable to reset input file: %s", err)
		}
		if err := f.cache.Add(layer, tmpfile); err != nil {
			return nil, fmt.Errorf("unable to add filtered layer blob to cache: %w", err)
		}
	}
	manifest.Data.Layers = filteredLayers
	return &manifest, nil
}

func NewOCIImageFilter(removePatterns []string) process.ResourceStreamProcessor {
	obj := ociImageFilter{
		removePatterns: removePatterns,
	}
	return &obj
}
