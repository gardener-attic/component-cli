// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"fmt"

	"github.com/gardener/component-cli/pkg/transport/filters"
	"sigs.k8s.io/yaml"
)

const (
	ComponentNameFilterType = "ComponentNameFilter"
	ResourceTypeFilterType  = "ResourceTypeFilter"
	AccessTypeFilterType    = "ResourceAccessTypeFilter"
)

func NewFilterFactory() *FilterFactory {
	return &FilterFactory{}
}

type FilterFactory struct{}

func (f *FilterFactory) Create(typ string, spec *json.RawMessage) (filters.Filter, error) {
	switch typ {
	case ComponentNameFilterType:
		return f.createComponentNameFilter(spec)
	case ResourceTypeFilterType:
		return f.createResourceTypeFilter(spec)
	case AccessTypeFilterType:
		return f.createAccessTypeFilter(spec)
	default:
		return nil, fmt.Errorf("unknown filter type %s", typ)
	}
}

func (f *FilterFactory) createComponentNameFilter(rawSpec *json.RawMessage) (filters.Filter, error) {
	type filterSpec struct {
		IncludeComponentNames []string `json:"includeComponentNames"`
	}

	var spec filterSpec
	err := yaml.Unmarshal(*rawSpec, &spec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return filters.NewComponentNameFilter(spec.IncludeComponentNames...)
}

func (f *FilterFactory) createResourceTypeFilter(rawSpec *json.RawMessage) (filters.Filter, error) {
	type filterSpec struct {
		IncludeResourceTypes []string `json:"includeResourceTypes"`
	}

	var spec filterSpec
	err := yaml.Unmarshal(*rawSpec, &spec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return filters.NewResourceTypeFilter(spec.IncludeResourceTypes...)
}

func (f *FilterFactory) createAccessTypeFilter(rawSpec *json.RawMessage) (filters.Filter, error) {
	type filterSpec struct {
		IncludeAccessTypes []string `json:"includeAccessTypes"`
	}

	var spec filterSpec
	err := yaml.Unmarshal(*rawSpec, &spec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return filters.NewResourceAccessTypeFilter(spec.IncludeAccessTypes...)
}
