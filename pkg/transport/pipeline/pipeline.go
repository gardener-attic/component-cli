package pipeline

import (
	"context"
	"os"
	"sync"
	"time"

	"archive/tar"
	"fmt"
	"io/ioutil"

	"github.com/gardener/component-cli/pkg/transport/download"
	"github.com/gardener/component-cli/pkg/transport/processor"
	"github.com/gardener/component-cli/pkg/transport/util"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

var TotalTime time.Duration = 0
var mux = sync.Mutex{}

type ResourceProcessingPipeline interface {
	Process(context.Context, *cdv2.ComponentDescriptor, cdv2.Resource) (*cdv2.ComponentDescriptor, cdv2.Resource, error)
}

type sequentialPipeline struct {
	processors []processor.ResourceStreamProcessor
}

func (p *sequentialPipeline) Process(ctx context.Context, cd *cdv2.ComponentDescriptor, res cdv2.Resource) (*cdv2.ComponentDescriptor, cdv2.Resource, error) {
	infile, err := ioutil.TempFile("", "out")
	if err != nil {
		return nil, cdv2.Resource{}, fmt.Errorf("unable to create temporary infile: %w", err)
	}

	err = util.WriteArchive(ctx, cd, res, nil, tar.NewWriter(infile))
	if err != nil {
		return nil, cdv2.Resource{}, fmt.Errorf("unable to write: %w", err)
	}

	start := time.Now()

	for _, proc := range p.processors {
		outfile, err := p.process(ctx, infile, proc)
		if err != nil {
			return nil, cdv2.Resource{}, err
		}

		infile = outfile
	}

	end := time.Now()
	delta := end.Sub(start)
	mux.Lock()
	TotalTime += delta
	mux.Unlock()

	defer infile.Close()

	_, err = infile.Seek(0, 0)
	if err != nil {
		return nil, cdv2.Resource{}, err
	}

	cd, res, _, err = util.ReadArchive(tar.NewReader(infile))
	if err != nil {
		return nil, cdv2.Resource{}, fmt.Errorf("unable to read output data: %w", err)
	}

	return cd, res, nil
}

func (p *sequentialPipeline) process(ctx context.Context, infile *os.File, proc processor.ResourceStreamProcessor) (*os.File, error) {
	defer infile.Close()

	_, err := infile.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("unable to seek to beginning of input file: %w", err)
	}

	outfile, err := ioutil.TempFile("", "out")
	if err != nil {
		return nil, fmt.Errorf("unable to create temporary outfile: %w", err)
	}

	inreader := infile
	outwriter := outfile

	err = proc.Process(ctx, inreader, outwriter)
	if err != nil {
		return nil, fmt.Errorf("unable to process resource: %w", err)
	}

	return outfile, nil
}

func NewSequentialPipeline() (ResourceProcessingPipeline, error) {
	procBins := []string{
		"/Users/i500806/dev/pipeman/bin/processor_1",
		"/Users/i500806/dev/pipeman/bin/processor_2",
		"/Users/i500806/dev/pipeman/bin/processor_3",
	}

	procs := []processor.ResourceStreamProcessor{
		download.NewLocalOCIBlobDownloader(),
	}

	for _, procBin := range procBins {
		exec, err := processor.NewUDSExecutable(procBin)
		if err != nil {
			return nil, err
		}
		procs = append(procs, exec)
	}

	// procs = append(procs, upload.NewLocalOCIBlobUploader())

	pip := sequentialPipeline{
		processors: procs,
	}

	return &pip, nil
}
