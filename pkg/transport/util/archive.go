package util

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"sigs.k8s.io/yaml"
)

const (
	ResourceFile            = "resource.yaml"
	ComponentDescriptorFile = "component-descriptor.yaml"
	BlobFile                = "blob"
)

func WriteFile(fname string, content io.Reader, outArchive *tar.Writer) error {
	tmpfile, err := ioutil.TempFile("", "tmp")
	if err != nil {
		return fmt.Errorf("unable to create tempfile: %w", err)
	}
	defer tmpfile.Close()

	_, err = io.Copy(tmpfile, content)
	if err != nil {
		return fmt.Errorf("unable to write content to tempfile: %w", err)
	}

	_, err = tmpfile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("unable to seek to beginning of tempfile: %w", err)
	}

	fstat, err := tmpfile.Stat()
	if err != nil {
		return fmt.Errorf("unable to get file stats: %w", err)
	}

	header := tar.Header{
		Name:    fname,
		Size:    fstat.Size(),
		Mode:    int64(fstat.Mode()),
		ModTime: time.Now(),
	}

	if err = outArchive.WriteHeader(&header); err != nil {
		return fmt.Errorf("unable to write tar header: %w", err)
	}

	_, err = io.Copy(outArchive, tmpfile)
	if err != nil {
		return fmt.Errorf("unable to write file to archive: %w", err)
	}

	return nil
}

func WriteArchive(ctx context.Context, cd *cdv2.ComponentDescriptor, res cdv2.Resource, resourceBlobReader io.Reader, outwriter *tar.Writer) error {
	defer outwriter.Close()

	println("start writing data")

	marshaledCD, err := yaml.Marshal(cd)
	if err != nil {
		return fmt.Errorf("unable to marshal component descriptor: %w", err)
	}

	println("writing component descriptor")
	err = WriteFile(ComponentDescriptorFile, bytes.NewReader(marshaledCD), outwriter)
	if err != nil {
		return fmt.Errorf("unable to write component descriptor: %w", err)
	}

	marshaledRes, err := yaml.Marshal(res)
	if err != nil {
		return fmt.Errorf("unable to marshal resource: %w", err)
	}

	println("writing resource")
	err = WriteFile(ResourceFile, bytes.NewReader(marshaledRes), outwriter)
	if err != nil {
		return fmt.Errorf("unable to write resource: %w", err)
	}

	if resourceBlobReader != nil {
		println("writing blob")
		err = WriteFile(BlobFile, resourceBlobReader, outwriter)
		if err != nil {
			return fmt.Errorf("unable to write blob: %w", err)
		}
	}

	println("finished writing data")

	return nil
}

func ReadArchive(r *tar.Reader) (*cdv2.ComponentDescriptor, cdv2.Resource, io.ReadCloser, error) {
	var cd *cdv2.ComponentDescriptor
	var res cdv2.Resource

	for {
		header, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, cdv2.Resource{}, nil, fmt.Errorf("%w", err)
		}

		switch header.Name {
		case ResourceFile:
			res, err = ParseResource(r)
			if err != nil {
				return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to read %s: %w", ResourceFile, err)
			}
		case ComponentDescriptorFile:
			cd, err = ParseComponentDescriptor(r)
			if err != nil {
				return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to read %s: %w", ComponentDescriptorFile, err)
			}
		}
	}

	return cd, res, nil, nil
}

func ParseResource(r *tar.Reader) (cdv2.Resource, error) {
	buf := bytes.NewBuffer([]byte{})
	_, err := io.Copy(buf, r)
	if err != nil {
		return cdv2.Resource{}, fmt.Errorf("unable to read from stream: %w", err)
	}

	var res cdv2.Resource
	err = yaml.Unmarshal(buf.Bytes(), &res)
	if err != nil {
		return cdv2.Resource{}, fmt.Errorf("unable to unmarshal: %w", err)
	}

	return res, nil
}

func ParseComponentDescriptor(r *tar.Reader) (*cdv2.ComponentDescriptor, error) {
	buf := bytes.NewBuffer([]byte{})
	_, err := io.Copy(buf, r)
	if err != nil {
		return nil, fmt.Errorf("unable to read from stream: %w", err)
	}

	var cd cdv2.ComponentDescriptor
	err = yaml.Unmarshal(buf.Bytes(), &cd)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal: %w", err)
	}

	return &cd, nil
}
