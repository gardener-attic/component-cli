// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive_test

import (
	"context"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/pkg/commands/componentarchive"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf"
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
		opts.ComponentArchivePath = "./create-test"
		//opts.ComponentNameMapping = "urlPath"
		opts.BaseUrl = "example.com/testurl"

		err := testdataFs.Mkdir(opts.ComponentArchivePath, os.ModePerm)
		Expect(err).ToNot(HaveOccurred())

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())
		Expect(cd.Name).To(Equal(opts.Name), "component name should be the same")
		Expect(cd.Version).To(Equal(opts.Version), "component version should be the same")

		// Expect(cd.RepositoryContexts.Len() > 0).To(BeTrue(), "The repository contexts should return some data")
		repoCtx := cd.RepositoryContexts[0]
		Expect(repoCtx.GetType()).To(Equal(cdv2.OCIRegistryType), "check the repository context")

		// check overwrite
		opts.Version = "v0.0.2"
		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(HaveOccurred(), "Should not overwrite existing component")

		opts.Overwrite = true
		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed(), "Should overwrite existing component")

		data, err = vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd = &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())
		Expect(cd.Name).To(Equal(opts.Name), "component name should be the same")
		Expect(cd.Version).To(Equal(opts.Version), "component version should be the same")

		repoCtx = cd.RepositoryContexts[0]
		Expect(repoCtx.GetType()).To(Equal(cdv2.OCIRegistryType), "check the repository context")

	})

})
