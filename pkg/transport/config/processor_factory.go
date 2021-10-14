// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"fmt"

	"github.com/gardener/component-cli/pkg/transport/process"
)

func NewProcessorFactory() *ProcessorFactory{
	return &ProcessorFactory{}
}

type ProcessorFactory struct {
}

func (f *ProcessorFactory) Create(typ string, spec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	switch typ {
	case "executable":
		return createExecutable(spec)
	default:
		return nil, fmt.Errorf("unable to create processor: unknown type %s", typ)
	}
}