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
	"strings"
	"syscall"
	"time"

	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/utils"
)

// ServerAddressEnv is the environment variable key which is used to store the
// address under which a resource processor server should start.
const ServerAddressEnv = "SERVER_ADDRESS"

type udsExecutable struct {
	bin  string
	args []string
	env  []string
	addr string
}

// NewUDSExecutable runs a resource processor extension executable in the background.
// It communicates with this processor via Unix Domain Sockets.
func NewUDSExecutable(bin string, args []string, env []string) (process.ResourceStreamProcessor, error) {
	for _, e := range env {
		if strings.HasPrefix(e, ServerAddressEnv+"=") {
			return nil, fmt.Errorf("the env variable %s is not allowed to be set manually", ServerAddressEnv)
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s/%s.sock", wd, utils.RandomString(8))
	env = append(env, fmt.Sprintf("%s=%s", ServerAddressEnv, addr))

	e := udsExecutable{
		bin:  bin,
		args: args,
		env:  env,
		addr: addr,
	}

	return &e, nil
}

func (e *udsExecutable) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cmd := exec.CommandContext(ctx, e.bin, e.args...)
	cmd.Env = e.env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("unable to start processor: %w", err)
	}

	conn, err := tryConnect(e.addr)
	if err != nil {
		return fmt.Errorf("unable to connect to processor: %w", err)
	}

	if _, err := io.Copy(conn, r); err != nil {
		return fmt.Errorf("unable to write input: %w", err)
	}

	usock := conn.(*net.UnixConn)
	if err := usock.CloseWrite(); err != nil {
		return fmt.Errorf("unable to close input writer: %w", err)
	}

	if _, err := io.Copy(w, conn); err != nil {
		return fmt.Errorf("unable to read output: %w", err)
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("unable to send SIGTERM to processor: %w", err)
	}

	// extension servers must implement ordinary shutdown (!)
	if err := cmd.Wait(); err != nil {
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
