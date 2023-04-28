package signature_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/gardener/component-cli/ociclient/test/envtest"
	"github.com/gardener/component-cli/pkg/logger"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "signature Test Suite")
}

var (
	testenv  *envtest.Environment
	client   ociclient.ExtendedClient
	ociCache cache.Cache
	keyring  *credentials.GeneralOciKeyring
)

var _ = BeforeSuite(func() {
	testenv = envtest.New(envtest.Options{
		RegistryBinaryPath: filepath.Join("../../../../", envtest.DefaultRegistryBinaryPath),
		Stdout:             GinkgoWriter,
		Stderr:             GinkgoWriter,
	})
	Expect(testenv.Start(context.Background())).To(Succeed())

	keyring = credentials.New()
	Expect(keyring.AddAuthConfig(testenv.Addr, credentials.AuthConfig{
		Username: testenv.BasicAuth.Username,
		Password: testenv.BasicAuth.Password,
	})).To(Succeed())
	ociCache = cache.NewInMemoryCache()
	var err error
	client, err = ociclient.NewClient(logr.Discard(), ociclient.WithKeyring(keyring), ociclient.WithCache(ociCache))
	Expect(err).ToNot(HaveOccurred())

	logger.SetLogger(logr.Discard())
}, 60)

var _ = AfterSuite(func() {
	Expect(testenv.Close()).To(Succeed())
})
