// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package extensions

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/utils"
)

const serverAddressFlag = "--addr"

type udsExecutable struct {
	processor *exec.Cmd
	addr      string
	conn      net.Conn
}

// NewUDSExecutable runs a resource processor extension executable in the background.
// It communicates with this processor via Unix Domain Sockets.
func NewUDSExecutable(ctx context.Context, bin string, args []string, env []string) (process.ResourceStreamProcessor, error) {
	for _, arg := range args {
		if arg == serverAddressFlag {
			return nil, fmt.Errorf("the flag %s is not allowed to be set manually", serverAddressFlag)
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s/%s.sock", wd, utils.RandomString(8))
	args = append(args, "--addr", addr)

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("unable to start processor: %w", err)
	}

	conn, err := tryConnect(addr)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to processor: %w", err)
	}

	e := udsExecutable{
		processor: cmd,
		addr:      addr,
		conn:      conn,
	}

	return &e, nil
}

func (e *udsExecutable) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	if _, err := io.Copy(e.conn, r); err != nil {
		return fmt.Errorf("unable to write input: %w", err)
	}

	usock := e.conn.(*net.UnixConn)
	if err := usock.CloseWrite(); err != nil {
		return fmt.Errorf("unable to close input writer: %w", err)
	}

	if _, err := io.Copy(w, e.conn); err != nil {
		return fmt.Errorf("unable to read output: %w", err)
	}

	if err := e.processor.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("unable to send SIGTERM to processor: %w", err)
	}

	// extension servers must implement ordinary shutdown (!)
	if err := e.processor.Wait(); err != nil {
		return fmt.Errorf("unable to wait for processor: %w", err)
	}

	return nil
}

func tryConnect(addr string) (net.Conn, error) {
	const (
		maxRetries = 5
		sleeptime  = 500 * time.Millisecond
	)

	var conn net.Conn
	var err error
	for i := 0; i <= maxRetries; i++ {
		conn, err = net.Dial("unix", addr)
		if err == nil {
			break
		}

		time.Sleep(sleeptime)
	}

	return conn, err
}
