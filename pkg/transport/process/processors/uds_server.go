// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package processors

import (
	"io"
	"log"
	"net"
	"sync"
)

type ProcessorHandlerFunc func(io.Reader, io.WriteCloser)

type UDSServer struct {
	listener net.Listener
	quit     chan interface{}
	wg       sync.WaitGroup
	handler  ProcessorHandlerFunc
}

func NewUDSServer(addr string, h ProcessorHandlerFunc) (*UDSServer, error) {
	l, err := net.Listen("unix", addr)
	if err != nil {
		return nil, err
	}
	s := &UDSServer{
		quit:     make(chan interface{}),
		listener: l,
		handler:  h,
	}
	return s, nil
}

func (s *UDSServer) Start() {
	s.wg.Add(1)
	go s.serve()
}

func (s *UDSServer) serve() {
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

func (s *UDSServer) Stop() {
	close(s.quit)
	if err := s.listener.Close(); err != nil {
		println(err)
	}
	s.wg.Wait()
}
