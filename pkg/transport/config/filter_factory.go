// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"fmt"

	"github.com/gardener/component-cli/pkg/transport/filter"
)

func NewFilterFactory() *FilterFactory {
	return &FilterFactory{}
}

type FilterFactory struct{}

func (f *FilterFactory) Create(typ string, spec *json.RawMessage) (filter.Filter, error) {
	switch typ {
	case "ComponentFilter":
		filter, err := filter.CreateComponentFilterFromConfig(spec)
		if err != nil {
			return nil, fmt.Errorf("can not create filter %s with provided args", typ)
		}
		return filter, nil
	default:
		return nil, fmt.Errorf("unable to create downloader: unknown type %s", typ)
	}
}
