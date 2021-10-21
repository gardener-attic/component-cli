// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package filters

import (
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

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

func NewAccessTypeFilter(includeAccessTypes ...string) (Filter, error) {
	if len(includeAccessTypes) == 0 {
		return nil, fmt.Errorf("includeAccessTypes must not be empty")
	}

	filter := accessTypeFilter{
		includeAccessTypes: includeAccessTypes,
	}

	return &filter, nil
}
