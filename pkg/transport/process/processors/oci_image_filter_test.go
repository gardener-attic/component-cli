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

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/oci"
	"github.com/gardener/component-cli/pkg/testutils"
	"github.com/gardener/component-cli/pkg/transport/process/processors"
	processutils "github.com/gardener/component-cli/pkg/transport/process/utils"
)

var _ = Describe("ociImageFilter", func() {

	Context("Process", func() {

		It("should filter files from oci image", func() {
			expectedRes := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "v0.1.0",
					Type:    "ociImage",
				},
			}

			l1Files := map[string][]byte{
				"test": []byte("test-content"),
			}

			config := []byte("{}")

			layers := [][]byte{
				testutils.CreateTARArchive(l1Files).Bytes(),
			}

			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					Resources: []cdv2.Resource{
						expectedRes,
					},
				},
			}

			removePatterns := []string{
				"",
			}

			ociCache := cache.NewInMemoryCache()

			manifestData, manifestDesc := testutils.CreateManifest(config, layers, ociCache)

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
			Expect(processutils.WriteProcessorMessage(cd, expectedRes, r1, inBuf)).To(Succeed())

			outbuf := bytes.NewBuffer([]byte{})
			proc, err := processors.NewOCIImageFilter(ociCache, removePatterns)
			Expect(err).ToNot(HaveOccurred())
			Expect(proc.Process(context.TODO(), inBuf, outbuf)).To(Succeed())

			actualCD, actualRes, actualResBlobReader, err := processutils.ReadProcessorMessage(outbuf)
			Expect(err).ToNot(HaveOccurred())

			Expect(*actualCD).To(Equal(cd))
			Expect(actualRes).To(Equal(expectedRes))

			newCache := cache.NewInMemoryCache()
			actualOciArtifact, err := processutils.DeserializeOCIArtifact(actualResBlobReader, newCache)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualOciArtifact).To(Equal(ociArtifact))
		})

		It("should filter files from oci image index", func() {

		})

		It("should return error if cache is nil", func() {
			_, err := processors.NewOCIImageFilter(nil, []string{})
			Expect(err).To(MatchError("cache must not be nil"))
		})

	})
})
