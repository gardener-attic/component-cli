// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gardener/component-cli/pkg/transport/filter"
	"github.com/gardener/component-cli/pkg/transport/process"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"sigs.k8s.io/yaml"
)

type ResourcePipeline struct {
	Cd          *cdv2.ComponentDescriptor
	Resource    *cdv2.Resource
	Downloaders []ProcessorWithName
	Processors  []ProcessorWithName
	Uploaders   []ProcessorWithName
}

type ProcessorWithName struct {
	Processor process.ResourceStreamProcessor
	Name      string
}

type DD struct {
	name    string
	typ     ExtensionType
	spec    *json.RawMessage
	filters []filter.Filter
}

type PD struct {
	name string
	typ  ExtensionType
	spec *json.RawMessage
}

type UD struct {
	name    string
	typ     ExtensionType
	spec    *json.RawMessage
	filters []filter.Filter
}

type RD struct {
	name       string
	processors []string
	filters    []filter.Filter
}

type ProcessorsLookup struct {
	downloaders []DD
	processors  []PD
	uploaders   []UD
	rules       []RD
}

func NewPipelineCompiler(transportCfgPath string, df *DownloaderFactory, pf *ProcessorFactory, uf *UploaderFactory) (*ProcessingPipelineCompiler, error) {
	transportCfgYaml, err := os.ReadFile(transportCfgPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read transport config file: %w", err)
	}

	var transportCfg transportConfig
	err = yaml.Unmarshal(transportCfgYaml, &transportCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to parse transport config file: %w", err)
	}

	compiler, err := compileFromConfig(&transportCfg)
	if err != nil {
		return nil, fmt.Errorf("failed creating lookup table %w", err)
	}

	c := ProcessingPipelineCompiler{
		lookup:            compiler,
		downloaderFactory: df,
		processorFactory:  pf,
		uploaderFactory:   uf,
	}

	return &c, nil
}

type ProcessingPipelineCompiler struct {
	lookup            *ProcessorsLookup
	uploaderFactory   *UploaderFactory
	downloaderFactory *DownloaderFactory
	processorFactory  *ProcessorFactory
}

// Create a ProcessingPipelineCompiler on the base of a config
func compileFromConfig(config *transportConfig) (*ProcessorsLookup, error) {
	var lookup ProcessorsLookup
	ff := NewFilterFactory()

	// downloader
	for _, downloaderDefinition := range config.Downloaders {
		filters, err := createFilterList(downloaderDefinition.Filters, ff)
		if err != nil {
			return nil, fmt.Errorf("failed creating downloader %s: %w", downloaderDefinition.Name, err)
		}
		lookup.downloaders = append(lookup.downloaders, DD{
			name:    downloaderDefinition.Name,
			typ:     downloaderDefinition.Type,
			spec:    downloaderDefinition.Spec,
			filters: filters,
		})
	}

	// processors
	for _, processorsDefinition := range config.Processors {
		lookup.processors = append(lookup.processors, PD{
			name: processorsDefinition.Name,
			typ:  processorsDefinition.Type,
			spec: processorsDefinition.Spec,
		})
	}

	// uploaders
	for _, uploaderDefinition := range config.Uploaders {
		filters, err := createFilterList(uploaderDefinition.Filters, ff)
		if err != nil {
			return nil, fmt.Errorf("failed creating downloader %s: %w", uploaderDefinition.Name, err)
		}
		lookup.uploaders = append(lookup.uploaders, UD{
			name:    uploaderDefinition.Name,
			typ:     uploaderDefinition.Type,
			spec:    uploaderDefinition.Spec,
			filters: filters,
		})
	}

	// rules
	for _, rule := range config.Rules {
		processors := []string{}
		for _, processor := range rule.Processors {
			processors = append(processors, processor.Name)
		}
		filters, err := createFilterList(rule.Filters, ff)
		if err != nil {
			return nil, fmt.Errorf("failed creating rule %s: %w", rule.Name, err)
		}
		ruleLookup := RD{
			name:       rule.Name,
			processors: processors,
			filters:    filters,
		}
		lookup.rules = append(lookup.rules, ruleLookup)
	}

	return &lookup, nil
}

func (c *ProcessingPipelineCompiler) CreateResourcePipeline(cd cdv2.ComponentDescriptor, res cdv2.Resource) (*ResourcePipeline, error) {
	pipeline := ResourcePipeline{
		Cd:       &cd,
		Resource: &res,
	}

	// find matching downloader
	for _, downloader := range c.lookup.downloaders {
		matches := doesAllFilterMatch(downloader.filters, cd, res)
		if matches {
			dl, err := c.downloaderFactory.Create(string(downloader.typ), downloader.spec)
			if err != nil {
				return nil, err
			}
			pipeline.Downloaders = append(pipeline.Downloaders, ProcessorWithName{
				Name:      downloader.name,
				Processor: dl,
			})
		}
	}

	// find matching uploader
	for _, uploader := range c.lookup.uploaders {
		matches := doesAllFilterMatch(uploader.filters, cd, res)
		if matches {
			ul, err := c.downloaderFactory.Create(string(uploader.typ), uploader.spec)
			if err != nil {
				return nil, err
			}
			pipeline.Uploaders = append(pipeline.Uploaders, ProcessorWithName{
				Name:      uploader.name,
				Processor: ul,
			})
		}
	}

	// loop through all rules to find corresponding processors
	for _, rule := range c.lookup.rules {
		matches := doesAllFilterMatch(rule.filters, cd, res)
		if matches {
			for _, processorName := range rule.processors {
				processorDefined, err := lookupProcessorByName(processorName, c.lookup)
				if err != nil {
					return nil, fmt.Errorf("failed compiling rule %s: %w", rule.name, err)
				}
				p, err := c.processorFactory.Create(string(processorDefined.typ), processorDefined.spec)
				if err != nil {
					return nil, err
				}
				pipeline.Processors = append(pipeline.Processors, ProcessorWithName{
					Name:      processorDefined.name,
					Processor: p,
				})
			}
		}
	}

	return &pipeline, nil
}

func doesAllFilterMatch(filters []filter.Filter, cd cdv2.ComponentDescriptor, res cdv2.Resource) bool {
	for _, filter := range filters {
		if !filter.Matches(&cd, res) {
			return false
		}
	}
	return true
}

func createFilterList(filterDefinitions []FilterDefinition, ff *FilterFactory) ([]filter.Filter, error) {
	var filters []filter.Filter
	for _, f := range filterDefinitions {
		filter, err := ff.Create(f.Type, f.Args)
		if err != nil {
			return nil, fmt.Errorf("error creating filter list for type %s with args %s: %w", f.Type, string(*f.Args), err)
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

func lookupProcessorByName(name string, lookup *ProcessorsLookup) (*PD, error) {
	for _, processor := range lookup.processors {
		if processor.name == name {
			return &processor, nil
		}
	}
	return nil, fmt.Errorf("can not find processor %s", name)
}
