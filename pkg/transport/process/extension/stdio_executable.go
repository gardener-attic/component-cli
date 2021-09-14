package extension

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/gardener/component-cli/pkg/transport/process"
)

type stdIOExecutable struct {
	processor *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.Reader
}

// NewStdIOExecutable runs resource processor in the background.
// It communicates with this processor via stdin/stdout pipes.
func NewStdIOExecutable(ctx context.Context, bin string, args ...string) (process.ResourceStreamProcessor, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("unable to start processor: %w", err)
	}

	e := stdIOExecutable{
		processor: cmd,
		stdin:     stdin,
		stdout:    stdout,
	}

	return &e, nil
}

func (e *stdIOExecutable) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	_, err := io.Copy(e.stdin, r)
	if err != nil {
		return fmt.Errorf("unable to write input: %w", err)
	}

	err = e.stdin.Close()
	if err != nil {
		return fmt.Errorf("unable to close input writer: %w", err)
	}

	_, err = io.Copy(w, e.stdout)
	if err != nil {
		return fmt.Errorf("unable to read output: %w", err)
	}

	err = e.processor.Wait()
	if err != nil {
		return fmt.Errorf("unable to stop processor: %w", err)
	}

	return nil
}
