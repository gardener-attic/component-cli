// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier
package serialize

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/oci"
	"github.com/gardener/component-cli/pkg/utils"
	"github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	ManifestFile = "manifest.json"
	BlobsDir     = "blobs"
)

func SerializeOCIArtifact(ociArtifact oci.Artifact, cache cache.Cache) (io.ReadCloser, error) {
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, fmt.Errorf("unable to create tempfile: %w", err)
	}

	if ociArtifact.IsIndex() {
		if err := serializeImageIndex(cache, ociArtifact.GetIndex(), tmpfile); err != nil {
			return nil, fmt.Errorf("unable to serialize image index: %w", err)
		}
	} else {
		if err := serializeImage(cache, ociArtifact.GetManifest(), ManifestFile, tar.NewWriter(tmpfile)); err != nil {
			return nil, fmt.Errorf("unable to serialize image: %w", err)
		}
	}

	if _, err := tmpfile.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("unable to seek to beginning of tempfile: %w", err)
	}

	return tmpfile, nil
}

func serializeImageIndex(cache cache.Cache, index *oci.Index, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	descs := []ocispecv1.Descriptor{}
	for _, m := range index.Manifests {
		manifestFile := path.Join(BlobsDir, m.Descriptor.Digest.Encoded())
		if err := serializeImage(cache, m, manifestFile, tw); err != nil {
			return fmt.Errorf("unable to serialize image: %w", err)
		}
		descs = append(descs, m.Descriptor)
	}

	i := ocispecv1.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Manifests:   descs,
		Annotations: index.Annotations,
	}

	indexBytes, err := json.Marshal(i)
	if err != nil {
		return fmt.Errorf("unable to marshal index manifest: %w", err)
	}

	if err := utils.WriteFileToTARArchive(ManifestFile, bytes.NewReader(indexBytes), tw); err != nil {
		return fmt.Errorf("unable to write index manifest: %w", err)
	}

	return nil
}

func serializeImage(cache cache.Cache, manifest *oci.Manifest, manifestFile string, tw *tar.Writer) error {
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("unable to marshal manifest: %w", err)
	}

	if err := utils.WriteFileToTARArchive(manifestFile, bytes.NewReader(manifestBytes), tw); err != nil {
		return fmt.Errorf("unable to write manifest: %w", err)
	}

	configReader, err := cache.Get(manifest.Data.Config)
	if err != nil {
		return fmt.Errorf("unable to get config blob from cache: %w", err)
	}
	defer configReader.Close()

	cfgFile := path.Join(BlobsDir, manifest.Data.Config.Digest.Encoded())
	if err := utils.WriteFileToTARArchive(cfgFile, configReader, tw); err != nil {
		return fmt.Errorf("unable to write config: %w", err)
	}

	for _, layer := range manifest.Data.Layers {
		layerReader, err := cache.Get(layer)
		if err != nil {
			return fmt.Errorf("unable to get layer blob from cache: %w", err)
		}
		defer layerReader.Close()

		layerFile := path.Join(BlobsDir, layer.Digest.Encoded())
		if err := utils.WriteFileToTARArchive(layerFile, layerReader, tw); err != nil {
			return fmt.Errorf("unable to write layer: %w", err)
		}
	}

	return nil
}

func DeserializeOCIArtifact(r io.Reader, cache cache.Cache) (*oci.Artifact, error) {
	// tr := tar.NewReader(r)

	// for {
	// 	header, err := tr.Next()
	// 	if err != nil {
	// 		if err == io.EOF {
	// 			break
	// 		}
	// 		return nil, fmt.Errorf("unable to read tar header: %w", err)
	// 	}

	// 	if header.Name == ManifestFile {

	// 	} else if strings.HasPrefix(header.Name, BlobsDir) {
	// 	} else {
	// 		return nil, fmt.Errorf()
	// 	}
	// }

	// if f == nil {
	// 	return cd, res, nil, nil
	// }

	// if _, err := f.Seek(0, io.SeekStart); err != nil {
	// 	return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to seek to beginning of file: %w", err)
	// }

	// return cd, res, f, nil

	// desc := ocispecv1.Descriptor{}

	// cache.Add(desc, layerReader)

	// ociArtifact := oci.Artifact{}

	return nil, nil
}
