// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package process_test

import (
	"bytes"
	"io"
	"strings"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/pkg/transport/process"
)

var _ = Describe("util", func() {

	Context("WriteProcessorMessage & ReadProcessorMessage", func() {

		It("should correctly write and read a processor message", func() {
			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "v0.1.0",
					Type:    "ociImage",
				},
			}
			resourceData := "test-data"

			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					Resources: []cdv2.Resource{
						res,
					},
				},
			}

			processMsgBuf := bytes.NewBuffer([]byte{})
			err := process.WriteProcessorMessage(cd, res, strings.NewReader(resourceData), processMsgBuf)
			Expect(err).ToNot(HaveOccurred())

			actualCD, actualRes, resourceBlobReader, err := process.ReadProcessorMessage(processMsgBuf)
			Expect(err).ToNot(HaveOccurred())

			Expect(*actualCD).To(Equal(cd))
			Expect(actualRes).To(Equal(res))

			resourceBlobBuf := bytes.NewBuffer([]byte{})
			_, err = io.Copy(resourceBlobBuf, resourceBlobReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(resourceBlobBuf.String()).To(Equal(resourceData))
		})

	})
})
