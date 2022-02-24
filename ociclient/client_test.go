// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient_test

import (
	"bytes"
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
	"github.com/gardener/component-cli/ociclient/oci"
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

	expectedManifest, expectedManifestDesc, blobMap := testutils.CreateImage(configData, layersData)
	expectedManifestDesc.MediaType = manifestMediaType
	expectedManifestBytes, err := json.Marshal(expectedManifest)
	Expect(err).ToNot(HaveOccurred())

	store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		_, err := writer.Write(blobMap[desc.Digest])
		return err
	})

	Expect(client.PushRawManifest(ctx, ref, expectedManifestDesc, expectedManifestBytes, ociclient.WithStore(store))).To(Succeed())

	actualManifestDesc, actualManifestBytes, err := client.GetRawManifest(ctx, ref)
	Expect(err).ToNot(HaveOccurred())
	Expect(actualManifestDesc).To(Equal(expectedManifestDesc))
	Expect(actualManifestBytes).To(Equal(expectedManifestBytes))

	actualManifest := ocispecv1.Manifest{}
	Expect(json.Unmarshal(actualManifestBytes, &actualManifest)).To(Succeed())

	actualConfigBuf := bytes.NewBuffer([]byte{})
	Expect(client.Fetch(ctx, ref, actualManifest.Config, actualConfigBuf)).To(Succeed())
	Expect(actualConfigBuf.Bytes()).To(Equal(configData))

	for i, layerDesc := range actualManifest.Layers {
		actualLayerBuf := bytes.NewBuffer([]byte{})
		Expect(client.Fetch(ctx, ref, layerDesc, actualLayerBuf)).To(Succeed())
		Expect(actualLayerBuf.Bytes()).To(Equal(layersData[i]))
	}
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

	manifest1, manifest1Desc, blobMap := testutils.CreateImage(configData, layersData)
	manifest1Desc.Platform = &ocispecv1.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}
	manifest1Bytes, err := json.Marshal(manifest1)
	Expect(err).ToNot(HaveOccurred())
	store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		_, err := writer.Write(blobMap[desc.Digest])
		return err
	})
	Expect(client.PushRawManifest(ctx, ref, manifest1Desc, manifest1Bytes, ociclient.WithStore(store)))

	manifest2, manifest2Desc, blobMap := testutils.CreateImage(configData, layersData)
	manifest2Desc.Platform = &ocispecv1.Platform{
		Architecture: "amd64",
		OS:           "windows",
	}
	manifest2Bytes, err := json.Marshal(manifest2)
	Expect(err).ToNot(HaveOccurred())
	store = ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		_, err := writer.Write(blobMap[desc.Digest])
		return err
	})
	Expect(client.PushRawManifest(ctx, ref, manifest2Desc, manifest2Bytes, ociclient.WithStore(store)))

	index := ocispecv1.Index{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Manifests: []ocispecv1.Descriptor{
			manifest1Desc,
			manifest2Desc,
		},
		Annotations: map[string]string{
			"test": "test",
		},
	}

	indexBytes, err := json.Marshal(index)
	Expect(err).ToNot(HaveOccurred())

	indexDesc := ocispecv1.Descriptor{
		MediaType: indexMediaType,
		Digest:    digest.FromBytes(indexBytes),
		Size:      int64(len(indexBytes)),
	}
	blobMap[indexDesc.Digest] = indexBytes

	store = ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		_, err := writer.Write(blobMap[desc.Digest])
		return err
	})

	Expect(client.PushRawManifest(ctx, ref, indexDesc, indexBytes, ociclient.WithStore(store))).To(Succeed())

	actualIndexDesc, actualIndexBytes, err := client.GetRawManifest(ctx, ref)
	Expect(err).ToNot(HaveOccurred())
	Expect(actualIndexDesc).To(Equal(indexDesc))
	Expect(actualIndexBytes).To(Equal(indexBytes))
}

var _ = Describe("client", func() {

	Context("Client", func() {

		It("should push and pull a single architecture image without modifications (oci media type)", func() {
			RunPushAndPullImageTest("single-arch-tests/0/artifact:0.0.1", ocispecv1.MediaTypeImageManifest)
		}, 20)

		// TODO: investigate why this test isn't working (could be registry not accepting docker media type)
		// It("should push and pull a single architecture image without modifications (docker media type)", func() {
		// 	RunPushAndPullTest("single-arch-tests/1/artifact:0.0.1", images.MediaTypeDockerSchema2Manifest)
		// }, 20)

		It("should push and pull a multi architecture image without modifications (oci media type)", func() {
			RunPushAndPullImageIndexTest("multi-arch-tests/0/artifact:0.0.1", ocispecv1.MediaTypeImageIndex)
		}, 20)

		// TODO: investigate why this test isn't working (could be registry not accepting docker media type)
		// It("should push and pull a multi architecture image without modifications (docker media type)", func() {
		// 	RunPushAndPullImageIndexTest("multi-arch-tests/1/artifact:0.0.1", images.MediaTypeDockerSchema2ManifestList)
		// }, 20)

		It("should push and pull an oci artifact", func() {
			ctx := context.Background()
			defer ctx.Done()

			ref := testenv.Addr + "/test/artifact:v0.0.1"
			manifest, mdesc, err := testutils.UploadTestManifest(ctx, client, ref)
			Expect(err).ToNot(HaveOccurred())

			res, err := client.GetManifest(ctx, ref)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Config).To(Equal(manifest.Config))
			Expect(res.Layers).To(Equal(manifest.Layers))

			// TODO: oci image index test only working because cache is filled in this function with config/layer blobs. should be fixed
			expectedManifest := oci.Manifest{
				Descriptor: mdesc,
				Data:       manifest,
			}
			testutils.CompareRemoteManifest(
				client,
				ref,
				expectedManifest,
				[]byte("config-data"),
				[][]byte{
					[]byte("layer-data"),
				},
			)
		}, 20)

		It("should push and pull an oci image index", func() {
			ctx := context.Background()
			defer ctx.Done()

			indexRef := testenv.Addr + "/image-index/1/img:v0.0.1"
			index, err := testutils.UploadTestIndex(ctx, client, indexRef)
			Expect(err).ToNot(HaveOccurred())

			actualArtifact, err := client.GetOCIArtifact(ctx, indexRef)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualArtifact.IsManifest()).To(BeFalse())
			Expect(actualArtifact.IsIndex()).To(BeTrue())
			Expect(actualArtifact.GetIndex()).To(Equal(index))
		}, 20)

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
			imageRef := testenv.Addr + "/multi-arch-tests/4/img:v0.0.1"

			m, mdesc, blobMap := testutils.CreateImage(configData, layersData)
			mbytes, err := json.Marshal(m)
			Expect(err).ToNot(HaveOccurred())

			store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
				_, err := writer.Write(blobMap[desc.Digest])
				return err
			})

			Expect(client.PushRawManifest(ctx, imageRef, mdesc, mbytes, ociclient.WithStore(store))).To(Succeed())

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

			store = ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
				_, err := writer.Write(indexBytes)
				return err
			})

			Expect(client.PushRawManifest(ctx, imageRef, indexDesc, indexBytes, ociclient.WithStore(store))).To(Succeed())

			actualIndexDesc, actualIndexBytes, err := client.GetRawManifest(ctx, imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualIndexDesc).To(Equal(indexDesc))
			Expect(actualIndexBytes).To(Equal(indexBytes))
		}, 20)

		It("should copy an oci artifact", func() {
			ctx := context.Background()
			defer ctx.Done()

			ref := testenv.Addr + "/test/artifact:v0.0.1"
			manifest, _, err := testutils.UploadTestManifest(ctx, client, ref)
			Expect(err).ToNot(HaveOccurred())

			newRef := testenv.Addr + "/new/artifact:v0.0.1"
			Expect(ociclient.Copy(ctx, client, ref, newRef)).To(Succeed())

			res, err := client.GetManifest(ctx, newRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Config).To(Equal(manifest.Config))
			Expect(res.Layers).To(Equal(manifest.Layers))

			var configBlob bytes.Buffer
			Expect(client.Fetch(ctx, ref, res.Config, &configBlob)).To(Succeed())
			Expect(configBlob.String()).To(Equal("config-data"))

			var layerBlob bytes.Buffer
			Expect(client.Fetch(ctx, ref, res.Layers[0], &layerBlob)).To(Succeed())
			Expect(layerBlob.String()).To(Equal("layer-data"))
		}, 20)

		It("should copy an oci image index", func() {
			ctx := context.Background()
			defer ctx.Done()

			ref := testenv.Addr + "/copy/image-index/src/img:v0.0.1"
			index, err := testutils.UploadTestIndex(ctx, client, ref)
			Expect(err).ToNot(HaveOccurred())

			newRef := testenv.Addr + "/copy/image-index/tgt/img:v0.0.1"
			Expect(ociclient.Copy(ctx, client, ref, newRef)).To(Succeed())

			actualArtifact, err := client.GetOCIArtifact(ctx, newRef)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualArtifact.IsManifest()).To(BeFalse())
			Expect(actualArtifact.IsIndex()).To(BeTrue())
			Expect(actualArtifact.GetIndex()).To(Equal(index))

			for i := range actualArtifact.GetIndex().Manifests {
				testutils.CompareRemoteManifest(
					client,
					ref,
					*index.Manifests[i],
					[]byte("config-data"),
					[][]byte{
						[]byte("layer-data"),
					},
				)
			}
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
