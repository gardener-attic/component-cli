// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/gardener/component-cli/pkg/transport/process"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

const processorName = "test-processor"

type ProcessorHandlerFunc func(io.Reader, io.WriteCloser)

type Server struct {
	listener net.Listener
	quit     chan interface{}
	wg       sync.WaitGroup
	handler  ProcessorHandlerFunc
}

func NewServer(addr string, h ProcessorHandlerFunc) (*Server, error) {
	l, err := net.Listen("unix", addr)
	if err != nil {
		return nil, err
	}
	s := &Server{
		quit:     make(chan interface{}),
		listener: l,
		handler:  h,
	}
	return s, nil
}

func (s *Server) Start() {
	s.wg.Add(1)
	go s.serve()
}

func (s *Server) serve() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Println("accept error", err)
			}
		} else {
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				s.handler(conn, conn)
			}()
		}
	}
}

func (s *Server) Stop() {
	close(s.quit)
	if err := s.listener.Close(); err != nil {
		println(err)
	}
	s.wg.Wait()
}

func main() {
	addr := flag.String("addr", "", "")
	flag.Parse()

	if *addr == "" {
		// if addr is not set, use stdin/stdout for communication
		if err := ProcessorRoutine(os.Stdin, os.Stdout); err != nil {
			log.Fatal(err)
		}
		return
	}

	h := func(r io.Reader, w io.WriteCloser) {
		if err := ProcessorRoutine(r, w); err != nil {
			log.Fatal(err)
		}
	}

	srv, err := NewServer(*addr, h)
	if err != nil {
		log.Fatal(err)
	}

	srv.Start()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	srv.Stop()
}

func ProcessorRoutine(inputStream io.Reader, outputStream io.WriteCloser) error {
	defer outputStream.Close()

	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer tmpfile.Close()

	if _, err := io.Copy(tmpfile, inputStream); err != nil {
		return err
	}

	if _, err := tmpfile.Seek(0, io.SeekStart); err != nil {
		return err
	}

	cd, res, resourceBlobReader, err := process.ReadProcessorMessage(tar.NewReader(tmpfile))
	if err != nil {
		return err
	}
	if resourceBlobReader != nil {
		defer resourceBlobReader.Close()
	}

	buf := bytes.NewBuffer([]byte{})
	if _, err := io.Copy(buf, resourceBlobReader); err != nil {
		return err
	}
	outputData := fmt.Sprintf("%s\n%s", buf.String(), processorName)

	l := cdv2.Label{
		Name:  "processor-name",
		Value: json.RawMessage(`"` + processorName + `"`),
	}
	res.Labels = append(res.Labels, l)

	if err := process.WriteProcessorMessage(*cd, res, strings.NewReader(outputData), tar.NewWriter(outputStream)); err != nil {
		return err
	}

	return nil
}
