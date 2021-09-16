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

// NewStdIOExecutable runs a resource processor extension executable in the background.
// It communicates with this processor via stdin/stdout pipes.
func NewStdIOExecutable(ctx context.Context, bin string, args []string, env []string) (process.ResourceStreamProcessor, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = env
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr

	e := stdIOExecutable{
		processor: cmd,
		stdin:     stdin,
		stdout:    stdout,
	}

	return &e, nil
}

func (e *stdIOExecutable) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	if err := e.processor.Start(); err != nil {
		return fmt.Errorf("unable to start processor: %w", err)
	}

	if _, err := io.Copy(e.stdin, r); err != nil {
		return fmt.Errorf("unable to write input: %w", err)
	}

	if err := e.stdin.Close(); err != nil {
		return fmt.Errorf("unable to close input writer: %w", err)
	}

	if _, err := io.Copy(w, e.stdout); err != nil {
		return fmt.Errorf("unable to read output: %w", err)
	}

	if err := e.processor.Wait(); err != nil {
		return fmt.Errorf("unable to stop processor: %w", err)
	}

	return nil
}
