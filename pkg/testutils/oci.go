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
	"github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
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

	manifest1, mdesc1, err := UploadTestManifest(ctx, client, manifest1Ref)
	if err != nil {
		return nil, err
	}
	mdesc1.Platform = &ocispecv1.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}

	manifest2, mdesc2, err := UploadTestManifest(ctx, client, manifest2Ref)
	if err != nil {
		return nil, err
	}
	mdesc2.Platform = &ocispecv1.Platform{
		Architecture: "amd64",
		OS:           "windows",
	}

	index := oci.Index{
		Manifests: []*oci.Manifest{
			{
				Descriptor: mdesc1,
				Data:       manifest1,
			},
			{
				Descriptor: mdesc2,
				Data:       manifest2,
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

// CreateManifest creates an oci manifest. if ocicache is set, all blobs are added to cache
func CreateManifest(configData []byte, layersData [][]byte, ocicache cache.Cache) (*ocispecv1.Manifest, ocispecv1.Descriptor) {
	configDesc := ocispecv1.Descriptor{
		MediaType: "text/plain",
		Digest:    digest.FromBytes(configData),
		Size:      int64(len(configData)),
	}
	if ocicache != nil {
		Expect(ocicache.Add(configDesc, io.NopCloser(bytes.NewReader(configData)))).To(Succeed())
	}

	layerDescs := []ocispecv1.Descriptor{}
	for _, layerData := range layersData {
		layerDesc := ocispecv1.Descriptor{
			MediaType: "text/plain",
			Digest:    digest.FromBytes(layerData),
			Size:      int64(len(layerData)),
		}
		layerDescs = append(layerDescs, layerDesc)
		if ocicache != nil {
			Expect(ocicache.Add(layerDesc, io.NopCloser(bytes.NewReader(layerData)))).To(Succeed())
		}
	}

	manifest := ocispecv1.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Config: configDesc,
		Layers: layerDescs,
	}

	manifestBytes, err := json.Marshal(manifest)
	Expect(err).ToNot(HaveOccurred())

	manifestDesc := ocispecv1.Descriptor{
		MediaType: ocispecv1.MediaTypeImageManifest,
		Digest:    digest.FromBytes(manifestBytes),
		Size:      int64(len(manifestBytes)),
	}
	if ocicache != nil {
		Expect(ocicache.Add(manifestDesc, io.NopCloser(bytes.NewReader(manifestBytes)))).To(Succeed())
	}

	return &manifest, manifestDesc
}

func CompareRemoteManifest(client ociclient.Client, ref string, expectedManifest oci.Manifest, expectedCfgBytes []byte, expectedLayers [][]byte) {
	buf := bytes.NewBuffer([]byte{})
	Expect(client.Fetch(context.TODO(), ref, expectedManifest.Descriptor, buf)).To(Succeed())
	manifestFromRemote := ocispecv1.Manifest{}
	Expect(json.Unmarshal(buf.Bytes(), &manifestFromRemote)).To(Succeed())
	Expect(manifestFromRemote).To(Equal(*expectedManifest.Data))

	buf = bytes.NewBuffer([]byte{})
	Expect(client.Fetch(context.TODO(), ref, manifestFromRemote.Config, buf)).To(Succeed())
	Expect(buf.Bytes()).To(Equal(expectedCfgBytes))

	for i, layerDesc := range manifestFromRemote.Layers {
		buf = bytes.NewBuffer([]byte{})
		Expect(client.Fetch(context.TODO(), ref, layerDesc, buf)).To(Succeed())
		Expect(buf.Bytes()).To(Equal(expectedLayers[i]))
	}
}
