// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package downloaders_test

import (
	"bytes"
	"context"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/downloaders"
)

var _ = Describe("localOciBlob", func() {

	Context("Process", func() {

		It("should download and stream resource", func() {
			localOciBlobRes := testComponent.Resources[localOciBlobResIndex]

			inProcessorMsg := bytes.NewBuffer([]byte{})
			err := process.WriteProcessorMessage(testComponent, localOciBlobRes, nil, inProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			d, err := downloaders.NewLocalOCIBlobDownloader(client)
			Expect(err).ToNot(HaveOccurred())

			outProcessorMsg := bytes.NewBuffer([]byte{})
			err = d.Process(context.TODO(), inProcessorMsg, outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			actualCd, actualRes, resBlobReader, err := process.ReadProcessorMessage(outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())
			defer resBlobReader.Close()

			Expect(*actualCd).To(Equal(testComponent))
			Expect(actualRes).To(Equal(localOciBlobRes))

			resBlob := bytes.NewBuffer([]byte{})
			_, err = io.Copy(resBlob, resBlobReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(resBlob.Bytes()).To(Equal(localOciBlobResData))
		})

		It("should return error if called with resource of invalid type", func() {
			cd := cdv2.ComponentDescriptor{}

			access, err := cdv2.NewUnstructured(
				cdv2.NewOCIRegistryAccess("example-registry.com/test/image:1.0.0"),
			)
			Expect(err).ToNot(HaveOccurred())

			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "0.1.0",
					Type:    "helm",
				},
				Access: &access,
			}

			d, err := downloaders.NewLocalOCIBlobDownloader(client)
			Expect(err).ToNot(HaveOccurred())

			b1 := bytes.NewBuffer([]byte{})
			err = process.WriteProcessorMessage(cd, res, nil, b1)
			Expect(err).ToNot(HaveOccurred())

			b2 := bytes.NewBuffer([]byte{})
			err = d.Process(context.TODO(), b1, b2)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported access type"))
		})

	})

})
