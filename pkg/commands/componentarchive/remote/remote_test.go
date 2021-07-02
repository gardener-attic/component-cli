// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package remote_test

import (
	"context"
	"os"

	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/ociclient/options"
	"github.com/gardener/component-cli/pkg/commands/componentarchive/remote"
)

var _ = Describe("Remote", func() {

	var testdataFs vfs.FileSystem

	BeforeEach(func() {
		baseFs, err := projectionfs.New(osfs.New(), "../")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
	})

	It("should push a component archive", func() {
		baseFs, err := projectionfs.New(osfs.New(), "../")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
		ctx := context.Background()

		cf, err := testenv.GetConfigFileBytes()
		Expect(err).ToNot(HaveOccurred())
		Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

		pushOpts := &remote.PushOptions{
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		pushOpts.ComponentArchivePath = "./testdata/00-ca"
		pushOpts.BaseUrl = testenv.Addr + "/test"

		res := remote.NewPushCommand(ctx)
		Expect(res)

		Expect(pushOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		repos, err := client.ListRepositories(ctx, testenv.Addr+"/test")
		Expect(err).ToNot(HaveOccurred())

		expectedRef := testenv.Addr + "/test/component-descriptors/example.com/component"
		Expect(repos).To(ContainElement(Equal(expectedRef)))

		manifest, err := client.GetManifest(ctx, expectedRef+":v0.0.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(manifest.Layers).To(HaveLen(1))
		Expect(manifest.Layers[0].MediaType).To(Equal(cdoci.ComponentDescriptorTarMimeType),
			"Expect that the first layer contains the component descriptor")
	})

	It("should get component archive", func() {
		baseFs, err := projectionfs.New(osfs.New(), "../")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
		ctx := context.Background()

		cf, err := testenv.GetConfigFileBytes()
		Expect(err).ToNot(HaveOccurred())
		Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

		pushOpts := &remote.PushOptions{
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		pushOpts.ComponentArchivePath = "./testdata/00-ca"
		pushOpts.BaseUrl = testenv.Addr + "/test"

		res := remote.NewPushCommand(ctx)
		Expect(res)

		Expect(pushOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		showOpts := &remote.ShowOptions{
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		showOpts.BaseUrl = testenv.Addr + "/test"
		showOpts.ComponentName = "example.com/component"
		showOpts.Version = "v0.0.0"

		res = remote.NewGetCommand(ctx)
		Expect(res)

		Expect(showOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())
	})

	It("should fail getting component archive which does not exist", func() {
		baseFs, err := projectionfs.New(osfs.New(), "../")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
		ctx := context.Background()

		cf, err := testenv.GetConfigFileBytes()
		Expect(err).ToNot(HaveOccurred())
		Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

		showOpts := &remote.ShowOptions{
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		showOpts.BaseUrl = testenv.Addr + "/test"
		showOpts.ComponentName = "example.com/component"
		showOpts.Version = "v6.6.6"

		res := remote.NewGetCommand(ctx)
		Expect(res)

		Expect(showOpts.Run(ctx, logr.Discard(), testdataFs)).To(HaveOccurred())
	})

})
