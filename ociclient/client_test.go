// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/gardener/component-cli/ociclient/oci"
	"github.com/gardener/component-cli/pkg/testutils"

	"github.com/gardener/component-cli/ociclient"
)

var _ = Describe("client", func() {

	Context("Client", func() {

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

			ref := testenv.Addr + "/image-index/2/empty-img:v0.0.1"
			index := oci.Index{
				Manifests: []*oci.Manifest{},
				Annotations: map[string]string{
					"test": "test",
				},
			}

			tmp, err := oci.NewIndexArtifact(&index)
			Expect(err).ToNot(HaveOccurred())

			err = client.PushOCIArtifact(ctx, ref, tmp)
			Expect(err).ToNot(HaveOccurred())

			actualArtifact, err := client.GetOCIArtifact(ctx, ref)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualArtifact.IsManifest()).To(BeFalse())
			Expect(actualArtifact.IsIndex()).To(BeTrue())
			Expect(actualArtifact.GetIndex()).To(Equal(&index))
		}, 20)

		It("should push and pull an oci image index with only 1 manifest and no platform information", func() {
			ctx := context.Background()
			defer ctx.Done()

			ref := testenv.Addr + "/image-index/3/img:v0.0.1"
			manifest1Ref := testenv.Addr + "/image-index/1/img-platform-1:v0.0.1"
			manifest, mdesc, err := testutils.UploadTestManifest(ctx, client, manifest1Ref)
			Expect(err).ToNot(HaveOccurred())

			index := oci.Index{
				Manifests: []*oci.Manifest{
					{
						Descriptor: mdesc,
						Data:       manifest,
					},
				},
				Annotations: map[string]string{
					"test": "test",
				},
			}

			tmp, err := oci.NewIndexArtifact(&index)
			Expect(err).ToNot(HaveOccurred())

			err = client.PushOCIArtifact(ctx, ref, tmp)
			Expect(err).ToNot(HaveOccurred())

			actualArtifact, err := client.GetOCIArtifact(ctx, ref)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualArtifact.IsManifest()).To(BeFalse())
			Expect(actualArtifact.IsIndex()).To(BeTrue())
			Expect(actualArtifact.GetIndex()).To(Equal(&index))
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
