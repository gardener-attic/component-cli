// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package uploaders_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/oci"
	"github.com/gardener/component-cli/pkg/testutils"
	"github.com/gardener/component-cli/pkg/transport/process/uploaders"
	"github.com/gardener/component-cli/pkg/transport/process/utils"
)

var _ = Describe("ociArtifact", func() {

	Context("Process", func() {

		It("should upload and stream oci image", func() {
			acc, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess("my-registry.com/image:0.1.0"))
			Expect(err).ToNot(HaveOccurred())
			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "0.1.0",
					Type:    "plain-text",
				},
			}
			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "github.com/component-cli/test-component",
						Version: "0.1.0",
					},
					Resources: []cdv2.Resource{
						res,
					},
				},
			}
			res.Access = &acc
			expectedImageRef := targetCtx.BaseURL + "/image:0.1.0"
			configData := []byte("config-data")
			layers := [][]byte{
				[]byte("layer-data"),
			}
			m, _ := testutils.CreateManifest(configData, layers, nil)

			expectedOciArtifact, err := oci.NewManifestArtifact(
				&oci.Manifest{
					Data: m,
				},
			)
			Expect(err).ToNot(HaveOccurred())

			serializeCache := cache.NewInMemoryCache()
			Expect(serializeCache.Add(m.Config, io.NopCloser(bytes.NewReader(configData)))).To(Succeed())
			Expect(serializeCache.Add(m.Layers[0], io.NopCloser(bytes.NewReader(layers[0])))).To(Succeed())

			serializedReader, err := utils.SerializeOCIArtifact(*expectedOciArtifact, serializeCache)
			Expect(err).ToNot(HaveOccurred())

			inProcessorMsg := bytes.NewBuffer([]byte{})
			Expect(utils.WriteProcessorMessage(cd, res, serializedReader, inProcessorMsg)).To(Succeed())
			Expect(err).ToNot(HaveOccurred())

			d, err := uploaders.NewOCIArtifactUploader(ociClient, serializeCache, targetCtx.BaseURL, false)
			Expect(err).ToNot(HaveOccurred())

			outProcessorMsg := bytes.NewBuffer([]byte{})
			err = d.Process(context.TODO(), inProcessorMsg, outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			actualCd, actualRes, resBlobReader, err := utils.ReadProcessorMessage(outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())
			defer resBlobReader.Close()

			Expect(*actualCd).To(Equal(cd))
			Expect(actualRes.Name).To(Equal(res.Name))
			Expect(actualRes.Version).To(Equal(res.Version))
			Expect(actualRes.Type).To(Equal(res.Type))

			ociAcc := cdv2.OCIRegistryAccess{}
			Expect(actualRes.Access.DecodeInto(&ociAcc)).To(Succeed())
			Expect(ociAcc.ImageReference).To(Equal(expectedImageRef))

			actualOciArtifact, err := utils.DeserializeOCIArtifact(resBlobReader, cache.NewInMemoryCache())
			Expect(err).ToNot(HaveOccurred())
			Expect(actualOciArtifact.GetManifest().Data).To(Equal(m))

			buf := bytes.NewBuffer([]byte{})
			Expect(ociClient.Fetch(context.TODO(), expectedImageRef, actualOciArtifact.GetManifest().Descriptor, buf)).To(Succeed())
			am := ocispecv1.Manifest{}
			Expect(json.Unmarshal(buf.Bytes(), &am)).To(Succeed())
			Expect(am).To(Equal(*m))

			buf = bytes.NewBuffer([]byte{})
			Expect(ociClient.Fetch(context.TODO(), expectedImageRef, am.Config, buf)).To(Succeed())
			Expect(buf.Bytes()).To(Equal(configData))

			buf = bytes.NewBuffer([]byte{})
			Expect(ociClient.Fetch(context.TODO(), expectedImageRef, am.Layers[0], buf)).To(Succeed())
			Expect(buf.Bytes()).To(Equal(layers[0]))
		})

		It("should upload and stream oci image index", func() {

		})

		It("should return error for invalid access type", func() {
			acc, err := cdv2.NewUnstructured(cdv2.NewLocalOCIBlobAccess("sha256:123"))
			Expect(err).ToNot(HaveOccurred())
			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "0.1.0",
					Type:    "plain-text",
				},
				Access: &acc,
			}
			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "github.com/component-cli/test-component",
						Version: "0.1.0",
					},
					Resources: []cdv2.Resource{
						res,
					},
				},
			}

			u, err := uploaders.NewOCIArtifactUploader(ociClient, ociCache, targetCtx.BaseURL, false)
			Expect(err).ToNot(HaveOccurred())

			b1 := bytes.NewBuffer([]byte{})
			err = utils.WriteProcessorMessage(cd, res, bytes.NewReader([]byte("Hello World")), b1)
			Expect(err).ToNot(HaveOccurred())

			b2 := bytes.NewBuffer([]byte{})
			err = u.Process(context.TODO(), b1, b2)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported access type"))
		})

	})

})
