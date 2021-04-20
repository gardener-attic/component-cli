// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	testlog "github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/ociclient"
)

var _ = Describe("client", func() {

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

		It("", func() {
			var (
				ctx        = context.Background()
				repository = "myproject/repo/myimage"
			)
			defer ctx.Done()
			handler = func(w http.ResponseWriter, req *http.Request) {
				Expect(req.URL.String()).To(Equal("/v2/myproject/repo/myimage/tags/list?n=1000"))
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`
{
  "tags": [ "0.0.1", "0.0.2" ]
}
`))
			}

			client, err := ociclient.NewClient(testlog.NullLogger{}, ociclient.AllowPlainHttp(true))
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
				Expect(req.URL.String()).To(Equal("/v2/_catalog?n=1000"))
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`
{
  "repositories": [ "myproject/repo/image1", "myproject/repo/image2" ]
}
`))
			}

			client, err := ociclient.NewClient(testlog.NullLogger{}, ociclient.AllowPlainHttp(true))
			Expect(err).ToNot(HaveOccurred())
			repos, err := client.ListRepositories(ctx, makeRef(repository))
			Expect(err).ToNot(HaveOccurred())
			Expect(repos).To(ConsistOf(makeRef("myproject/repo/image1"), makeRef("myproject/repo/image2")))
		})

	})

})
