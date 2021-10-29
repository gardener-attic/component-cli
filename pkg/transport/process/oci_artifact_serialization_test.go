// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package process_test

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/oci"
	"github.com/gardener/component-cli/pkg/transport/process"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("oci artifact serialization", func() {

	Context("serialize and deserialize oci artifact", func() {

		It("should correctly serialize and deserialize image", func() {
			configData := []byte("config-data")
			layerData := []byte("layer-data")
			m, _, err := createManifest(configData, layerData)
			Expect(err).ToNot(HaveOccurred())

			expectedOciArtifact, err := oci.NewManifestArtifact(
				&oci.Manifest{
					Data: m,
				},
			)
			Expect(err).ToNot(HaveOccurred())

			serializeCache := cache.NewInMemoryCache()
			Expect(serializeCache.Add(m.Config, io.NopCloser(bytes.NewReader(configData)))).To(Succeed())
			Expect(serializeCache.Add(m.Layers[0], io.NopCloser(bytes.NewReader(layerData)))).To(Succeed())

			serializedReader, err := process.SerializeOCIArtifact(*expectedOciArtifact, serializeCache)
			Expect(err).ToNot(HaveOccurred())

			deserializeCache := cache.NewInMemoryCache()
			actualOciArtifact, err := process.DeserializeOCIArtifact(serializedReader, deserializeCache)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualOciArtifact.GetManifest().Data).To(Equal(expectedOciArtifact.GetManifest().Data))

			actualConfigReader, err := deserializeCache.Get(actualOciArtifact.GetManifest().Data.Config)
			Expect(err).ToNot(HaveOccurred())
			actualConfigBuf := bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualConfigBuf, actualConfigReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualConfigBuf.Bytes()).To(Equal(configData))

			actualLayerReader, err := deserializeCache.Get(actualOciArtifact.GetManifest().Data.Layers[0])
			Expect(err).ToNot(HaveOccurred())
			actualLayerBuf := bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualLayerBuf, actualLayerReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualLayerBuf.Bytes()).To(Equal(layerData))
		})

		It("should correctly serialize and deserialize image index", func() {
			configData1 := []byte("config-data-1")
			layerData1 := []byte("layer-data-1")
			configData2 := []byte("config-data-2")
			layerData2 := []byte("layer-data-2")

			m1, m1Desc, err := createManifest(configData1, layerData1)
			Expect(err).ToNot(HaveOccurred())
			m2, m2Desc, err := createManifest(configData2, layerData2)
			Expect(err).ToNot(HaveOccurred())

			m1Bytes, err := json.Marshal(m1)
			Expect(err).ToNot(HaveOccurred())

			m2Bytes, err := json.Marshal(m2)
			Expect(err).ToNot(HaveOccurred())

			expectedOciArtifact, err := oci.NewIndexArtifact(
				&oci.Index{
					Manifests: []*oci.Manifest{
						{
							Data: m1,
						},
						{
							Data: m2,
						},
					},
					Annotations: map[string]string{
						"testkey": "testval",
					},
				},
			)
			Expect(err).ToNot(HaveOccurred())

			serializeCache := cache.NewInMemoryCache()
			Expect(serializeCache.Add(m1Desc, io.NopCloser(bytes.NewReader(m1Bytes)))).To(Succeed())
			Expect(serializeCache.Add(m1.Config, io.NopCloser(bytes.NewReader(configData1)))).To(Succeed())
			Expect(serializeCache.Add(m1.Layers[0], io.NopCloser(bytes.NewReader(layerData1)))).To(Succeed())
			Expect(serializeCache.Add(m2Desc, io.NopCloser(bytes.NewReader(m2Bytes)))).To(Succeed())
			Expect(serializeCache.Add(m2.Config, io.NopCloser(bytes.NewReader(configData2)))).To(Succeed())
			Expect(serializeCache.Add(m2.Layers[0], io.NopCloser(bytes.NewReader(layerData2)))).To(Succeed())

			serializedReader, err := process.SerializeOCIArtifact(*expectedOciArtifact, serializeCache)
			Expect(err).ToNot(HaveOccurred())

			deserializeCache := cache.NewInMemoryCache()
			actualOciArtifact, err := process.DeserializeOCIArtifact(serializedReader, deserializeCache)
			Expect(err).ToNot(HaveOccurred())

			// check image index and manifests
			Expect(actualOciArtifact.GetIndex().Annotations).To(Equal(expectedOciArtifact.GetIndex().Annotations))
			Expect(actualOciArtifact.GetIndex().Manifests[0].Data).To(Equal(m1))
			Expect(actualOciArtifact.GetIndex().Manifests[1].Data).To(Equal(m2))

			// check first manifest config and layer
			actualConfigReader, err := deserializeCache.Get(actualOciArtifact.GetIndex().Manifests[0].Data.Config)
			Expect(err).ToNot(HaveOccurred())
			actualConfigBuf := bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualConfigBuf, actualConfigReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualConfigBuf.Bytes()).To(Equal(configData1))

			actualLayerReader, err := deserializeCache.Get(actualOciArtifact.GetIndex().Manifests[0].Data.Layers[0])
			Expect(err).ToNot(HaveOccurred())
			actualLayerBuf := bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualLayerBuf, actualLayerReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualLayerBuf.Bytes()).To(Equal(layerData1))

			// check second manifest config and layer
			actualConfigReader, err = deserializeCache.Get(actualOciArtifact.GetIndex().Manifests[1].Data.Config)
			Expect(err).ToNot(HaveOccurred())
			actualConfigBuf = bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualConfigBuf, actualConfigReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualConfigBuf.Bytes()).To(Equal(configData2))

			actualLayerReader, err = deserializeCache.Get(actualOciArtifact.GetIndex().Manifests[1].Data.Layers[0])
			Expect(err).ToNot(HaveOccurred())
			actualLayerBuf = bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualLayerBuf, actualLayerReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualLayerBuf.Bytes()).To(Equal(layerData2))

		})

	})

	Context("serialize oci artifact", func() {

		It("should raise error if ....", func() {

		})

	})

	Context("deserialize oci artifact", func() {

		It("should raise error if ....", func() {

		})

	})

})

func createManifest(configData []byte, layerData []byte) (*ocispecv1.Manifest, ocispecv1.Descriptor, error) {
	configDesc := ocispecv1.Descriptor{
		MediaType: "text/plain",
		Digest:    digest.FromBytes(configData),
		Size:      int64(len(configData)),
	}

	layerDesc := ocispecv1.Descriptor{
		MediaType: "text/plain",
		Digest:    digest.FromBytes(layerData),
		Size:      int64(len(layerData)),
	}

	m := ocispecv1.Manifest{
		Config: configDesc,
		Layers: []ocispecv1.Descriptor{
			layerDesc,
		},
	}

	mBytes, err := json.Marshal(m)
	if err != nil {
		return nil, ocispecv1.Descriptor{}, err
	}

	d := ocispecv1.Descriptor{
		Digest: digest.FromBytes(mBytes),
	}

	return &m, d, nil
}
