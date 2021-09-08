package processor

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type stdIOExecutable struct {
	processor *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.Reader
}

func NewStdIOExecutable(bin string) (ResourceStreamProcessor, error) {
	cmd := exec.Command(bin)
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
		return err
	}

	_, err = io.Copy(w, e.stdout)
	if err != nil {
		return fmt.Errorf("unable to read output: %w", err)
	}

	return nil
}
