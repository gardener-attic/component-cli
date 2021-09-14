package process

import (
	"context"
	"os"
	"sync"
	"time"

	"archive/tar"
	"fmt"
	"io/ioutil"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

var TotalTime time.Duration = 0
var mux = sync.Mutex{}

type resourceProcessingPipelineImpl struct {
	processors []ResourceStreamProcessor
}

func (p *resourceProcessingPipelineImpl) Process(ctx context.Context, cd cdv2.ComponentDescriptor, res cdv2.Resource) (*cdv2.ComponentDescriptor, cdv2.Resource, error) {
	infile, err := ioutil.TempFile("", "out")
	if err != nil {
		return nil, cdv2.Resource{}, fmt.Errorf("unable to create temporary infile: %w", err)
	}

	err = WriteArchive(ctx, &cd, res, nil, tar.NewWriter(infile))
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

	processedCD, processedRes, blobreader, err := ReadArchive(tar.NewReader(infile))
	if err != nil {
		return nil, cdv2.Resource{}, fmt.Errorf("unable to read output data: %w", err)
	}
	defer blobreader.Close()

	return processedCD, processedRes, nil
}

func (p *resourceProcessingPipelineImpl) process(ctx context.Context, infile *os.File, proc ResourceStreamProcessor) (*os.File, error) {
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

func NewResourceProcessingPipeline(procs ...ResourceStreamProcessor) (ResourceProcessingPipeline, error) {
	p := resourceProcessingPipelineImpl{
		processors: procs,
	}
	return &p, nil
}
