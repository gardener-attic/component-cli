// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package extensions_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/extensions"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	defaultProcessorBinaryPath = "../../../../tmp/test/bin/processor"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "transport extensions Test Suite")
}

var _ = BeforeSuite(func() {
	info, err := os.Stat(defaultProcessorBinaryPath)
	Expect(err).ToNot(HaveOccurred())
	Expect(info.IsDir()).To(BeFalse())
}, 5)

var _ = Describe("transport extensions", func() {

	Context("stdio executable", func() {
		It("should modify the processed resource correctly", func() {
			args := []string{}
			env := []string{}
			processor, err := extensions.NewStdIOExecutable(context.TODO(), defaultProcessorBinaryPath, args, env)
			Expect(err).ToNot(HaveOccurred())

			testProcessor(processor)
		})
	})

	Context("uds executable", func() {
		It("should modify the processed resource correctly", func() {
			args := []string{}
			env := []string{}
			processor, err := extensions.NewUDSExecutable(context.TODO(), defaultProcessorBinaryPath, args, env)
			Expect(err).ToNot(HaveOccurred())

			testProcessor(processor)
		})
	})

})

func testProcessor(processor process.ResourceStreamProcessor) {
	const (
		processorName        = "test-processor"
		resourceData         = "12345"
		expectedResourceData = resourceData + "\n" + processorName
	)

	res := cdv2.Resource{
		IdentityObjectMeta: cdv2.IdentityObjectMeta{
			Name:    "my-res",
			Version: "v0.1.0",
			Type:    "ociImage",
		},
	}

	l := cdv2.Label{
		Name:  "processor-name",
		Value: json.RawMessage(`"` + processorName + `"`),
	}
	expectedRes := res
	expectedRes.Labels = append(expectedRes.Labels, l)

	cd := cdv2.ComponentDescriptor{
		ComponentSpec: cdv2.ComponentSpec{
			Resources: []cdv2.Resource{
				res,
			},
		},
	}

	inputBuf := bytes.NewBuffer([]byte{})
	err := process.WriteProcessorMessage(cd, res, strings.NewReader(resourceData), inputBuf)
	Expect(err).ToNot(HaveOccurred())

	outputBuf := bytes.NewBuffer([]byte{})
	err = processor.Process(context.TODO(), inputBuf, outputBuf)
	Expect(err).ToNot(HaveOccurred())

	processedCD, processedRes, processedBlobReader, err := process.ReadProcessorMessage(outputBuf)
	Expect(err).ToNot(HaveOccurred())

	Expect(*processedCD).To(Equal(cd))
	Expect(processedRes).To(Equal(expectedRes))

	processedResourceDataBuf := bytes.NewBuffer([]byte{})
	_, err = io.Copy(processedResourceDataBuf, processedBlobReader)
	Expect(err).ToNot(HaveOccurred())

	Expect(processedResourceDataBuf.String()).To(Equal(expectedResourceData))
}
