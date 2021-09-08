package processor

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/gardener/component-cli/pkg/utils"
)

type udsExecutable struct {
	processor *exec.Cmd
	addr   string
	conn   net.Conn
}

func NewUDSExecutable(bin string) (ResourceStreamProcessor, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s/%s.sock", wd, utils.RandomString(8))

	cmd := exec.Command(bin)
	cmd.Args = append(cmd.Args, "--addr", addr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// exec.CommandContext()

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("unable to start processor: %w", err)
	}

	conn, err := tryConnect(addr)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to processor: %w", err)
	}

	e := udsExecutable{
		processor: cmd,
		addr:   addr,
		conn:   conn,
	}

	return &e, nil
}

func (e *udsExecutable) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	_, err := io.Copy(e.conn, r)
	if err != nil {
		return fmt.Errorf("unable to write input: %w", err)
	}

	usock := e.conn.(*net.UnixConn)
	err = usock.CloseWrite()
	if err != nil {
		return err
	}

	_, err = io.Copy(w, e.conn)
	if err != nil {
		return fmt.Errorf("unable to read output: %w", err)
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
