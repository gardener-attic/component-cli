// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config_test

import (
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/pkg/transport/config"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Test Suite")
}

var (
	factory *config.ProcessingJobFactory
)

var _ = BeforeSuite(func() {
	transportCfg, err := config.ParseConfig("./testdata/transport.cfg")
	Expect(err).ToNot(HaveOccurred())

	client, err := ociclient.NewClient(logr.Discard())
	Expect(err).ToNot(HaveOccurred())
	ocicache := cache.NewInMemoryCache()
	targetCtx := cdv2.NewOCIRegistryRepository("my-target-registry.com/test", "")

	df := config.NewDownloaderFactory(client, ocicache)
	pf := config.NewProcessorFactory(ocicache)
	uf := config.NewUploaderFactory(client, ocicache, *targetCtx)

	factory, err = config.NewProcessingJobFactory(*transportCfg, df, pf, uf)
	Expect(err).ToNot(HaveOccurred())
}, 5)
