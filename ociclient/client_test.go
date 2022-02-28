// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/gardener/component-cli/pkg/testutils"
)

func RunPushAndPullImageTest(repository, manifestMediaType string) {
	ctx := context.Background()
	defer ctx.Done()

	configData := []byte("config-data")
	layersData := [][]byte{
		[]byte("layer-1-data"),
		[]byte("layer-2-data"),
	}
	ref := fmt.Sprintf("%s/%s", testenv.Addr, repository)

	manifestDesc, manifestBytes := testutils.UploadTestImage(ctx, client, ref, manifestMediaType, configData, layersData)

	testutils.CompareRemoteManifest(ctx, client, ref, manifestDesc, manifestBytes, configData, layersData)
}

func RunPushAndPullImageIndexTest(repository, indexMediaType string) {
	ctx := context.Background()
	defer ctx.Done()

	configData := []byte("config-data")
	layersData := [][]byte{
		[]byte("layer-1-data"),
		[]byte("layer-2-data"),
	}
	ref := fmt.Sprintf("%s/%s", testenv.Addr, repository)
	ref1 := ref + "-platform-1"
	ref2 := ref + "-platform-2"

	manifest1Desc, manifest1Bytes := testutils.UploadTestImage(ctx, client, ref1, ocispecv1.MediaTypeImageManifest, configData, layersData)
	manifest1IndexDesc := manifest1Desc
	manifest1IndexDesc.Platform = &ocispecv1.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}

	manifest2Desc, manifest2Bytes := testutils.UploadTestImage(ctx, client, ref2, ocispecv1.MediaTypeImageManifest, configData, layersData)
	manifest2IndexDesc := manifest2Desc
	manifest2IndexDesc.Platform = &ocispecv1.Platform{
		Architecture: "amd64",
		OS:           "windows",
	}

	index := ocispecv1.Index{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Manifests: []ocispecv1.Descriptor{
			manifest1IndexDesc,
			manifest2IndexDesc,
		},
		Annotations: map[string]string{
			"test": "test",
		},
	}

	indexDesc, indexBytes := testutils.UploadTestIndex(ctx, client, ref, indexMediaType, index)

	actualIndexDesc, actualIndexBytes, err := client.GetRawManifest(ctx, ref)
	Expect(err).ToNot(HaveOccurred())
	Expect(actualIndexDesc).To(Equal(indexDesc))
	Expect(actualIndexBytes).To(Equal(indexBytes))

	testutils.CompareRemoteManifest(ctx, client, ref1, manifest1Desc, manifest1Bytes, configData, layersData)
	testutils.CompareRemoteManifest(ctx, client, ref2, manifest2Desc, manifest2Bytes, configData, layersData)
}

var _ = Describe("client", func() {

	Context("Client", func() {

		It("should push and pull a single architecture image without modifications (oci media type)", func() {
			RunPushAndPullImageTest("single-arch-tests/0/artifact:0.0.1", ocispecv1.MediaTypeImageManifest)
		}, 20)

		It("should push and pull a multi architecture image without modifications (oci media type)", func() {
			RunPushAndPullImageIndexTest("multi-arch-tests/0/artifact:0.0.1", ocispecv1.MediaTypeImageIndex)
		}, 20)

		// TODO: investigate why this test isn't working (could be registry not accepting docker media type)
		// It("should push and pull a single architecture image without modifications (docker media type)", func() {
		// 	RunPushAndPullTest("single-arch-tests/1/artifact:0.0.1", images.MediaTypeDockerSchema2Manifest)
		// }, 20)

		// TODO: investigate why this test isn't working (could be registry not accepting docker media type)
		// It("should push and pull a multi architecture image without modifications (docker media type)", func() {
		// 	RunPushAndPullImageIndexTest("multi-arch-tests/1/artifact:0.0.1", images.MediaTypeDockerSchema2ManifestList)
		// }, 20)

		It("should push and pull an empty oci image index", func() {
			ctx := context.Background()
			defer ctx.Done()

			ref := testenv.Addr + "/multi-arch-tests/3/empty-img:v0.0.1"
			index := ocispecv1.Index{
				Versioned: specs.Versioned{
					SchemaVersion: 2,
				},
				Manifests: []ocispecv1.Descriptor{},
				Annotations: map[string]string{
					"test": "test",
				},
			}

			indexBytes, err := json.Marshal(index)
			Expect(err).ToNot(HaveOccurred())

			indexDesc := ocispecv1.Descriptor{
				MediaType: ocispecv1.MediaTypeImageIndex,
				Digest:    digest.FromBytes(indexBytes),
				Size:      int64(len(indexBytes)),
			}

			store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
				_, err := writer.Write(indexBytes)
				return err
			})

			Expect(client.PushRawManifest(ctx, ref, indexDesc, indexBytes, ociclient.WithStore(store))).To(Succeed())

			actualIndexDesc, actualIndexBytes, err := client.GetRawManifest(ctx, ref)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualIndexDesc).To(Equal(indexDesc))
			Expect(actualIndexBytes).To(Equal(indexBytes))
		}, 20)

		It("should push and pull an oci image index with only 1 manifest and no platform information", func() {
			ctx := context.Background()
			defer ctx.Done()

			configData := []byte("config-data")
			layersData := [][]byte{
				[]byte("layer-1-data"),
				[]byte("layer-2-data"),
			}
			ref := testenv.Addr + "/multi-arch-tests/4/img:v0.0.1"

			mdesc, _ := testutils.UploadTestImage(ctx, client, ref, ocispecv1.MediaTypeImageManifest, configData, layersData)

			index := ocispecv1.Index{
				Versioned: specs.Versioned{
					SchemaVersion: 2,
				},
				Manifests: []ocispecv1.Descriptor{
					mdesc,
				},
				Annotations: map[string]string{
					"test": "test",
				},
			}

			indexBytes, err := json.Marshal(index)
			Expect(err).ToNot(HaveOccurred())

			indexDesc := ocispecv1.Descriptor{
				MediaType: ocispecv1.MediaTypeImageIndex,
				Digest:    digest.FromBytes(indexBytes),
				Size:      int64(len(indexBytes)),
			}

			store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
				_, err := writer.Write(indexBytes)
				return err
			})

			Expect(client.PushRawManifest(ctx, ref, indexDesc, indexBytes, ociclient.WithStore(store))).To(Succeed())

			actualIndexDesc, actualIndexBytes, err := client.GetRawManifest(ctx, ref)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualIndexDesc).To(Equal(indexDesc))
			Expect(actualIndexBytes).To(Equal(indexBytes))
		}, 20)

		It("should copy an oci artifact", func() {
			ctx := context.Background()
			defer ctx.Done()

			configData := []byte("config-data")
			layersData := [][]byte{
				[]byte("layer-1-data"),
				[]byte("layer-2-data"),
			}
			ref := testenv.Addr + "/test/artifact:v0.0.1"
			mdesc, mbytes := testutils.UploadTestImage(ctx, client, ref, ocispecv1.MediaTypeImageManifest, configData, layersData)
			newRef := testenv.Addr + "/new/artifact:v0.0.1"

			Expect(ociclient.Copy(ctx, client, ref, newRef)).To(Succeed())

			testutils.CompareRemoteManifest(ctx, client, newRef, mdesc, mbytes, configData, layersData)
		}, 20)

		It("should copy an oci image index", func() {
			ctx := context.Background()
			defer ctx.Done()

			configData := []byte("config-data")
			layersData := [][]byte{
				[]byte("layer-1-data"),
				[]byte("layer-2-data"),
			}
			ref1 := testenv.Addr + "/copy/image-index/src/img:v0.0.1-platform-1"
			ref2 := testenv.Addr + "/copy/image-index/src/img:v0.0.1-platform-2"
			ref := testenv.Addr + "/copy/image-index/src/img:v0.0.1"

			manifest1Desc, manifest1Bytes := testutils.UploadTestImage(ctx, client, ref1, ocispecv1.MediaTypeImageManifest, configData, layersData)
			manifest1IndexDesc := manifest1Desc
			manifest1IndexDesc.Platform = &ocispecv1.Platform{
				Architecture: "amd64",
				OS:           "linux",
			}

			manifest2Desc, manifest2Bytes := testutils.UploadTestImage(ctx, client, ref2, ocispecv1.MediaTypeImageManifest, configData, layersData)
			manifest2IndexDesc := manifest2Desc
			manifest2IndexDesc.Platform = &ocispecv1.Platform{
				Architecture: "amd64",
				OS:           "windows",
			}

			index := ocispecv1.Index{
				Versioned: specs.Versioned{SchemaVersion: 2},
				Manifests: []ocispecv1.Descriptor{
					manifest1IndexDesc,
					manifest2IndexDesc,
				},
				Annotations: map[string]string{
					"test": "test",
				},
			}

			indexDesc, indexBytes := testutils.UploadTestIndex(ctx, client, ref, ocispecv1.MediaTypeImageIndex, index)

			newRef := testenv.Addr + "/copy/image-index/tgt/img:v0.0.1"
			Expect(ociclient.Copy(ctx, client, ref, newRef)).To(Succeed())

			actualIndexDesc, actualIndexBytes, err := client.GetRawManifest(ctx, ref)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualIndexDesc).To(Equal(indexDesc))
			Expect(actualIndexBytes).To(Equal(indexBytes))

			testutils.CompareRemoteManifest(ctx, client, ref1, manifest1Desc, manifest1Bytes, configData, layersData)
			testutils.CompareRemoteManifest(ctx, client, ref2, manifest2Desc, manifest2Bytes, configData, layersData)
		}, 20)

	})

	Context("ExtendedClient", func() {
		Context("ListTags", func() {
			var (
				server  *httptest.Server
				host    string
				handler func(http.ResponseWriter, *http.Request)
				makeRef = func(repo string) string {
					return fmt.Sprintf("%s/%s", host, repo)
				}
			)

			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					handler(writer, request)
				}))

				hostUrl, err := url.Parse(server.URL)
				Expect(err).ToNot(HaveOccurred())
				host = hostUrl.Host
			})

			AfterEach(func() {
				server.Close()
			})

			It("should return a list of tags", func() {
				var (
					ctx        = context.Background()
					repository = "myproject/repo/myimage"
				)
				defer ctx.Done()
				handler = func(w http.ResponseWriter, req *http.Request) {
					if req.URL.Path == "/v2/" {
						// first auth discovery call by the library
						w.WriteHeader(200)
						return
					}
					Expect(req.URL.String()).To(Equal("/v2/myproject/repo/myimage/tags/list?n=1000"))
					w.WriteHeader(200)
					_, _ = w.Write([]byte(`
{
  "tags": [ "0.0.1", "0.0.2" ]
}
`))
				}

				client, err := ociclient.NewClient(logr.Discard(),
					ociclient.AllowPlainHttp(true),
					ociclient.WithKeyring(credentials.New()))
				Expect(err).ToNot(HaveOccurred())
				tags, err := client.ListTags(ctx, makeRef(repository))
				Expect(err).ToNot(HaveOccurred())
				Expect(tags).To(ConsistOf("0.0.1", "0.0.2"))
			})

		})

		Context("ListRepositories", func() {
			var (
				server  *httptest.Server
				host    string
				handler func(http.ResponseWriter, *http.Request)
				makeRef = func(repo string) string {
					return fmt.Sprintf("%s/%s", host, repo)
				}
			)

			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					handler(writer, request)
				}))

				hostUrl, err := url.Parse(server.URL)
				Expect(err).ToNot(HaveOccurred())
				host = hostUrl.Host
			})

			AfterEach(func() {
				server.Close()
			})

			It("should return a list of repositories", func() {
				var (
					ctx        = context.Background()
					repository = "myproject/repo"
				)
				defer ctx.Done()
				handler = func(w http.ResponseWriter, req *http.Request) {
					if req.URL.Path == "/v2/" {
						// first auth discovery call by the library
						w.WriteHeader(200)
						return
					}
					Expect(req.URL.String()).To(Equal("/v2/_catalog?n=1000"))
					w.WriteHeader(200)
					_, _ = w.Write([]byte(`
{
  "repositories": [ "myproject/repo/image1", "myproject/repo/image2" ]
}
`))
				}

				client, err := ociclient.NewClient(logr.Discard(),
					ociclient.AllowPlainHttp(true),
					ociclient.WithKeyring(credentials.New()))
				Expect(err).ToNot(HaveOccurred())
				repos, err := client.ListRepositories(ctx, makeRef(repository))
				Expect(err).ToNot(HaveOccurred())
				Expect(repos).To(ConsistOf(makeRef("myproject/repo/image1"), makeRef("myproject/repo/image2")))
			})

		})
	})

})
