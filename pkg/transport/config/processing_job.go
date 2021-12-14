// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/pkg/transport/filters"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/utils"
)

// ProcessingJob defines a type which contains all data for processing a single resource
// ProcessingJob describes a chain of multiple processors for processing a resource.
// Each processor receives its input from the preceding processor and writes the output for the
// subsequent processor. To work correctly, a pipeline must consist of 1 downloader, 0..n processors,
// and 1..n uploaders.
type ProcessingJob struct {
	ComponentDescriptor    *cdv2.ComponentDescriptor
	Resource               *cdv2.Resource
	Downloaders            []NamedResourceStreamProcessor
	Processors             []NamedResourceStreamProcessor
	Uploaders              []NamedResourceStreamProcessor
	ProcessedResource      *cdv2.Resource
	MatchedProcessingRules []string
	Log                    logr.Logger
	ProcessorTimeout       time.Duration
}

type NamedResourceStreamProcessor struct {
	Processor process.ResourceStreamProcessor
	Name      string
}

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
func (c *ProcessingJobFactory) Create(cd cdv2.ComponentDescriptor, res cdv2.Resource) (*ProcessingJob, error) {
	jobLog := c.log.WithValues("component-name", cd.Name, "component-version", cd.Version, "resource-name", res.Name, "resource-version", res.Version)
	job := ProcessingJob{
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
			job.Downloaders = append(job.Downloaders, NamedResourceStreamProcessor{
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
			job.Uploaders = append(job.Uploaders, NamedResourceStreamProcessor{
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
				job.Processors = append(job.Processors, NamedResourceStreamProcessor{
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

func (j *ProcessingJob) GetMatching() map[string][]string {
	matching := map[string][]string{
		"processingRules": j.MatchedProcessingRules,
	}

	for _, d := range j.Downloaders {
		matching["downloaders"] = append(matching["downloaders"], d.Name)
	}

	for _, u := range j.Uploaders {
		matching["uploaders"] = append(matching["uploaders"], u.Name)
	}

	return matching
}

// Process processes the resource
func (p *ProcessingJob) Process(ctx context.Context) error {
	inputFile, err := ioutil.TempFile("", "")
	if err != nil {
		p.Log.Error(err, "unable to create temporary input file")
		return err
	}

	if err := utils.WriteProcessorMessage(*p.ComponentDescriptor, *p.Resource, nil, inputFile); err != nil {
		p.Log.Error(err, "unable to write processor message")
		return err
	}

	processors := []NamedResourceStreamProcessor{}
	processors = append(processors, p.Downloaders...)
	processors = append(processors, p.Processors...)
	processors = append(processors, p.Uploaders...)

	for _, proc := range processors {
		procLog := p.Log.WithValues("processor-name", proc.Name)
		outputFile, err := p.runProcessor(ctx, inputFile, proc, procLog)
		if err != nil {
			procLog.Error(err, "unable to run processor")
			return err
		}

		// set the output file of the current processor as the input file for the next processor
		// if the current processor isn't last in the chain -> close file in runProcessor() in next loop iteration
		// if the current processor is last in the chain -> explicitely close file after loop
		inputFile = outputFile
	}
	defer inputFile.Close()

	if _, err := inputFile.Seek(0, io.SeekStart); err != nil {
		p.Log.Error(err, "unable to seek to beginning of file")
		return err
	}

	_, processedRes, blobreader, err := utils.ReadProcessorMessage(inputFile)
	if err != nil {
		p.Log.Error(err, "unable to read processor message")
		return err
	}
	if blobreader != nil {
		defer blobreader.Close()
	}

	p.ProcessedResource = &processedRes

	return nil
}

func (p *ProcessingJob) runProcessor(ctx context.Context, infile *os.File, proc NamedResourceStreamProcessor, log logr.Logger) (*os.File, error) {
	defer infile.Close()

	if _, err := infile.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("unable to seek to beginning of input file: %w", err)
	}

	outfile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, fmt.Errorf("unable to create temporary output file: %w", err)
	}

	ctx, cancelfunc := context.WithTimeout(ctx, p.ProcessorTimeout)
	defer cancelfunc()

	log.V(7).Info("starting processor")
	if err := proc.Processor.Process(ctx, infile, outfile); err != nil {
		return nil, fmt.Errorf("processor returned with error: %w", err)
	}
	log.V(7).Info("processor finished successfully")

	return outfile, nil
}
