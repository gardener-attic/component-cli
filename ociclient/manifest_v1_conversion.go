// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/containerd/containerd/archive/compression"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/component-cli/pkg/utils"
)

// *************************************************************************************
// Docker Manifest v2 Schema 1 Support
// see also:
// - https://docs.docker.com/registry/spec/manifest-v2-1/
// - https://github.com/moby/moby/blob/master/image/v1/imagev1.go
// - https://github.com/containerd/containerd/blob/main/remotes/docker/schema1/converter.go
// *************************************************************************************

const (
	DockerV2Schema1MediaType       = "application/vnd.docker.distribution.manifest.v1+json"
	MediaTypeImageLayerZstd        = "application/vnd.oci.image.layer.v1.tar+zstd"
)

type fsLayer struct {
	BlobSum digest.Digest `json:"blobSum"`
}

type history struct {
	V1Compatibility string `json:"v1Compatibility"`
}

type v1Manifest struct {
	FSLayers []fsLayer `json:"fsLayers"`
	History  []history `json:"history"`
}

type v1History struct {
	Author          string    `json:"author,omitempty"`
	Created         time.Time `json:"created"`
	Comment         string    `json:"comment,omitempty"`
	ThrowAway       *bool     `json:"throwaway,omitempty"`
	Size            *int      `json:"Size,omitempty"`
	ContainerConfig struct {
		Cmd []string `json:"Cmd,omitempty"`
	} `json:"container_config,omitempty"`
}

func convertV1ManifestToV2(ctx context.Context, c *client, ref string, v1ManifestDesc ocispecv1.Descriptor) (ocispecv1.Descriptor, error) {
	buf := bytes.NewBuffer([]byte{})
	if err := c.Fetch(ctx, ref, v1ManifestDesc, buf); err != nil {
		return ocispecv1.Descriptor{}, fmt.Errorf("unable to fetch v1 manifest blob: %w", err)
	}

	var v1Manifest v1Manifest
	if err := json.Unmarshal(buf.Bytes(), &v1Manifest); err != nil {
		return ocispecv1.Descriptor{}, fmt.Errorf("unable to unmarshal v1 manifest: %w", err)
	}

	layers := []ocispecv1.Descriptor{}
	decompressedDigests := []digest.Digest{}
	history := []ocispecv1.History{}

	// layers in v1 are reversed compared to v2 --> iterate backwards
	for i := len(v1Manifest.FSLayers) - 1; i >= 0; i-- {
		var h v1History
		if err := json.Unmarshal([]byte(v1Manifest.History[i].V1Compatibility), &h); err != nil {
			return ocispecv1.Descriptor{}, fmt.Errorf("unable to unmarshal v1 history: %w", err)
		}

		emptyLayer := isEmptyLayer(&h)

		hs := ocispecv1.History{
			Author:     h.Author,
			Comment:    h.Comment,
			Created:    &h.Created,
			CreatedBy:  strings.Join(h.ContainerConfig.Cmd, " "),
			EmptyLayer: emptyLayer,
		}
		history = append(history, hs)

		if !emptyLayer {
			fslayer := v1Manifest.FSLayers[i]
			layerDesc := ocispecv1.Descriptor{
				Digest: fslayer.BlobSum,
				Size:   -1,
			}

			buf := bytes.NewBuffer([]byte{})
			if err := c.Fetch(ctx, ref, layerDesc, buf); err != nil {
				return ocispecv1.Descriptor{}, fmt.Errorf("unable to fetch layer blob: %w", err)
			}
			data := buf.Bytes()

			decompressedReader, err := compression.DecompressStream(bytes.NewReader(data))
			if err != nil {
				return ocispecv1.Descriptor{}, fmt.Errorf("unable to decompress layer blob: %w", err)
			}

			decompressedData, err := ioutil.ReadAll(decompressedReader)
			if err != nil {
				return ocispecv1.Descriptor{}, fmt.Errorf("unable to read decompressed layer blob: %w", err)
			}

			var mediatype string
			switch decompressedReader.GetCompression() {
			case compression.Uncompressed:
				mediatype = ocispecv1.MediaTypeImageLayer
			case compression.Gzip:
				mediatype = ocispecv1.MediaTypeImageLayerGzip
			case compression.Zstd:
				mediatype = MediaTypeImageLayerZstd
			}

			des := ocispecv1.Descriptor{
				Digest:    fslayer.BlobSum,
				MediaType: mediatype,
				Size:      int64(len(data)),
			}

			layers = append(layers, des)
			decompressedDigests = append(decompressedDigests, digest.FromBytes(decompressedData))
		}
	}

	configDesc, configBytes, err := createConfig(&v1Manifest, decompressedDigests, history)
	if err != nil {
		return ocispecv1.Descriptor{}, fmt.Errorf("unable to create config: %w", err)
	}

	v2ManifestDesc, v2ManifestBytes, err := createV2Manifest(configDesc, layers)
	if err != nil {
		return ocispecv1.Descriptor{}, fmt.Errorf("unable to create v2 manifest: %w", err)
	}

	err = c.cache.Add(configDesc, io.NopCloser(bytes.NewReader(configBytes)))
	if err != nil {
		return ocispecv1.Descriptor{}, fmt.Errorf("unable to write config blob to cache: %w", err)
	}

	err = c.cache.Add(v2ManifestDesc, io.NopCloser(bytes.NewReader(v2ManifestBytes)))
	if err != nil {
		return ocispecv1.Descriptor{}, fmt.Errorf("unable to write manifest blob to cache: %w", err)
	}

	return v2ManifestDesc, nil
}

func createV2Manifest(configDesc ocispecv1.Descriptor, layers []ocispecv1.Descriptor) (ocispecv1.Descriptor, []byte, error) {
	v2Manifest := ocispecv1.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Config: configDesc,
		Layers: layers,
	}

	marshaledV2Manifest, err := json.MarshalIndent(v2Manifest, "", "   ")
	if err != nil {
		return ocispecv1.Descriptor{}, nil, fmt.Errorf("unable to marshal manifest: %w", err)
	}

	v2ManifestDesc := ocispecv1.Descriptor{
		MediaType: ocispecv1.MediaTypeImageManifest,
		Digest:    digest.FromBytes(marshaledV2Manifest),
		Size:      int64(len(marshaledV2Manifest)),
	}

	return v2ManifestDesc, marshaledV2Manifest, nil
}

func createConfig(v1Manifest *v1Manifest, diffIDs []digest.Digest, history []ocispecv1.History) (ocispecv1.Descriptor, []byte, error) {
	var config map[string]*json.RawMessage
	if err := json.Unmarshal([]byte(v1Manifest.History[0].V1Compatibility), &config); err != nil {
		return ocispecv1.Descriptor{}, nil, fmt.Errorf("unable to unmarshal config from v1 history: %w", err)
	}

	delete(config, "id")
	delete(config, "parent")
	delete(config, "Size") // Size is calculated from data on disk and is inconsistent
	delete(config, "parent_id")
	delete(config, "layer_id")
	delete(config, "throwaway")

	rootfs := ocispecv1.RootFS{
		Type:    "layers",
		DiffIDs: diffIDs,
	}

	config["rootfs"] = utils.RawJSON(rootfs)
	config["history"] = utils.RawJSON(history)

	marshaledConfig, err := json.Marshal(config)
	if err != nil {
		return ocispecv1.Descriptor{}, nil, fmt.Errorf("unable to marshal config: %w", err)
	}

	configDesc := ocispecv1.Descriptor{
		MediaType: ocispecv1.MediaTypeImageConfig,
		Digest:    digest.FromBytes(marshaledConfig),
		Size:      int64(len(marshaledConfig)),
	}

	return configDesc, marshaledConfig, nil
}

// isEmptyLayer returns whether the v1 compatibility history describes an
// empty layer. A return value of true indicates the layer is empty,
// however false does not indicate non-empty.
func isEmptyLayer(h *v1History) bool {
	if h.ThrowAway != nil {
		return *h.ThrowAway
	}
	if h.Size != nil {
		return *h.Size == 0
	}

	// If no `Size` or `throwaway` field is given, then it cannot be determined whether the layer is empty
	// from the history, return false
	return false
}
