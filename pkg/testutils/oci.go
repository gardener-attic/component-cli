// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/oci"
)

func UploadTestManifest(ctx context.Context, client ociclient.Client, ref string) (*ocispecv1.Manifest, ocispecv1.Descriptor, error) {
	configData := []byte("test")
	layerData := []byte("layer-data")
	manifest := &ocispecv1.Manifest{
		Config: ocispecv1.Descriptor{
			MediaType: "text/plain",
			Digest:    digest.FromBytes(configData),
			Size:      int64(len(configData)),
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
			_, err := writer.Write(configData)
			return err
		default:
			_, err := writer.Write(layerData)
			return err
		}
	})

	if err := client.PushManifest(ctx, ref, manifest, ociclient.WithStore(store)); err != nil {
		return nil, ocispecv1.Descriptor{}, err
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, ocispecv1.Descriptor{}, err
	}

	desc := ocispecv1.Descriptor{
		MediaType: ocispecv1.MediaTypeImageManifest,
		Digest:    digest.FromBytes(manifestBytes),
		Size:      int64(len(manifestBytes)),
	}

	return manifest, desc, nil
}

func CompareManifestToTestManifest(ctx context.Context, client ociclient.Client, ref string, manifest *ocispecv1.Manifest) {
	var configBlob bytes.Buffer
	Expect(client.Fetch(ctx, ref, manifest.Config, &configBlob)).To(Succeed())
	Expect(configBlob.String()).To(Equal("test"))

	var layerBlob bytes.Buffer
	Expect(client.Fetch(ctx, ref, manifest.Layers[0], &layerBlob)).To(Succeed())
	Expect(layerBlob.String()).To(Equal("layer-data"))
}

func UploadTestIndex(ctx context.Context, client ociclient.Client, indexRef string) (*oci.Index, error) {
	splitted := strings.Split(indexRef, ":")
	indexRepo := strings.Join(splitted[0:len(splitted)-1], ":")
	tag := splitted[len(splitted)-1]

	manifest1Ref := fmt.Sprintf("%s-platform-1:%s", indexRepo, tag)
	manifest2Ref := fmt.Sprintf("%s-platform-2:%s", indexRepo, tag)

	manifest1, _, err := UploadTestManifest(ctx, client, manifest1Ref)
	if err != nil {
		return nil, err
	}

	manifest2, _, err := UploadTestManifest(ctx, client, manifest2Ref)
	if err != nil {
		return nil, err
	}

	index := oci.Index{
		Manifests: []*oci.Manifest{
			{
				Descriptor: ocispecv1.Descriptor{
					Platform: &ocispecv1.Platform{
						Architecture: "amd64",
						OS:           "linux",
					},
				},
				Data: manifest1,
			},
			{
				Descriptor: ocispecv1.Descriptor{
					Platform: &ocispecv1.Platform{
						Architecture: "amd64",
						OS:           "windows",
					},
				},
				Data: manifest2,
			},
		},
		Annotations: map[string]string{
			"test": "test",
		},
	}

	ociArtifact, err := oci.NewIndexArtifact(&index)
	if err != nil {
		return nil, err
	}

	if err := client.PushOCIArtifact(ctx, indexRef, ociArtifact); err != nil {
		return nil, err
	}

	return &index, nil
}

func CompareImageIndices(actualIndex *oci.Index, expectedIndex *oci.Index) {
	Expect(actualIndex.Annotations).To(Equal(expectedIndex.Annotations))
	Expect(len(actualIndex.Manifests)).To(Equal(len(expectedIndex.Manifests)))

	for i := 0; i < len(actualIndex.Manifests); i++ {
		actualManifest := actualIndex.Manifests[i]
		expectedManifest := expectedIndex.Manifests[i]

		expectedManifestBytes, err := json.Marshal(expectedManifest.Data)
		Expect(err).ToNot(HaveOccurred())

		Expect(actualManifest.Descriptor.MediaType).To(Equal(ocispecv1.MediaTypeImageManifest))
		Expect(actualManifest.Descriptor.Digest).To(Equal(digest.FromBytes(expectedManifestBytes)))
		Expect(actualManifest.Descriptor.Size).To(Equal(int64(len(expectedManifestBytes))))
		Expect(actualManifest.Descriptor.Platform).To(Equal(expectedManifest.Descriptor.Platform))
		Expect(actualManifest.Data).To(Equal(expectedManifest.Data))
	}
}
