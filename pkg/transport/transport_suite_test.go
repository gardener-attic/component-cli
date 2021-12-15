// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package transport_test

import (
	"testing"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/pkg/transport"
	"github.com/gardener/component-cli/pkg/transport/config"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Transport Test Suite")
}

var (
	factory *transport.ProcessingJobFactory
)

var _ = BeforeSuite(func() {
	transportCfg, err := config.ParseTransportConfig("./testdata/transport.cfg")
	Expect(err).ToNot(HaveOccurred())

	client, err := ociclient.NewClient(logr.Discard())
	Expect(err).ToNot(HaveOccurred())
	ocicache := cache.NewInMemoryCache()
	targetCtx := cdv2.NewOCIRegistryRepository("my-target-registry.com/test", "")

	factory, err = transport.NewProcessingJobFactory(*transportCfg, client, ocicache, *targetCtx, logr.Discard(), 30*time.Second)
	Expect(err).ToNot(HaveOccurred())
}, 5)
