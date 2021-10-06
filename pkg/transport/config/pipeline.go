package config

import (
	"encoding/json"
	"fmt"

	"github.com/gardener/component-cli/pkg/transport/filter"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/downloaders"
	"github.com/gardener/component-cli/pkg/transport/process/uploaders"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
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

type ProcessorsLookup struct {
	downloaders []struct {
		ProcessorWithName
		filters []filter.Filter
	}
	processors []ProcessorWithName

	uploaders []struct {
		ProcessorWithName
		filters []filter.Filter
	}

	rules []struct {
		name       string
		processors []string
		filters    []filter.Filter
	}
}

type ProcessingPipelineCompiler struct {
	lookup ProcessorsLookup
}

// Create a ProcessingPipelineCompiler on the base of a config
func CompileFromConfig(config *Config) (*ProcessingPipelineCompiler, error) {
	var lookup ProcessorsLookup

	// downloader
	for _, downlaoderDefinition := range config.Downloaders {
		if downlaoderDefinition.Type == ExecutableProcessor {
			fmt.Println("Not yet implemented")
		} else {
			downloader := createBuiltInProcessor(string(downlaoderDefinition.Type), downlaoderDefinition.Spec)
			filters, err := createFilterList(downlaoderDefinition.Filters)
			if err != nil {
				return nil, fmt.Errorf("failed creating downloader %s: %w", downlaoderDefinition.Name, err)
			}
			lookup.downloaders = append(lookup.downloaders, struct {
				ProcessorWithName
				filters []filter.Filter
			}{ProcessorWithName{downloader, downlaoderDefinition.Name}, filters})
		}
	}

	// processors
	for _, processorsDefinition := range config.Processors {
		if processorsDefinition.Type == ExecutableProcessor {
			fmt.Println("Not yet implemented")
		} else {
			processor := createBuiltInProcessor(string(processorsDefinition.Type), processorsDefinition.Spec)
			lookup.processors = append(lookup.processors, struct {
				Processor process.ResourceStreamProcessor
				Name      string
			}{processor, processorsDefinition.Name})
		}
	}

	// uploaders
	for _, uploaderDefinition := range config.Uploaders {
		if uploaderDefinition.Type == ExecutableProcessor {
			fmt.Println("Not yet implemented")
		} else {
			uploader := createBuiltInProcessor(string(uploaderDefinition.Type), uploaderDefinition.Spec)
			filters, err := createFilterList(uploaderDefinition.Filters)
			if err != nil {
				return nil, fmt.Errorf("failed creating downloader %s: %w", uploaderDefinition.Name, err)
			}
			lookup.uploaders = append(lookup.uploaders, struct {
				ProcessorWithName
				filters []filter.Filter
			}{ProcessorWithName{uploader, uploaderDefinition.Name}, filters})
		}
	}

	// rules
	for _, rule := range config.Rules {
		var ruleLookup struct {
			name       string
			processors []string
			filters    []filter.Filter
		}
		ruleLookup.name = rule.Name
		for _, processor := range rule.Processors {
			ruleLookup.processors = append(ruleLookup.processors, processor.Name)
		}
		filters, err := createFilterList(rule.Filters)
		if err != nil {
			return nil, fmt.Errorf("failed creating rule %s: %w", rule.Name, err)
		}
		ruleLookup.filters = filters
		lookup.rules = append(lookup.rules, ruleLookup)
	}

	return &ProcessingPipelineCompiler{
		lookup: lookup,
	}, nil
}

func (c *ProcessingPipelineCompiler) CreateResourcePipeline(cds []cdv2.ComponentDescriptor) ([]ResourcePipeline, error) {
	var pipelines []ResourcePipeline

	// loop through all resources
	for _, cd := range cds {
		for _, res := range cd.Resources {
			var pipeline ResourcePipeline
			pipeline.Cd = &cd
			pipeline.Resource = &res

			// find matching downloader
			for _, downloader := range c.lookup.downloaders {
				matches := doesAllFilterMatch(downloader.filters, cd, res)
				if matches {
					pipeline.Downloaders = append(pipeline.Downloaders, ProcessorWithName{downloader.Processor, downloader.Name})
				}
			}

			// find matching uploader
			for _, uploader := range c.lookup.uploaders {
				matches := doesAllFilterMatch(uploader.filters, cd, res)
				if matches {
					pipeline.Uploaders = append(pipeline.Uploaders, ProcessorWithName{uploader.Processor, uploader.Name})
				}
			}

			// loop through all rules to find corresponding processors
			for _, rule := range c.lookup.rules {
				matches := doesAllFilterMatch(rule.filters, cd, res)
				if matches {
					for _, processorName := range rule.processors {
						processorDefined, err := lookupProcessorByName(processorName, &c.lookup)
						if err != nil {
							return nil, fmt.Errorf("failed compiling rule %s: %w", rule.name, err)
						}
						pipeline.Processors = append(pipeline.Processors, ProcessorWithName{processorDefined.Processor, processorDefined.Name})
					}
				}
			}
			pipelines = append(pipelines, pipeline)
		}
	}

	return pipelines, nil
}

func doesAllFilterMatch(filters []filter.Filter, cd cdv2.ComponentDescriptor, res cdv2.Resource) bool {
	for _, filter := range filters {
		if !filter.Matches(&cd, res) {
			return false
		}
	}
	return true
}

func createBuiltInProcessor(builtinType string, spec *json.RawMessage) process.ResourceStreamProcessor {
	switch builtinType {
	case "LocalOCIBlobDownloader": //TODO: make to constant
		// TODO parse config into corresponding config structure
		return downloaders.NewLocalOCIBlobDownloader(nil) // TODO: pass correct oci client
	case "LocalOCIBlobUploader":
		// TODO parse config into corresponding config structure
		return uploaders.NewLocalOCIBlobUploader(nil, cdv2.OCIRegistryRepository{}) // TODO: pass correct oci client
	}
	return nil // TODO: change to error
}

func createFilterList(filterDefinitions []FilterDefinition) ([]filter.Filter, error) {
	var filters []filter.Filter
	for _, f := range filterDefinitions {
		filter, err := createFilter(f.Type, f.Args)
		if err != nil {
			return nil, fmt.Errorf("error creating filter list for type %s with args %s: %w", f.Type, string(*f.Args), err)
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

func createFilter(filterType string, args *json.RawMessage) (filter.Filter, error) {
	switch filterType {
	case "ComponentFilter": // TODO: make constant
		filter, err := filter.CreateComponentFilterFromConfig(args)
		if err != nil {
			return nil, fmt.Errorf("can not create filter %s with provided args", filterType)
		}
		return filter, nil
	}
	return nil, fmt.Errorf("can not find filter %s", filterType)
}

func lookupProcessorByName(name string, lookup *ProcessorsLookup) (*ProcessorWithName, error) {
	for _, processor := range lookup.processors {
		if processor.Name == name {
			return &processor, nil
		}
	}
	return nil, fmt.Errorf("can not find processor %s", name)
}
