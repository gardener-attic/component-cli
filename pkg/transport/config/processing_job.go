// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gardener/component-cli/pkg/transport/filter"
	"github.com/gardener/component-cli/pkg/transport/process"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"sigs.k8s.io/yaml"
)

type ProcessingJob struct {
	ComponentDescriptor *cdv2.ComponentDescriptor
	Resource            *cdv2.Resource
	Downloaders         []namedResourceStreamProcessor
	Processors          []namedResourceStreamProcessor
	Uploaders           []namedResourceStreamProcessor
	ProcessedResource   *cdv2.Resource
}

type namedResourceStreamProcessor struct {
	Processor process.ResourceStreamProcessor
	Name      string
}

type parsedDownloaderDefinition struct {
	Name    string
	Type    string
	Spec    *json.RawMessage
	Filters []filter.Filter
}

type parsedProcessorDefinition struct {
	Name string
	Type string
	Spec *json.RawMessage
}

type parsedUploaderDefinition struct {
	Name    string
	Type    string
	Spec    *json.RawMessage
	Filters []filter.Filter
}

type parsedRuleDefinition struct {
	Name       string
	Processors []string
	Filters    []filter.Filter
}

type parsedTransportConfig struct {
	Downloaders []parsedDownloaderDefinition
	Processors  []parsedProcessorDefinition
	Uploaders   []parsedUploaderDefinition
	Rules       []parsedRuleDefinition
}

func NewProcessingJobFactory(transportCfgPath string, df *DownloaderFactory, pf *ProcessorFactory, uf *UploaderFactory) (*ProcessingJobFactory, error) {
	transportCfgYaml, err := os.ReadFile(transportCfgPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read transport config file: %w", err)
	}

	var transportCfg transportConfig
	err = yaml.Unmarshal(transportCfgYaml, &transportCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to parse transport config file: %w", err)
	}

	parsedTransportConfig, err := parseTransportConfig(&transportCfg)
	if err != nil {
		return nil, fmt.Errorf("failed creating lookup table %w", err)
	}

	c := ProcessingJobFactory{
		parsedConfig:      parsedTransportConfig,
		downloaderFactory: df,
		processorFactory:  pf,
		uploaderFactory:   uf,
	}

	return &c, nil
}

type ProcessingJobFactory struct {
	parsedConfig      *parsedTransportConfig
	uploaderFactory   *UploaderFactory
	downloaderFactory *DownloaderFactory
	processorFactory  *ProcessorFactory
}

// Create a ProcessorsLookup on the base of a config
func parseTransportConfig(config *transportConfig) (*parsedTransportConfig, error) {
	var parsedConfig parsedTransportConfig
	ff := NewFilterFactory()

	// downloaders
	for _, downloaderDefinition := range config.Downloaders {
		filters, err := createFilterList(downloaderDefinition.Filters, ff)
		if err != nil {
			return nil, fmt.Errorf("unable to create downloader %s: %w", downloaderDefinition.Name, err)
		}
		parsedConfig.Downloaders = append(parsedConfig.Downloaders, parsedDownloaderDefinition{
			Name:    downloaderDefinition.Name,
			Type:    downloaderDefinition.Type,
			Spec:    downloaderDefinition.Spec,
			Filters: filters,
		})
	}

	// processors
	for _, processorsDefinition := range config.Processors {
		parsedConfig.Processors = append(parsedConfig.Processors, parsedProcessorDefinition{
			Name: processorsDefinition.Name,
			Type: processorsDefinition.Type,
			Spec: processorsDefinition.Spec,
		})
	}

	// uploaders
	for _, uploaderDefinition := range config.Uploaders {
		filters, err := createFilterList(uploaderDefinition.Filters, ff)
		if err != nil {
			return nil, fmt.Errorf("unable to create uploader %s: %w", uploaderDefinition.Name, err)
		}
		parsedConfig.Uploaders = append(parsedConfig.Uploaders, parsedUploaderDefinition{
			Name:    uploaderDefinition.Name,
			Type:    uploaderDefinition.Type,
			Spec:    uploaderDefinition.Spec,
			Filters: filters,
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
			return nil, fmt.Errorf("unable to create rule %s: %w", rule.Name, err)
		}
		ruleLookup := parsedRuleDefinition{
			Name:       rule.Name,
			Processors: processors,
			Filters:    filters,
		}
		parsedConfig.Rules = append(parsedConfig.Rules, ruleLookup)
	}

	return &parsedConfig, nil
}

func (c *ProcessingJobFactory) Create(cd cdv2.ComponentDescriptor, res cdv2.Resource) (*ProcessingJob, error) {
	job := ProcessingJob{
		ComponentDescriptor: &cd,
		Resource:            &res,
	}

	// find matching downloader
	for _, downloader := range c.parsedConfig.Downloaders {
		if areAllFiltersMatching(downloader.Filters, cd, res) {
			dl, err := c.downloaderFactory.Create(string(downloader.Type), downloader.Spec)
			if err != nil {
				return nil, err
			}
			job.Downloaders = append(job.Downloaders, namedResourceStreamProcessor{
				Name:      downloader.Name,
				Processor: dl,
			})
		}
	}

	// find matching uploader
	for _, uploader := range c.parsedConfig.Uploaders {
		if areAllFiltersMatching(uploader.Filters, cd, res) {
			ul, err := c.uploaderFactory.Create(string(uploader.Type), uploader.Spec)
			if err != nil {
				return nil, err
			}
			job.Uploaders = append(job.Uploaders, namedResourceStreamProcessor{
				Name:      uploader.Name,
				Processor: ul,
			})
		}
	}

	// find matching processing rules
	for _, rule := range c.parsedConfig.Rules {
		if areAllFiltersMatching(rule.Filters, cd, res) {
			for _, processorName := range rule.Processors {
				processorDefined, err := findProcessorByName(processorName, c.parsedConfig)
				if err != nil {
					return nil, fmt.Errorf("failed compiling rule %s: %w", rule.Name, err)
				}
				p, err := c.processorFactory.Create(string(processorDefined.Type), processorDefined.Spec)
				if err != nil {
					return nil, err
				}
				job.Processors = append(job.Processors, namedResourceStreamProcessor{
					Name:      processorDefined.Name,
					Processor: p,
				})
			}
		}
	}

	return &job, nil
}

func areAllFiltersMatching(filters []filter.Filter, cd cdv2.ComponentDescriptor, res cdv2.Resource) bool {
	for _, filter := range filters {
		if !filter.Matches(&cd, res) {
			return false
		}
	}
	return true
}

func createFilterList(filterDefinitions []filterDefinition, ff *FilterFactory) ([]filter.Filter, error) {
	var filters []filter.Filter
	for _, f := range filterDefinitions {
		filter, err := ff.Create(f.Type, f.Spec)
		if err != nil {
			return nil, fmt.Errorf("error creating filter list for type %s with args %s: %w", f.Type, string(*f.Spec), err)
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

func findProcessorByName(name string, lookup *parsedTransportConfig) (*parsedProcessorDefinition, error) {
	for _, processor := range lookup.Processors {
		if processor.Name == name {
			return &processor, nil
		}
	}
	return nil, fmt.Errorf("unable to find processor %s", name)
}

func (j *ProcessingJob) Process(ctx context.Context) error {
	processors := []process.ResourceStreamProcessor{}

	for _, d := range j.Downloaders {
		processors = append(processors, d.Processor)
	}

	for _, p := range j.Processors {
		processors = append(processors, p.Processor)
	}

	for _, u := range j.Uploaders {
		processors = append(processors, u.Processor)
	}

	p := process.NewResourceProcessingPipeline(processors...)
	_, processedResource, err := p.Process(ctx, *j.ComponentDescriptor, *j.Resource)
	if err != nil {
		return err
	}

	j.ProcessedResource = &processedResource

	return nil
}
