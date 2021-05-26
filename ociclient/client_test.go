// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	testlog "github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/component-cli/ociclient/credentials"

	"github.com/gardener/component-cli/ociclient"
)

var _ = Describe("client", func() {

	Context("Client", func() {

		It("should push and pull an oci artifact", func() {
			ctx := context.Background()
			defer ctx.Done()

			manifest, ref := uploadDefaultManifest(ctx)

			res, err := client.GetManifest(ctx, ref)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Config).To(Equal(manifest.Config))
			Expect(res.Layers).To(Equal(manifest.Layers))

			var configBlob bytes.Buffer
			Expect(client.Fetch(ctx, ref, res.Config, &configBlob)).To(Succeed())
			Expect(configBlob.String()).To(Equal("test"))

			var layerBlob bytes.Buffer
			Expect(client.Fetch(ctx, ref, res.Layers[0], &layerBlob)).To(Succeed())
			Expect(layerBlob.String()).To(Equal("test-config"))
		}, 20)

		It("should copy an oci artifact", func() {
			ctx := context.Background()
			defer ctx.Done()

			manifest, ref := uploadDefaultManifest(ctx)

			newRef := testenv.Addr + "/new/artifact:v0.0.1"
			Expect(ociclient.Copy(ctx, client, ref, newRef)).To(Succeed())

			res, err := client.GetManifest(ctx, newRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Config).To(Equal(manifest.Config))
			Expect(res.Layers).To(Equal(manifest.Layers))

			var configBlob bytes.Buffer
			Expect(client.Fetch(ctx, ref, res.Config, &configBlob)).To(Succeed())
			Expect(configBlob.String()).To(Equal("test"))

			var layerBlob bytes.Buffer
			Expect(client.Fetch(ctx, ref, res.Layers[0], &layerBlob)).To(Succeed())
			Expect(layerBlob.String()).To(Equal("test-config"))
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

				client, err := ociclient.NewClient(testlog.NullLogger{},
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

				client, err := ociclient.NewClient(testlog.NullLogger{},
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

func uploadDefaultManifest(ctx context.Context) (*ocispecv1.Manifest, string) {
	data := []byte("test")
	layerData := []byte("test-config")
	manifest := &ocispecv1.Manifest{
		Config: ocispecv1.Descriptor{
			MediaType: "text/plain",
			Digest:    digest.FromBytes(data),
			Size:      int64(len(data)),
		},
		Layers: []ocispecv1.Descriptor{
			{
				MediaType: "text/plain",
				Digest:    digest.FromBytes(layerData),
				Size:      int64(len(layerData)),
			},
		},
	}
	store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		switch desc.Digest.String() {
		case manifest.Config.Digest.String():
			_, err := writer.Write(data)
			return err
		default:
			_, err := writer.Write(layerData)
			return err
		}
	})
	ref := testenv.Addr + "/test/artifact:v0.0.1"
	Expect(client.PushManifest(ctx, ref, manifest, ociclient.WithStore(store))).To(Succeed())
	return manifest, ref
}
