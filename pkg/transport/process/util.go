// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package process

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"sigs.k8s.io/yaml"
)

const (
	// ComponentDescriptorFile is the filename of the component descriptor in a processor message tar archive
	ComponentDescriptorFile = "component-descriptor.yaml"

	// ResourceFile is the filename of the resource in a processor message tar archive
	ResourceFile = "resource.yaml"

	// ResourceBlobFile is the filename of the resource blob in a processor message tar archive
	ResourceBlobFile = "resource-blob"
)

// WriteProcessorMessage writes a component descriptor, resource and resource blob as a processor
// message (tar archive with fixed filenames for component descriptor, resource, and resource blob)
// which can be consumed by processors.
func WriteProcessorMessage(cd cdv2.ComponentDescriptor, res cdv2.Resource, resourceBlobReader io.Reader, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	marshaledCD, err := yaml.Marshal(cd)
	if err != nil {
		return fmt.Errorf("unable to marshal component descriptor: %w", err)
	}

	if err := writeFileToTARArchive(ComponentDescriptorFile, bytes.NewReader(marshaledCD), tw); err != nil {
		return fmt.Errorf("unable to write %s: %w", ComponentDescriptorFile, err)
	}

	marshaledRes, err := yaml.Marshal(res)
	if err != nil {
		return fmt.Errorf("unable to marshal resource: %w", err)
	}

	if err := writeFileToTARArchive(ResourceFile, bytes.NewReader(marshaledRes), tw); err != nil {
		return fmt.Errorf("unable to write %s: %w", ResourceFile, err)
	}

	if resourceBlobReader != nil {
		if err := writeFileToTARArchive(ResourceBlobFile, resourceBlobReader, tw); err != nil {
			return fmt.Errorf("unable to write %s: %w", ResourceBlobFile, err)
		}
	}

	return nil
}

func writeFileToTARArchive(filename string, contentReader io.Reader, outArchive *tar.Writer) error {
	tempfile, err := ioutil.TempFile("", "")
	if err != nil {
		return fmt.Errorf("unable to create tempfile: %w", err)
	}
	defer tempfile.Close()

	if _, err := io.Copy(tempfile, contentReader); err != nil {
		return fmt.Errorf("unable to write content to file: %w", err)
	}

	if _, err := tempfile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("unable to seek to beginning of file: %w", err)
	}

	fstat, err := tempfile.Stat()
	if err != nil {
		return fmt.Errorf("unable to get file info: %w", err)
	}

	header := tar.Header{
		Name:    filename,
		Size:    fstat.Size(),
		Mode:    int64(fstat.Mode()),
		ModTime: time.Now(),
	}

	if err := outArchive.WriteHeader(&header); err != nil {
		return fmt.Errorf("unable to write tar header: %w", err)
	}

	if _, err := io.Copy(outArchive, tempfile); err != nil {
		return fmt.Errorf("unable to write file to tar archive: %w", err)
	}

	return nil
}

// ReadProcessorMessage reads the component descriptor, resource and resource blob from a processor message
// (tar archive with fixed filenames for component descriptor, resource, and resource blob) which is
// produced by processors. The resource blob reader can be nil. If a non-nil value is returned, it must
// be closed by the caller.
func ReadProcessorMessage(r io.Reader) (*cdv2.ComponentDescriptor, cdv2.Resource, io.ReadCloser, error) {
	tr := tar.NewReader(r)

	var cd *cdv2.ComponentDescriptor
	var res cdv2.Resource
	var f *os.File

	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to read tar header: %w", err)
		}

		switch header.Name {
		case ResourceFile:
			if res, err = readResource(tr); err != nil {
				return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to read %s: %w", ResourceFile, err)
			}
		case ComponentDescriptorFile:
			if cd, err = readComponentDescriptor(tr); err != nil {
				return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to read %s: %w", ComponentDescriptorFile, err)
			}
		case ResourceBlobFile:
			if f, err = ioutil.TempFile("", ""); err != nil {
				return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to create tempfile: %w", err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to read %s: %w", ResourceBlobFile, err)
			}
		}
	}

	if f == nil {
		return cd, res, nil, nil
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to seek to beginning of file: %w", err)
	}

	return cd, res, f, nil
}

func readResource(r *tar.Reader) (cdv2.Resource, error) {
	buf := bytes.NewBuffer([]byte{})
	if _, err := io.Copy(buf, r); err != nil {
		return cdv2.Resource{}, fmt.Errorf("unable to read from stream: %w", err)
	}

	var res cdv2.Resource
	if err := yaml.Unmarshal(buf.Bytes(), &res); err != nil {
		return cdv2.Resource{}, fmt.Errorf("unable to unmarshal: %w", err)
	}

	return res, nil
}

func readComponentDescriptor(r *tar.Reader) (*cdv2.ComponentDescriptor, error) {
	buf := bytes.NewBuffer([]byte{})
	if _, err := io.Copy(buf, r); err != nil {
		return nil, fmt.Errorf("unable to read from stream: %w", err)
	}

	var cd cdv2.ComponentDescriptor
	if err := yaml.Unmarshal(buf.Bytes(), &cd); err != nil {
		return nil, fmt.Errorf("unable to unmarshal: %w", err)
	}

	return &cd, nil
}
