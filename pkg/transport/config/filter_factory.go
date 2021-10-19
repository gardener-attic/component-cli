// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"fmt"

	"github.com/gardener/component-cli/pkg/transport/filter"
	"sigs.k8s.io/yaml"
)

func NewFilterFactory() *FilterFactory {
	return &FilterFactory{}
}

type FilterFactory struct{}

func (f *FilterFactory) Create(typ string, spec *json.RawMessage) (filter.Filter, error) {
	switch typ {
	case "ComponentFilter":
		return f.createComponentFilter(spec)
	case "ResourceTypeFilter":
		return f.createResourceTypeFilter(spec)
	case "AccessTypeFilter":
		return f.createAccessTypeFilter(spec)
	default:
		return nil, fmt.Errorf("unknown filter type %s", typ)
	}
}

func (f *FilterFactory) createComponentFilter(rawSpec *json.RawMessage) (filter.Filter, error) {
	type filterSpec struct {
		IncludeComponentNames []string `json:"includeComponentNames"`
	}

	var spec filterSpec
	err := yaml.Unmarshal(*rawSpec, &spec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return filter.NewComponentFilter(spec.IncludeComponentNames...)
}

func (f *FilterFactory) createResourceTypeFilter(rawSpec *json.RawMessage) (filter.Filter, error) {
	type filterSpec struct {
		IncludeResourceTypes []string `json:"includeResourceTypes"`
	}

	var spec filterSpec
	err := yaml.Unmarshal(*rawSpec, &spec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return filter.NewResourceTypeFilter(spec.IncludeResourceTypes...)
}

func (f *FilterFactory) createAccessTypeFilter(rawSpec *json.RawMessage) (filter.Filter, error) {
	type filterSpec struct {
		IncludeAccessTypes []string `json:"includeAccessTypes"`
	}

	var spec filterSpec
	err := yaml.Unmarshal(*rawSpec, &spec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return filter.NewAccessTypeFilter(spec.IncludeAccessTypes...)
}
