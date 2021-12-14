// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/pkg/transport/filters"
	"github.com/gardener/component-cli/pkg/transport/process"
)

type parsedDownloaderDefinition struct {
	Name    string
	Type    string
	Spec    *json.RawMessage
	Filters []filters.Filter
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
	Filters []filters.Filter
}

type parsedProcessingRuleDefinition struct {
	Name       string
	Processors []string
	Filters    []filters.Filter
}

type ParsedTransportConfig struct {
	Downloaders     []parsedDownloaderDefinition
	Processors      []parsedProcessorDefinition
	Uploaders       []parsedUploaderDefinition
	ProcessingRules []parsedProcessingRuleDefinition
}

// NewProcessingJobFactory creates a new processing job factory
func NewProcessingJobFactory(transportCfg ParsedTransportConfig, df *DownloaderFactory, pf *ProcessorFactory, uf *UploaderFactory, log logr.Logger, processorTimeout time.Duration) (*ProcessingJobFactory, error) {
	c := ProcessingJobFactory{
		parsedConfig:      &transportCfg,
		downloaderFactory: df,
		processorFactory:  pf,
		uploaderFactory:   uf,
		log:               log,
		processorTimeout:  processorTimeout,
	}

	return &c, nil
}

// ProcessingJobFactory defines a helper struct for creating processing jobs
type ProcessingJobFactory struct {
	parsedConfig      *ParsedTransportConfig
	uploaderFactory   *UploaderFactory
	downloaderFactory *DownloaderFactory
	processorFactory  *ProcessorFactory
	log               logr.Logger
	processorTimeout  time.Duration
}

func ParseConfig(configFilePath string) (*ParsedTransportConfig, error) {
	transportCfgYaml, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read transport config file: %w", err)
	}

	var config TransportConfig
	if err := yaml.Unmarshal(transportCfgYaml, &config); err != nil {
		return nil, fmt.Errorf("unable to parse transport config file: %w", err)
	}

	var parsedConfig ParsedTransportConfig
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
	for _, rule := range config.ProcessingRules {
		processors := []string{}
		for _, processor := range rule.Processors {
			processors = append(processors, processor.Name)
		}
		filters, err := createFilterList(rule.Filters, ff)
		if err != nil {
			return nil, fmt.Errorf("unable to create rule %s: %w", rule.Name, err)
		}
		rule := parsedProcessingRuleDefinition{
			Name:       rule.Name,
			Processors: processors,
			Filters:    filters,
		}
		parsedConfig.ProcessingRules = append(parsedConfig.ProcessingRules, rule)
	}

	return &parsedConfig, nil
}

// Create creates a new processing job for a resource
func (c *ProcessingJobFactory) Create(cd cdv2.ComponentDescriptor, res cdv2.Resource) (*process.ProcessingJob, error) {
	jobLog := c.log.WithValues("component-name", cd.Name, "component-version", cd.Version, "resource-name", res.Name, "resource-version", res.Version)
	job := process.ProcessingJob{
		ComponentDescriptor: &cd,
		Resource:            &res,
		Log:                 jobLog,
		ProcessorTimeout:    c.processorTimeout,
	}

	// find matching downloader
	for _, downloader := range c.parsedConfig.Downloaders {
		if areAllFiltersMatching(downloader.Filters, cd, res) {
			dl, err := c.downloaderFactory.Create(string(downloader.Type), downloader.Spec)
			if err != nil {
				return nil, err
			}
			job.Downloaders = append(job.Downloaders, process.NamedResourceStreamProcessor{
				Name:      downloader.Name,
				Processor: dl,
			})
		}
	}

	// find matching uploaders
	for _, uploader := range c.parsedConfig.Uploaders {
		if areAllFiltersMatching(uploader.Filters, cd, res) {
			ul, err := c.uploaderFactory.Create(string(uploader.Type), uploader.Spec)
			if err != nil {
				return nil, err
			}
			job.Uploaders = append(job.Uploaders, process.NamedResourceStreamProcessor{
				Name:      uploader.Name,
				Processor: ul,
			})
		}
	}

	// find matching processing rules
	for _, rule := range c.parsedConfig.ProcessingRules {
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
				job.Processors = append(job.Processors, process.NamedResourceStreamProcessor{
					Name:      processorDefined.Name,
					Processor: p,
				})
				job.MatchedProcessingRules = append(job.MatchedProcessingRules, rule.Name)
			}
		}
	}

	return &job, nil
}

func areAllFiltersMatching(filters []filters.Filter, cd cdv2.ComponentDescriptor, res cdv2.Resource) bool {
	for _, filter := range filters {
		if !filter.Matches(cd, res) {
			return false
		}
	}
	return true
}

func createFilterList(filterDefinitions []FilterDefinition, ff *FilterFactory) ([]filters.Filter, error) {
	var filters []filters.Filter
	for _, f := range filterDefinitions {
		filter, err := ff.Create(f.Type, f.Spec)
		if err != nil {
			return nil, fmt.Errorf("error creating filter list for type %s with args %s: %w", f.Type, string(*f.Spec), err)
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

func findProcessorByName(name string, lookup *ParsedTransportConfig) (*parsedProcessorDefinition, error) {
	for _, processor := range lookup.Processors {
		if processor.Name == name {
			return &processor, nil
		}
	}
	return nil, fmt.Errorf("unable to find processor %s", name)
}
