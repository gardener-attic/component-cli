// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package utils_test

import (
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/pkg/utils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utils Test Suite")
}

var (
	defaultRepoCtx  *cdv2.OCIRegistryRepository
	repoCtxOverride *utils.RepositoryContextOverride
)

var _ = BeforeSuite(func() {
	defaultRepoCtx = cdv2.NewOCIRegistryRepository("example-oci-registry.com/base", "")
	var err error
	repoCtxOverride, err = utils.ParseRepositoryContextOverrideConfig("./testdata/repo-ctx-override.cfg")
	Expect(err).ToNot(HaveOccurred())
}, 5)
