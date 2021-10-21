// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package filters

import (
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

type resourceTypeFilter struct {
	includeResourceTypes []string
}

func (f resourceTypeFilter) Matches(cd cdv2.ComponentDescriptor, r cdv2.Resource) bool {
	for _, resourceType := range f.includeResourceTypes {
		if r.Type == resourceType {
			return true
		}
	}
	return false
}

// NewResourceTypeFilter creates a new resourceTypeFilter
func NewResourceTypeFilter(includeResourceTypes ...string) (Filter, error) {
	if len(includeResourceTypes) == 0 {
		return nil, fmt.Errorf("includeResourceTypes must not be empty")
	}

	filter := resourceTypeFilter{
		includeResourceTypes: includeResourceTypes,
	}

	return &filter, nil
}
