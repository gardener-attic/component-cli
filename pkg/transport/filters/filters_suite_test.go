// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package filters_test

import (
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	filter "github.com/gardener/component-cli/pkg/transport/filters"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Filters Test Suite")
}

var _ = Describe("filters", func() {

	Context("resourceAccessTypeFilter", func() {

		It("should match if access type is in include list", func() {
			cd := cdv2.ComponentDescriptor{}
			res := cdv2.Resource{
				Access: cdv2.NewEmptyUnstructured(cdv2.OCIRegistryType),
			}

			f, err := filter.NewResourceAccessTypeFilter(cdv2.OCIRegistryType)
			Expect(err).ToNot(HaveOccurred())

			actualMatch := f.Matches(cd, res)
			Expect(actualMatch).To(Equal(true))
		})

		It("should not match if access type is not in include list", func() {
			cd := cdv2.ComponentDescriptor{}
			res := cdv2.Resource{
				Access: cdv2.NewEmptyUnstructured(cdv2.OCIRegistryType),
			}

			f, err := filter.NewResourceAccessTypeFilter(cdv2.LocalOCIBlobType)
			Expect(err).ToNot(HaveOccurred())

			actualMatch := f.Matches(cd, res)
			Expect(actualMatch).To(Equal(false))
		})

		It("should return error upon creation if include list is empty", func() {
			includeAccessTypes := []string{}
			_, err := filter.NewResourceAccessTypeFilter(includeAccessTypes...)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("includeAccessTypes must not be empty"))
		})

	})

	Context("resourceTypeFilter", func() {

		It("should match if resource type is in include list", func() {
			cd := cdv2.ComponentDescriptor{}
			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "v0.1.0",
					Type:    cdv2.OCIImageType,
				},
			}

			f, err := filter.NewResourceTypeFilter(cdv2.OCIImageType)
			Expect(err).ToNot(HaveOccurred())

			actualMatch := f.Matches(cd, res)
			Expect(actualMatch).To(Equal(true))
		})

		It("should not match if resource type is not in include list", func() {
			cd := cdv2.ComponentDescriptor{}
			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "v0.1.0",
					Type:    "helm",
				},
			}

			f, err := filter.NewResourceTypeFilter(cdv2.OCIImageType)
			Expect(err).ToNot(HaveOccurred())

			actualMatch := f.Matches(cd, res)
			Expect(actualMatch).To(Equal(false))
		})

		It("should return error upon creation if include list is empty", func() {
			includeResourceTypes := []string{}
			_, err := filter.NewResourceTypeFilter(includeResourceTypes...)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("includeResourceTypes must not be empty"))
		})

	})

	Context("componentNameFilter", func() {

		It("should match if component name is in include list", func() {
			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name: "github.com/test/my-component",
					},
				},
			}
			res := cdv2.Resource{}

			f1, err := filter.NewComponentNameFilter("github.com/test/my-component")
			Expect(err).ToNot(HaveOccurred())

			match1 := f1.Matches(cd, res)
			Expect(match1).To(Equal(true))

			f2, err := filter.NewComponentNameFilter("github.com/test/*")
			Expect(err).ToNot(HaveOccurred())

			match2 := f2.Matches(cd, res)
			Expect(match2).To(Equal(true))
		})

		It("should not match if component name is not in include list", func() {
			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name: "github.com/test/my-component",
					},
				},
			}
			res := cdv2.Resource{}

			f1, err := filter.NewComponentNameFilter("github.com/test/my-other-component")
			Expect(err).ToNot(HaveOccurred())

			match1 := f1.Matches(cd, res)
			Expect(match1).To(Equal(false))

			f2, err := filter.NewComponentNameFilter("github.com/test-2/*")
			Expect(err).ToNot(HaveOccurred())

			match2 := f2.Matches(cd, res)
			Expect(match2).To(Equal(false))
		})

		It("should return error upon creation if include list is empty", func() {
			includeComponentNames := []string{}
			_, err := filter.NewComponentNameFilter(includeComponentNames...)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("includeComponentNames must not be empty"))
		})

		It("should return error upon creation if regexp is invalid", func() {
			_, err := filter.NewComponentNameFilter("github.com/\\")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error parsing regexp"))
		})

	})

})
