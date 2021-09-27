// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier
package serialize

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/oci"
	"github.com/gardener/component-cli/pkg/utils"
)

func SerializeOCIArtifact(ctx context.Context, client ociclient.Client, ref string) (io.ReadCloser, error) {
	ociArtifact, err := client.GetOCIArtifact(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("unable to get oci artifact: %w", err)
	}

	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, fmt.Errorf("unable to create tempfile: %w", err)
	}

	if ociArtifact.IsIndex() {
		if err := serializeImageIndex(ctx, client, ref, ociArtifact.GetIndex(), tmpfile); err != nil {
			return nil, fmt.Errorf("unable to serialize image index: %w", err)
		}
	} else {
		if err := serializeImage(ctx, client, ref, ociArtifact.GetManifest(), tar.NewWriter(tmpfile)); err != nil {
			return nil, fmt.Errorf("unable to serialize image: %w", err)
		}
	}

	if _, err := tmpfile.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("unable to seek to beginning of tempfile: %w", err)
	}

	return tmpfile, nil
}

func serializeImageIndex(ctx context.Context, client ociclient.Client, ref string, index *oci.Index, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	for _, m := range index.Manifests {
		if err := serializeImage(ctx, client, ref, m, tw); err != nil {
			return fmt.Errorf("unable to serialize image: %w", err)
		}
	}

	return nil
}

func serializeImage(ctx context.Context, client ociclient.Client, ref string, manifest *oci.Manifest, tw *tar.Writer) error {
	imageFilesPrefix := manifest.Descriptor.Digest.Encoded()

	manifestFile := path.Join(imageFilesPrefix, "manifest.json")
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("unable to marshal manifest: %w", err)
	}

	if err := utils.WriteFileToTARArchive(manifestFile, bytes.NewReader(manifestBytes), tw); err != nil {
		return fmt.Errorf("unable to write manifest: %w", err)
	}

	buf := bytes.NewBuffer([]byte{})
	if err := client.Fetch(ctx, ref, manifest.Data.Config, buf); err != nil {
		return fmt.Errorf("unable to fetch config blob: %w", err)
	}

	cfgFile := path.Join(imageFilesPrefix, "config.json")
	cfgBytes, err := json.Marshal(buf.Bytes())
	if err != nil {
		return fmt.Errorf("unable to marshal config: %w", err)
	}

	if err := utils.WriteFileToTARArchive(cfgFile, bytes.NewReader(cfgBytes), tw); err != nil {
		return fmt.Errorf("unable to write config: %w", err)
	}

	layerFilesPrefix := path.Join(imageFilesPrefix, "layers")
	for _, layer := range manifest.Data.Layers {
		tmpfile, err := ioutil.TempFile("", "")
		if err != nil {
			return fmt.Errorf("unable to create tempfile: %w", err)
		}
		defer tmpfile.Close()

		if err := client.Fetch(ctx, ref, layer, tmpfile); err != nil {
			return fmt.Errorf("unable to fetch layer blob: %w", err)
		}

		if _, err := tmpfile.Seek(0, 0); err != nil {
			return fmt.Errorf("unable to seek to beginning of tempfile: %w", err)
		}

		layerFile := path.Join(layerFilesPrefix, layer.Digest.Encoded())
		if err := utils.WriteFileToTARArchive(layerFile, tmpfile, tw); err != nil {
			return fmt.Errorf("unable to write layer: %w", err)
		}
	}

	return nil
}
