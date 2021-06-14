// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config/types"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/gardener/component-cli/ociclient/test/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CTF Test Suite")
}

var (
	testenv *envtest.Environment
	client  ociclient.ExtendedClient
	keyring *credentials.GeneralOciKeyring
)

var _ = BeforeSuite(func() {
	testenv = envtest.New(envtest.Options{
		RegistryBinaryPath: filepath.Join("../../../", envtest.DefaultRegistryBinaryPath),
		Stdout:             GinkgoWriter,
		Stderr:             GinkgoWriter,
	})
	Expect(testenv.Start(context.Background())).To(Succeed())

	keyring = credentials.New()
	Expect(keyring.AddAuthConfig(testenv.Addr, types.AuthConfig{
		Username: testenv.BasicAuth.Username,
		Password: testenv.BasicAuth.Password,
	})).To(Succeed())
	var err error
	client, err = ociclient.NewClient(logr.Discard(), ociclient.WithKeyring(keyring))
	Expect(err).ToNot(HaveOccurred())
}, 60)

var _ = AfterSuite(func() {
	Expect(testenv.Close()).To(Succeed())
})
