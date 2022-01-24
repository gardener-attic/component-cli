// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package processors

import (
	"encoding/json"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/extensions"
)

const (
	// ResourceLabelerProcessorType defines the type of a resource labeler
	ResourceLabelerProcessorType = "ResourceLabeler"

	// OCIArtifactFilterProcessorType defines the type of an oci artifact filter
	OCIArtifactFilterProcessorType = "OciArtifactFilter"
)

// NewProcessorFactory creates a new processor factory
func NewProcessorFactory(ociCache cache.Cache, log logr.Logger) *ProcessorFactory {
	return &ProcessorFactory{
		cache: ociCache,
		log:   log,
	}
}

// ProcessorFactory defines a helper struct for creating processors
type ProcessorFactory struct {
	cache cache.Cache
	log   logr.Logger
}

// Create creates a new processor defined by a type and a spec
func (f *ProcessorFactory) Create(processorType string, spec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	switch processorType {
	case ResourceLabelerProcessorType:
		return f.createResourceLabeler(spec)
	case OCIArtifactFilterProcessorType:
		return f.createOCIArtifactFilter(spec)
	case extensions.ExecutableType:
		return extensions.CreateExecutable(spec, f.log)
	default:
		return nil, fmt.Errorf("unknown processor type %s", processorType)
	}
}

func (f *ProcessorFactory) createResourceLabeler(rawSpec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	type processorSpec struct {
		Labels cdv2.Labels `json:"labels"`
	}

	var spec processorSpec
	if err := yaml.Unmarshal(*rawSpec, &spec); err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return NewResourceLabeler(spec.Labels...), nil
}

func (f *ProcessorFactory) createOCIArtifactFilter(rawSpec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	type processorSpec struct {
		RemovePatterns []string `json:"removePatterns"`
	}

	var spec processorSpec
	if err := yaml.Unmarshal(*rawSpec, &spec); err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return NewOCIArtifactFilter(f.cache, spec.RemovePatterns)
}
