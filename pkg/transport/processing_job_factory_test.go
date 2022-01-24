// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package transport_test

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("processing job", func() {

	Context("processing job factory", func() {

		It("should create processing job", func() {
			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "github.com/my-component",
						Version: "0.1.0",
					},
				},
			}
			acc, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess("test.com"))
			Expect(err).ToNot(HaveOccurred())
			res := cdv2.Resource{
				Access: &acc,
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "0.1.0",
					Type:    cdv2.OCIImageType,
				},
			}

			job, err := factory.Create(cd, res)
			Expect(err).ToNot(HaveOccurred())

			Expect(*job.ComponentDescriptor).To(Equal(cd))
			Expect(*job.Resource).To(Equal(res))

			Expect(len(job.Downloaders)).To(Equal(1))
			Expect(job.Downloaders[0].Name).To(Equal("oci-artifact-dl"))

			Expect(len(job.Processors)).To(Equal(3))
			Expect(job.Processors[0].Name).To(Equal("my-oci-filter"))
			Expect(job.Processors[1].Name).To(Equal("my-labeler"))
			Expect(job.Processors[2].Name).To(Equal("my-extension"))

			Expect(len(job.Uploaders)).To(Equal(1))
			Expect(job.Uploaders[0].Name).To(Equal("oci-artifact-ul"))
		})

		It("should create processing job", func() {
			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "github.com/my-component",
						Version: "0.1.0",
					},
				},
			}
			acc, err := cdv2.NewUnstructured(cdv2.NewLocalOCIBlobAccess("sha256:123"))
			Expect(err).ToNot(HaveOccurred())
			res := cdv2.Resource{
				Access: &acc,
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "0.1.0",
					Type:    "helm",
				},
			}

			job, err := factory.Create(cd, res)
			Expect(err).ToNot(HaveOccurred())

			Expect(*job.ComponentDescriptor).To(Equal(cd))
			Expect(*job.Resource).To(Equal(res))

			Expect(len(job.Downloaders)).To(Equal(1))
			Expect(job.Downloaders[0].Name).To(Equal("local-oci-blob-dl"))

			Expect(len(job.Processors)).To(Equal(1))
			Expect(job.Processors[0].Name).To(Equal("my-labeler"))

			Expect(len(job.Uploaders)).To(Equal(1))
			Expect(job.Uploaders[0].Name).To(Equal("local-oci-blob-ul"))
		})

	})

})
