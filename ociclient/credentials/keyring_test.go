// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package credentials_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/ociclient/credentials"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "credentials Test Suite")
}

var _ = Describe("Keyrings", func() {

	It("should parse authentication config from a dockerconfig and match the hostname", func() {
		keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{"./testdata/dockerconfig.json"})
		Expect(err).ToNot(HaveOccurred())

		auth, ok := keyring.Get("eu.gcr.io")
		Expect(ok).To(BeTrue())
		Expect(auth.Username).To(Equal("test"))
	})

	It("should parse authentication config from a dockerconfig and match the hostname with protocol", func() {
		keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{"./testdata/dockerconfig.json"})
		Expect(err).ToNot(HaveOccurred())

		auth, ok := keyring.Get("docker.io")
		Expect(ok).To(BeTrue())
		Expect(auth.Username).To(Equal("docker"))
	})

	It("should match a whole resource url", func() {
		keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{"./testdata/dockerconfig.json"})
		Expect(err).ToNot(HaveOccurred())

		auth, ok := keyring.Get("eu.gcr.io/my-other-config/my-test:v1.2.3")
		Expect(ok).To(BeTrue())
		Expect(auth.Username).To(Equal("test"))
	})

	It("should match the hostname with a prefix", func() {
		keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{"./testdata/dockerconfig.json"})
		Expect(err).ToNot(HaveOccurred())

		auth, ok := keyring.Get("eu.gcr.io/my-proj/my-test:v1.2.3")
		Expect(ok).To(BeTrue())
		Expect(auth.Username).To(Equal("myproj"))
	})

})
