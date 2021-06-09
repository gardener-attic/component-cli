// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package components_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/golang/mock/gomock"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	mock_ociclient "github.com/gardener/component-cli/ociclient/mock"
	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/gardener/component-cli/pkg/components"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Components Test Suite")
}

var _ = Describe("Components", func() {

	var (
		mockCtrl      *gomock.Controller
		mockOCIClient *mock_ociclient.MockClient
		testdatafs    vfs.FileSystem
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockOCIClient = mock_ociclient.NewMockClient(mockCtrl)

		fs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdatafs = layerfs.New(memoryfs.New(), fs)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("#ResolveInLocalCache", func() {
		It("should resolve a component from a local cache", func() {
			Expect(os.Setenv(constants.ComponentRepositoryCacheDirEnvVar, "./cache")).To(Succeed())
			repoCtx := cdv2.RepositoryContext{
				Type:    cdv2.OCIRegistryType,
				BaseURL: "eu.gcr.io/my-context/dev",
			}
			cd, err := components.ResolveInLocalCache(testdatafs, repoCtx, "github.com/gardener/component-cli", "v0.1.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(cd.Name).To(Equal("github.com/gardener/component-cli"))
			Expect(cd.Version).To(Equal("v0.1.0"))
		})
	})

	Context("#Resolver", func() {
		It("should resolve a component from a local cache", func() {
			Expect(os.Setenv(constants.ComponentRepositoryCacheDirEnvVar, "./cache")).To(Succeed())
			repoCtx := cdv2.RepositoryContext{
				Type:    cdv2.OCIRegistryType,
				BaseURL: "eu.gcr.io/my-context/dev",
			}

			cd, err := cdoci.NewResolver(mockOCIClient).
				WithCache(components.NewLocalComponentCache(testdatafs)).
				Resolve(context.TODO(), repoCtx, "github.com/gardener/component-cli", "v0.1.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(cd.Name).To(Equal("github.com/gardener/component-cli"))
			Expect(cd.Version).To(Equal("v0.1.0"))
		})

		It("should fallback to the oci client if a component is not in the local cache", func() {
			ctx := context.Background()
			defer ctx.Done()
			Expect(os.Setenv(constants.ComponentRepositoryCacheDirEnvVar, "./cache")).To(Succeed())
			repoCtx := cdv2.RepositoryContext{
				Type:    cdv2.OCIRegistryType,
				BaseURL: "eu.gcr.io/my-context/dev",
			}

			mockOCIClient.EXPECT().GetManifest(ctx, gomock.Any()).Times(1).Return(
				&ocispecv1.Manifest{
					Config: ocispecv1.Descriptor{
						MediaType: cdoci.ComponentDescriptorConfigMimeType,
						Digest:    digest.Digest("abc"),
					},
					Layers: []ocispecv1.Descriptor{
						{
							MediaType: cdoci.ComponentDescriptorJSONMimeType,
							Digest:    digest.Digest("efg"),
						},
					},
				}, nil)
			mockOCIClient.EXPECT().Fetch(ctx, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error {
					data, err := json.Marshal(cdoci.ComponentDescriptorConfig{
						ComponentDescriptorLayer: &cdoci.OciBlobRef{
							MediaType: cdoci.ComponentDescriptorJSONMimeType,
							Digest:    "efg",
						},
					})
					Expect(err).ToNot(HaveOccurred())
					_, err = io.Copy(writer, bytes.NewBuffer(data))
					Expect(err).ToNot(HaveOccurred())
					return nil
				})
			mockOCIClient.EXPECT().Fetch(ctx, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error {
					cd := &cdv2.ComponentDescriptor{
						Metadata: cdv2.Metadata{Version: "v2"},
						ComponentSpec: cdv2.ComponentSpec{
							ObjectMeta: cdv2.ObjectMeta{
								Name:    "github.com/gardener/component-cli",
								Version: "v0.2.0",
							},
							Provider:           cdv2.InternalProvider,
							RepositoryContexts: []cdv2.RepositoryContext{repoCtx},
						},
					}
					Expect(cdv2.DefaultComponent(cd)).To(Succeed())
					data, err := json.Marshal(cd)
					Expect(err).ToNot(HaveOccurred())
					_, err = io.Copy(writer, bytes.NewBuffer(data))
					Expect(err).ToNot(HaveOccurred())
					return nil
				})

			cd, err := cdoci.NewResolver(mockOCIClient).
				WithCache(components.NewLocalComponentCache(testdatafs)).
				Resolve(context.TODO(), repoCtx, "github.com/gardener/component-cli", "v0.2.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(cd.Name).To(Equal("github.com/gardener/component-cli"))
			Expect(cd.Version).To(Equal("v0.2.0"))
		})
	})

})
