// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package utils_test

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("repository context override", func() {

	Context("processing job", func() {

		It("should return overridden repository context if component name matches", func() {
			componentName := "github.com/gardener/component-cli"
			expectedRepoCtx := cdv2.NewOCIRegistryRepository("example-oci-registry.com/override", "")
			actualRepoCtx := repoCtxOverride.GetRepositoryContext(componentName, *defaultRepoCtx)
			Expect(actualRepoCtx).To(Equal(expectedRepoCtx))
		})

		It("should return default repository context if component name doesn't match", func() {
			componentName := "github.com/gardener/not-component-cli"
			expectedRepoCtx := cdv2.NewOCIRegistryRepository("example-oci-registry.com/base", "")
			actualRepoCtx := repoCtxOverride.GetRepositoryContext(componentName, *defaultRepoCtx)
			Expect(actualRepoCtx).To(Equal(expectedRepoCtx))
		})

	})

})
