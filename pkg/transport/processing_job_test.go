// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package transport_test

import (
	"context"
	"encoding/json"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/pkg/transport"
	"github.com/gardener/component-cli/pkg/transport/process/processors"
)

var _ = Describe("processing job", func() {

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
			l3 := cdv2.Label{
				Name:  "processor-2",
				Value: json.RawMessage(`"true"`),
			}
			expectedRes := res
			expectedRes.Labels = append(expectedRes.Labels, l1)
			expectedRes.Labels = append(expectedRes.Labels, l2)
			expectedRes.Labels = append(expectedRes.Labels, l3)

			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					Resources: []cdv2.Resource{
						res,
					},
				},
			}

			p1 := transport.NamedResourceStreamProcessor{
				Name:      "p1",
				Processor: processors.NewResourceLabeler(l1),
			}
			p2 := transport.NamedResourceStreamProcessor{
				Name:      "p2",
				Processor: processors.NewResourceLabeler(l2),
			}
			p3 := transport.NamedResourceStreamProcessor{
				Name:      "p3",
				Processor: processors.NewResourceLabeler(l3),
			}

			pj, err := transport.NewProcessingJob(
				cd,
				res,
				[]transport.NamedResourceStreamProcessor{p1},
				[]transport.NamedResourceStreamProcessor{p2},
				[]transport.NamedResourceStreamProcessor{p3},
				logr.Discard(),
				10*time.Second,
			)
			Expect(err).ToNot(HaveOccurred())

			err = pj.Process(context.TODO())
			Expect(err).ToNot(HaveOccurred())
			Expect(*pj.GetProcessedResource()).To(Equal(expectedRes))
		})

	})

})
