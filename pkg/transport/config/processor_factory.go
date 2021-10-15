// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"fmt"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/processors"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"sigs.k8s.io/yaml"
)

func NewProcessorFactory(ociCache cache.Cache) *ProcessorFactory {
	return &ProcessorFactory{
		cache: ociCache,
	}
}

type ProcessorFactory struct {
	cache cache.Cache
}

func (f *ProcessorFactory) Create(typ string, spec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	switch typ {
	case "label":
		return f.createLabellingProcessor(spec)
	case "ociImageFilter":
		return f.createOCIImageFilter(spec)
	case "executable":
		return createExecutable(spec)
	default:
		return nil, fmt.Errorf("unknown processor type %s", typ)
	}
}

func (f *ProcessorFactory) createLabellingProcessor(rawSpec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	type processorSpec struct {
		Labels cdv2.Labels `json:"labels"`
	}

	var spec processorSpec
	err := yaml.Unmarshal(*rawSpec, &spec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return processors.NewLabellingProcessor(spec.Labels...), nil
}

func (f *ProcessorFactory) createOCIImageFilter(rawSpec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	type processorSpec struct {
		RemovePatterns []string `json:"removePatterns"`
	}

	var spec processorSpec
	err := yaml.Unmarshal(*rawSpec, &spec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return processors.NewOCIImageFilter(f.cache, spec.RemovePatterns), nil
}
