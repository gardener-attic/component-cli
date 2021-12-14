// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"

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
	Processor ResourceStreamProcessor
	Name      string
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
