// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config_test

import (
	"context"
	"encoding/json"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/pkg/transport/config"
	"github.com/gardener/component-cli/pkg/transport/process/processors"
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

	Context("processing job", func() {

		It("should correctly process resource", func() {
			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "v0.1.0",
					Type:    "ociImage",
				},
			}

			l1 := cdv2.Label{
				Name:  "processor-0",
				Value: json.RawMessage(`"true"`),
			}
			l2 := cdv2.Label{
				Name:  "processor-1",
				Value: json.RawMessage(`"true"`),
			}
			expectedRes := res
			expectedRes.Labels = append(expectedRes.Labels, l1)
			expectedRes.Labels = append(expectedRes.Labels, l2)

			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					Resources: []cdv2.Resource{
						res,
					},
				},
			}

			p1 := processors.NewResourceLabeler(l1)
			p2 := processors.NewResourceLabeler(l2)

			procs := []config.NamedResourceStreamProcessor{
				{
					Name:      "p1",
					Processor: p1,
				},
				{
					Name:      "p2",
					Processor: p2,
				},
			}

			pj := config.ProcessingJob{
				ComponentDescriptor: &cd,
				Resource:            &res,
				Processors:          procs,
				Log:                 logr.Discard(),
			}

			err := pj.Process(context.TODO())
			Expect(err).ToNot(HaveOccurred())
			Expect(*pj.ProcessedResource).To(Equal(expectedRes))
		})

	})

})
