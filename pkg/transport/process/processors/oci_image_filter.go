// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package processors

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/containerd/containerd/images"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/oci"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/serialize"
	"github.com/gardener/component-cli/pkg/utils"
)

type ociImageFilter struct {
	cache          cache.Cache
	removePatterns []string
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

			manifestBytes, err := json.Marshal(filteredManifest.Data)
			if err != nil {
				return fmt.Errorf("unable to marshal manifest: ")
			}

			if err := f.cache.Add(filteredManifest.Descriptor, io.NopCloser(bytes.NewReader(manifestBytes))); err != nil {
				return fmt.Errorf("unable to add filtered manifest to cache: %w", err)
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
			return fmt.Errorf("unable to filter image: %w", err)
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
	diffIDs := []digest.Digest{}
	digestMappings := map[digest.Digest]digest.Digest{}
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
		var layerBlobWriter io.WriteCloser = tmpfile

		isGzipCompressedLayer := layer.MediaType == ocispecv1.MediaTypeImageLayerGzip || layer.MediaType == images.MediaTypeDockerSchema2LayerGzip

		if isGzipCompressedLayer {
			layerBlobReader, err = gzip.NewReader(layerBlobReader)
			if err != nil {
				return nil, fmt.Errorf("unable to create gzip reader for layer: %w", err)
			}
			gzipw := gzip.NewWriter(layerBlobWriter)
			defer gzipw.Close()
			layerBlobWriter = gzipw
		}

		uncompressedHasher := sha256.New()
		mw := io.MultiWriter(layerBlobWriter, uncompressedHasher)

		if err = utils.FilterTARArchive(layerBlobReader, mw, f.removePatterns); err != nil {
			return nil, fmt.Errorf("unable to filter blob: %w", err)
		}

		if isGzipCompressedLayer {
			// close gzip writer (flushes any unwritten data and writes gzip footer)
			if err := layerBlobWriter.Close(); err != nil {
				return nil, fmt.Errorf("unable to close layer writer: %w", err)
			}
		}

		if _, err := tmpfile.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("unable to reset input file: %s", err)
		}

		filteredDigest, err := digest.FromReader(tmpfile)
		if err != nil {
			return nil, fmt.Errorf("unable to calculate digest for layer %+v: %w", layer, err)
		}

		digestMappings[layer.Digest] = filteredDigest
		diffIDs = append(diffIDs, digest.NewDigestFromEncoded(digest.SHA256, hex.EncodeToString(uncompressedHasher.Sum(nil))))

		fstat, err := tmpfile.Stat()
		if err != nil {
			return nil, fmt.Errorf("unable to get file stat: %w", err)
		}

		desc := ocispecv1.Descriptor{
			MediaType:   layer.MediaType,
			Digest:      filteredDigest,
			Size:        fstat.Size(),
			URLs:        layer.URLs,
			Platform:    layer.Platform,
			Annotations: layer.Annotations,
		}
		filteredLayers = append(filteredLayers, desc)

		if _, err := tmpfile.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("unable to reset input file: %s", err)
		}
		if err := f.cache.Add(desc, tmpfile); err != nil {
			return nil, fmt.Errorf("unable to add filtered layer blob to cache: %w", err)
		}
	}
	manifest.Data.Layers = filteredLayers

	cfgBlob, err := f.cache.Get(manifest.Data.Config)
	if err != nil {
		return nil, fmt.Errorf("unable to get config blob from cache: %w", err)
	}

	data, err := io.ReadAll(cfgBlob)
	if err != nil {
		return nil, fmt.Errorf("unable to read config blob: %w", err)
	}

	var config map[string]*json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unable to unmarshal config: %w", err)
	}

	rootfs := ocispecv1.RootFS{
		Type:    "layers",
		DiffIDs: diffIDs,
	}
	rootfsRaw, err := utils.RawJSON(rootfs)
	if err != nil {
		return nil, fmt.Errorf("unable to convert rootfs to JSON: %w", err)
	}
	config["rootfs"] = rootfsRaw

	marshaledConfig, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal config: %w", err)
	}

	configDesc := ocispecv1.Descriptor{
		MediaType: ocispecv1.MediaTypeImageConfig,
		Digest:    digest.FromBytes(marshaledConfig),
		Size:      int64(len(marshaledConfig)),
	}
	manifest.Data.Config = configDesc

	if err := f.cache.Add(configDesc, io.NopCloser(bytes.NewReader(marshaledConfig))); err != nil {
		return nil, fmt.Errorf("unable to add filtered layer blob to cache: %w", err)
	}

	manifestBytes, err := json.Marshal(manifest.Data)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal manifest: %w", err)
	}

	manifest.Descriptor.Size = int64(len(manifestBytes))
	manifest.Descriptor.Digest = digest.FromBytes(manifestBytes)

	return &manifest, nil
}

func NewOCIImageFilter(cache cache.Cache, removePatterns []string) process.ResourceStreamProcessor {
	obj := ociImageFilter{
		cache:          cache,
		removePatterns: removePatterns,
	}
	return &obj
}
