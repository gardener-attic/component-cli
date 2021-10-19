// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package filter

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

type accessTypeFilter struct {
	includeAccessTypes []string
}

func (f accessTypeFilter) Matches(cd *cdv2.ComponentDescriptor, r cdv2.Resource) bool {
	for _, accessType := range f.includeAccessTypes {
		if r.Access.Type == accessType {
			return true
		}
	}
	return false
}

func NewAccessTypeFilter(includeAccessTypes ...string) (Filter, error) {
	filter := accessTypeFilter{
		includeAccessTypes: includeAccessTypes,
	}

	return &filter, nil
}
