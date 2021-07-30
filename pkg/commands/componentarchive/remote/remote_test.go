// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package remote_test

import (
	"bytes"
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

		Expect(showOpts.Run(ctx, logr.Discard(), testdataFs)).To(HaveOccurred())
	})

	It("should copy a component descriptor and its blobs from the source repository to the target repository.", func() {
		baseFs, err := projectionfs.New(osfs.New(), "../")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
		ctx := context.Background()

		cf, err := testenv.GetConfigFileBytes()
		Expect(err).ToNot(HaveOccurred())
		Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

		baseURLSource := testenv.Addr + "/test-source"
		baseURLTarget := testenv.Addr + "/test-target"

		pushOpts := &remote.PushOptions{
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		pushOpts.ComponentArchivePath = "./testdata/01-ca-blob"
		pushOpts.BaseUrl = baseURLSource

		Expect(pushOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		repos, err := client.ListRepositories(ctx, baseURLSource)
		Expect(err).ToNot(HaveOccurred())

		componentName := "example.com/component"
		componentVersion := "v0.0.0"

		sourceRef := baseURLSource + "/component-descriptors/" + componentName
		Expect(repos).To(ContainElement(Equal(sourceRef)))

		manifest, err := client.GetManifest(ctx, sourceRef+":"+componentVersion)
		Expect(err).ToNot(HaveOccurred())
		Expect(manifest.Layers).To(HaveLen(2))
		Expect(manifest.Layers[0].MediaType).To(Equal(cdoci.ComponentDescriptorTarMimeType),
			"Expect that the first layer contains the component descriptor")
		Expect(manifest.Layers[1].MediaType).To(Equal("text/plain"),
			"Expect that the second layer contains the local blob")

		var layerBlob bytes.Buffer
		Expect(client.Fetch(ctx, sourceRef+":"+componentVersion, manifest.Layers[1], &layerBlob)).To(Succeed())
		Expect(layerBlob.String()).To(Equal("blob test\n"))

		copyOpts := &remote.CopyOptions{
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		copyOpts.SourceRepository = baseURLSource
		copyOpts.ComponentName = componentName
		copyOpts.ComponentVersion = componentVersion
		copyOpts.TargetRepository = baseURLTarget
		//copyOpts.Force = true

		Expect(copyOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		repos, err = client.ListRepositories(ctx, baseURLTarget)
		Expect(err).ToNot(HaveOccurred())

		targetRef := baseURLTarget + "/component-descriptors/" + componentName
		Expect(repos).To(ContainElement(Equal(targetRef)))

		manifestTarget, err := client.GetManifest(ctx, targetRef+":"+componentVersion)
		Expect(err).ToNot(HaveOccurred())
		Expect(manifestTarget.Layers).To(HaveLen(2))
		Expect(manifestTarget.Layers[0].MediaType).To(Equal(cdoci.ComponentDescriptorTarMimeType),
			"Expect that the first layer contains the component descriptor")
		Expect(manifestTarget.Layers[1].MediaType).To(Equal("text/plain"),
			"Expect that the second layer contains the local blob")

		var layerBlobTarget bytes.Buffer
		Expect(client.Fetch(ctx, targetRef+":"+componentVersion, manifest.Layers[1], &layerBlobTarget)).To(Succeed())
		Expect(layerBlobTarget.String()).To(Equal("blob test\n"))
	})

})
