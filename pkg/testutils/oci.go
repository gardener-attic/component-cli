// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/gardener/component-cli/ociclient"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func UploadTestManifest(ctx context.Context, client ociclient.Client, ref string) (*ocispecv1.Manifest, ocispecv1.Descriptor) {
	data := []byte("test")
	layerData := []byte("test-config")
	manifest := &ocispecv1.Manifest{
		Config: ocispecv1.Descriptor{
			MediaType: "text/plain",
			Digest:    digest.FromBytes(data),
			Size:      int64(len(data)),
		},
		Layers: []ocispecv1.Descriptor{
			{
				MediaType: "text/plain",
				Digest:    digest.FromBytes(layerData),
				Size:      int64(len(layerData)),
			},
		},
	}
	store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		switch desc.Digest.String() {
		case manifest.Config.Digest.String():
			_, err := writer.Write(data)
			return err
		default:
			_, err := writer.Write(layerData)
			return err
		}
	})
	Expect(client.PushManifest(ctx, ref, manifest, ociclient.WithStore(store))).To(Succeed())

	manifestBytes, err := json.Marshal(manifest)
	Expect(err).ToNot(HaveOccurred())

	desc := ocispecv1.Descriptor{
		MediaType: ocispecv1.MediaTypeImageManifest,
		Digest:    digest.FromBytes(manifestBytes),
		Size:      int64(len(manifestBytes)),
	}

	return manifest, desc
}

func CompareManifestToTestManifest(ctx context.Context, client ociclient.Client, ref string, manifest *ocispecv1.Manifest) {
	var configBlob bytes.Buffer
	Expect(client.Fetch(ctx, ref, manifest.Config, &configBlob)).To(Succeed())
	Expect(configBlob.String()).To(Equal("test"))

	var layerBlob bytes.Buffer
	Expect(client.Fetch(ctx, ref, manifest.Layers[0], &layerBlob)).To(Succeed())
	Expect(layerBlob.String()).To(Equal("test-config"))
}
