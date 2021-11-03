// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	"github.com/gardener/component-cli/pkg/transport/process/extensions"
	"github.com/gardener/component-cli/pkg/transport/process/utils"
)

const processorName = "example-processor"

// a test processor which adds its name to the resource labels and the resource blob.
// the resource blob is expected to be plain text data.
func main() {
	addr := os.Getenv(extensions.ServerAddressEnv)

	if addr == "" {
		// if addr is not set, use stdin/stdout for communication
		if err := processorRoutine(os.Stdin, os.Stdout); err != nil {
			log.Fatal(err)
		}
		return
	}

	h := func(r io.Reader, w io.WriteCloser) {
		if err := processorRoutine(r, w); err != nil {
			log.Fatal(err)
		}
	}

	srv, err := utils.NewUDSServer(addr, h)
	if err != nil {
		log.Fatal(err)
	}

	srv.Start()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	srv.Stop()
}

func processorRoutine(inputStream io.Reader, outputStream io.WriteCloser) error {
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

	cd, res, resourceBlobReader, err := utils.ReadProcessorMessage(tmpfile)
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

	if err := utils.WriteProcessorMessage(*cd, res, strings.NewReader(outputData), outputStream); err != nil {
		return err
	}

	return nil
}
