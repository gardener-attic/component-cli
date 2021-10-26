// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package downloaders_test

import (
	"bytes"
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/downloaders"
	"github.com/gardener/component-cli/pkg/transport/process/serialize"
)

var _ = Describe("ociArtifact", func() {

	Context("Process", func() {

		It("should download and stream resource", func() {
			ociArtifactRes := testComponent.Resources[ociArtifactResIndex]

			inProcessorMsg := bytes.NewBuffer([]byte{})
			err := process.WriteProcessorMessage(testComponent, ociArtifactRes, nil, inProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			d, err := downloaders.NewOCIArtifactDownloader(ociClient, ociCache)
			Expect(err).ToNot(HaveOccurred())

			outProcessorMsg := bytes.NewBuffer([]byte{})
			err = d.Process(context.TODO(), inProcessorMsg, outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())

			actualCd, actualRes, resBlobReader, err := process.ReadProcessorMessage(outProcessorMsg)
			Expect(err).ToNot(HaveOccurred())
			defer resBlobReader.Close()

			Expect(*actualCd).To(Equal(testComponent))
			Expect(actualRes).To(Equal(ociArtifactRes))

			actualOciArtifact, err := serialize.DeserializeOCIArtifact(resBlobReader, ociCache)
			Expect(err).ToNot(HaveOccurred())
			Expect(*actualOciArtifact).To(Equal(expectedOciArtifact))
		})

		It("should return error if called with resource of invalid type", func() {
			localOciBlobRes := testComponent.Resources[localOciBlobResIndex]

			d, err := downloaders.NewOCIArtifactDownloader(ociClient, ociCache)
			Expect(err).ToNot(HaveOccurred())

			b1 := bytes.NewBuffer([]byte{})
			err = process.WriteProcessorMessage(testComponent, localOciBlobRes, nil, b1)
			Expect(err).ToNot(HaveOccurred())

			b2 := bytes.NewBuffer([]byte{})
			err = d.Process(context.TODO(), b1, b2)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported access type"))
		})

	})

})
