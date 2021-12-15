// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"

	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/utils"
)

func NewProcessingJob(
	cd cdv2.ComponentDescriptor,
	res cdv2.Resource,
	downloaders []NamedResourceStreamProcessor,
	processors []NamedResourceStreamProcessor,
	uploaders []NamedResourceStreamProcessor,
	log logr.Logger,
	processorTimeout time.Duration,
) (*ProcessingJob, error) {
	if len(downloaders) != 1 {
		return nil, fmt.Errorf("a processing job must exactly have 1 downloader, found %d", len(downloaders))
	}

	if len(uploaders) < 1 {
		return nil, fmt.Errorf("a processing job must have at least 1 uploader, found %d", len(uploaders))
	}

	if log == nil {
		return nil, errors.New("log must not be nil")
	}

	j := ProcessingJob{
		ComponentDescriptor: &cd,
		Resource:            &res,
		Downloaders:         downloaders,
		Processors:          processors,
		Uploaders:           uploaders,
		Log:                 log,
		ProcessorTimeout:    processorTimeout,
	}
	return &j, nil
}

// ProcessingJob defines a type which contains all data for processing a single resource
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

func (j *ProcessingJob) GetProcessedResource() *cdv2.Resource {
	return j.ProcessedResource
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

func (j *ProcessingJob) Validate() error {
	if j.ComponentDescriptor == nil {
		return errors.New("component descriptor must not be nil")
	}

	if j.Resource == nil {
		return errors.New("resource must not be nil")
	}

	if len(j.Downloaders) != 1 {
		return fmt.Errorf("a processing job must exactly have 1 downloader, found %d", len(j.Downloaders))
	}

	if len(j.Uploaders) < 1 {
		return fmt.Errorf("a processing job must have at least 1 uploader, found %d", len(j.Uploaders))
	}

	return nil
}

// Process runs the processing job, by calling downloader, processors, and uploaders sequentially
// for the defined component descriptor and resource. Each processor receives its input from the
// preceding processor and writes the output for the subsequent processor. To work correctly,
// a processing job must consist of 1 downloader, 0..n processors, and 1..n uploaders.
func (j *ProcessingJob) Process(ctx context.Context) error {
	if err := j.Validate(); err != nil {
		j.Log.Error(err, "invalid processing job")
		return err
	}

	inputFile, err := ioutil.TempFile("", "")
	if err != nil {
		j.Log.Error(err, "unable to create temporary input file")
		return err
	}

	if err := utils.WriteProcessorMessage(*j.ComponentDescriptor, *j.Resource, nil, inputFile); err != nil {
		j.Log.Error(err, "unable to write processor message")
		return err
	}

	processors := []NamedResourceStreamProcessor{}
	processors = append(processors, j.Downloaders...)
	processors = append(processors, j.Processors...)
	processors = append(processors, j.Uploaders...)

	for _, proc := range processors {
		procLog := j.Log.WithValues("processor-name", proc.Name)
		outputFile, err := j.runProcessor(ctx, inputFile, proc, procLog)
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
		j.Log.Error(err, "unable to seek to beginning of file")
		return err
	}

	_, processedRes, blobreader, err := utils.ReadProcessorMessage(inputFile)
	if err != nil {
		j.Log.Error(err, "unable to read processor message")
		return err
	}
	if blobreader != nil {
		defer blobreader.Close()
	}

	j.ProcessedResource = &processedRes

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
