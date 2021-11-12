// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package processors_test

import (
	"bytes"
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/oci"
	"github.com/gardener/component-cli/pkg/testutils"
	"github.com/gardener/component-cli/pkg/transport/process/processors"
	processutils "github.com/gardener/component-cli/pkg/transport/process/utils"
)

var _ = Describe("ociArtifactFilter", func() {

	Context("Process", func() {

		It("should filter files from oci image", func() {
			expectedRes := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "v0.1.0",
					Type:    "ociImage",
				},
			}
			expectedCd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					Resources: []cdv2.Resource{
						expectedRes,
					},
				},
			}

			removePatterns := []string{
				"filter-this/*",
			}

			l1Files := map[string][]byte{
				"test":              []byte("test-content"),
				"filter-this/file1": []byte("file1-content"),
				"filter-this/file2": []byte("file2-content"),
			}

			// TODO: add gzipped layer
			layers := [][]byte{
				testutils.CreateTARArchive(l1Files).Bytes(),
			}

			expectedL1Files := map[string][]byte{
				"test": []byte("test-content"),
			}

			expectedLayers := [][]byte{
				testutils.CreateTARArchive(expectedL1Files).Bytes(),
			}

			configData := []byte("{}")

			expectedManifestData, expectedManifestDesc := testutils.CreateManifest(configData, expectedLayers, nil)
			em := oci.Manifest{
				Descriptor: expectedManifestDesc,
				Data:       expectedManifestData,
			}
			expectedOciArtifact, err := oci.NewManifestArtifact(&em)
			Expect(err).ToNot(HaveOccurred())

			ociCache := cache.NewInMemoryCache()

			manifestData, manifestDesc := testutils.CreateManifest(configData, layers, ociCache)
			m := oci.Manifest{
				Descriptor: manifestDesc,
				Data:       manifestData,
			}

			ociArtifact, err := oci.NewManifestArtifact(&m)
			Expect(err).ToNot(HaveOccurred())

			r1, err := processutils.SerializeOCIArtifact(*ociArtifact, ociCache)
			Expect(err).ToNot(HaveOccurred())
			defer r1.Close()

			inBuf := bytes.NewBuffer([]byte{})
			Expect(processutils.WriteProcessorMessage(expectedCd, expectedRes, r1, inBuf)).To(Succeed())

			outbuf := bytes.NewBuffer([]byte{})
			proc, err := processors.NewOCIArtifactFilter(ociCache, removePatterns)
			Expect(err).ToNot(HaveOccurred())
			Expect(proc.Process(context.TODO(), inBuf, outbuf)).To(Succeed())

			actualCD, actualRes, actualResBlobReader, err := processutils.ReadProcessorMessage(outbuf)
			Expect(err).ToNot(HaveOccurred())

			Expect(*actualCD).To(Equal(expectedCd))
			Expect(actualRes).To(Equal(expectedRes))

			deserializeCache := cache.NewInMemoryCache()
			actualOciArtifact, err := processutils.DeserializeOCIArtifact(actualResBlobReader, deserializeCache)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualOciArtifact).To(Equal(expectedOciArtifact))

			r, err := deserializeCache.Get(actualOciArtifact.GetManifest().Data.Layers[0])
			Expect(err).ToNot(HaveOccurred())
			testutils.CheckTARArchive(r, expectedL1Files)
		})

		It("should filter files from all images of an oci image index", func() {
			expectedRes := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "v0.1.0",
					Type:    "ociImage",
				},
			}
			expectedCd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					Resources: []cdv2.Resource{
						expectedRes,
					},
				},
			}

			removePatterns := []string{
				"filter-this/*",
			}

			l1Files := map[string][]byte{
				"test":              []byte("test-content"),
				"filter-this/file1": []byte("file1-content"),
				"filter-this/file2": []byte("file2-content"),
			}

			// TODO: add gzipped layer
			layers := [][]byte{
				testutils.CreateTARArchive(l1Files).Bytes(),
			}

			expectedL1Files := map[string][]byte{
				"test": []byte("test-content"),
			}

			expectedLayers := [][]byte{
				testutils.CreateTARArchive(expectedL1Files).Bytes(),
			}

			configData := []byte("{}")

			expectedManifestData, expectedManifestDesc := testutils.CreateManifest(configData, expectedLayers, nil)
			ei := oci.Index{
				Manifests: []*oci.Manifest{
					{
						Descriptor: ocispecv1.Descriptor{
							MediaType: expectedManifestDesc.MediaType,
							Digest:    expectedManifestDesc.Digest,
							Size:      expectedManifestDesc.Size,
							Platform: &ocispecv1.Platform{
								Architecture: "amd64",
								OS:           "linux",
							},
						},
						Data: expectedManifestData,
					},
					{
						Descriptor: ocispecv1.Descriptor{
							MediaType: expectedManifestDesc.MediaType,
							Digest:    expectedManifestDesc.Digest,
							Size:      expectedManifestDesc.Size,
							Platform: &ocispecv1.Platform{
								Architecture: "amd64",
								OS:           "windows",
							},
						},
						Data: expectedManifestData,
					},
				},
				Annotations: map[string]string{
					"test": "test",
				},
			}
			expectedOciArtifact, err := oci.NewIndexArtifact(&ei)
			Expect(err).ToNot(HaveOccurred())

			ociCache := cache.NewInMemoryCache()

			manifestData, manifestDesc := testutils.CreateManifest(configData, layers, ociCache)

			index := oci.Index{
				Manifests: []*oci.Manifest{
					{
						Descriptor: ocispecv1.Descriptor{
							MediaType: manifestDesc.MediaType,
							Digest:    manifestDesc.Digest,
							Size:      manifestDesc.Size,
							Platform: &ocispecv1.Platform{
								Architecture: "amd64",
								OS:           "linux",
							},
						},
						Data: manifestData,
					},
					{
						Descriptor: ocispecv1.Descriptor{
							MediaType: manifestDesc.MediaType,
							Digest:    manifestDesc.Digest,
							Size:      manifestDesc.Size,
							Platform: &ocispecv1.Platform{
								Architecture: "amd64",
								OS:           "windows",
							},
						},
						Data: manifestData,
					},
				},
				Annotations: map[string]string{
					"test": "test",
				},
			}

			ociArtifact, err := oci.NewIndexArtifact(&index)
			Expect(err).ToNot(HaveOccurred())

			r1, err := processutils.SerializeOCIArtifact(*ociArtifact, ociCache)
			Expect(err).ToNot(HaveOccurred())
			defer r1.Close()

			inBuf := bytes.NewBuffer([]byte{})
			Expect(processutils.WriteProcessorMessage(expectedCd, expectedRes, r1, inBuf)).To(Succeed())

			outbuf := bytes.NewBuffer([]byte{})
			proc, err := processors.NewOCIArtifactFilter(ociCache, removePatterns)
			Expect(err).ToNot(HaveOccurred())
			Expect(proc.Process(context.TODO(), inBuf, outbuf)).To(Succeed())

			actualCD, actualRes, actualResBlobReader, err := processutils.ReadProcessorMessage(outbuf)
			Expect(err).ToNot(HaveOccurred())

			Expect(*actualCD).To(Equal(expectedCd))
			Expect(actualRes).To(Equal(expectedRes))

			deserializeCache := cache.NewInMemoryCache()
			actualOciArtifact, err := processutils.DeserializeOCIArtifact(actualResBlobReader, deserializeCache)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualOciArtifact).To(Equal(expectedOciArtifact))

			firstMan := actualOciArtifact.GetIndex().Manifests[0]
			fr, err := deserializeCache.Get(firstMan.Data.Layers[0])
			Expect(err).ToNot(HaveOccurred())
			testutils.CheckTARArchive(fr, expectedL1Files)

			secondMan := actualOciArtifact.GetIndex().Manifests[1]
			sr, err := deserializeCache.Get(secondMan.Data.Layers[0])
			Expect(err).ToNot(HaveOccurred())
			testutils.CheckTARArchive(sr, expectedL1Files)
		})

		It("should return error if cache is nil", func() {
			_, err := processors.NewOCIArtifactFilter(nil, []string{})
			Expect(err).To(MatchError("cache must not be nil"))
		})

	})
})
