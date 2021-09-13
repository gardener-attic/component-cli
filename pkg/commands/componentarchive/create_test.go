// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive_test

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/pkg/commands/componentarchive"
	"github.com/gardener/component-cli/pkg/utils"
)

var _ = Describe("Create", func() {

	var testdataFs vfs.FileSystem

	BeforeEach(func() {
		baseFs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
	})

	It("should create a component archive", func() {
		opts := &componentarchive.CreateOptions{}
		opts.Name = "example.com/component/name"
		opts.Version = "v0.0.1"

		fileName := "component-descriptor.yaml"

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		outputfileinfo, err := testdataFs.Stat(fileName)
		Expect(err).ToNot(HaveOccurred())
		Expect(outputfileinfo.IsDir()).To(BeFalse(), "output filepath should not be a directory")

		mediatype, err := utils.GetFileType(testdataFs, fileName)
		Expect(err).ToNot(HaveOccurred())
		Expect(mediatype).To(ContainSubstring("application/octet-stream"))

		fileCDbytes, err := vfs.ReadFile(testdataFs, fileName)
		Expect(err).ToNot(HaveOccurred())

		Expect(fileCDbytes).To(ContainSubstring(opts.Name))
		Expect(fileCDbytes).To(ContainSubstring(opts.Version))
	})

})
