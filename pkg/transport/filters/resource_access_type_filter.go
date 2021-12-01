// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package filters

import (
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

type AccessTypeFilterSpec struct {
	IncludeAccessTypes []string `json:"includeAccessTypes"`
}

type accessTypeFilter struct {
	includeAccessTypes []string
}

func (f accessTypeFilter) Matches(cd cdv2.ComponentDescriptor, r cdv2.Resource) bool {
	for _, accessType := range f.includeAccessTypes {
		if r.Access.Type == accessType {
			return true
		}
	}
	return false
}

// NewAccessTypeFilter creates a new accessTypeFilter
func NewAccessTypeFilter(spec AccessTypeFilterSpec) (Filter, error) {
	if len(spec.IncludeAccessTypes) == 0 {
		return nil, fmt.Errorf("includeAccessTypes must not be empty")
	}

	filter := accessTypeFilter{
		includeAccessTypes: spec.IncludeAccessTypes,
	}

	return &filter, nil
}
