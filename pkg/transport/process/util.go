package process

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"sigs.k8s.io/yaml"
)

const (
	componentDescriptorFile = "component-descriptor.yaml"
	resourceFile            = "resource.yaml"
	resourceBlobFile        = "resource-blob"
)

// WriteTARArchive writes the component descriptor, resource and resource blob to a TAR archive
func WriteTARArchive(ctx context.Context, cd cdv2.ComponentDescriptor, res cdv2.Resource, resourceBlobReader io.Reader, outArchive *tar.Writer) error {
	defer outArchive.Close()

	marshaledCD, err := yaml.Marshal(cd)
	if err != nil {
		return fmt.Errorf("unable to marshal component descriptor: %w", err)
	}

	err = writeFileToTARArchive(componentDescriptorFile, bytes.NewReader(marshaledCD), outArchive)
	if err != nil {
		return fmt.Errorf("unable to write %s: %w", componentDescriptorFile, err)
	}

	marshaledRes, err := yaml.Marshal(res)
	if err != nil {
		return fmt.Errorf("unable to marshal resource: %w", err)
	}

	err = writeFileToTARArchive(resourceFile, bytes.NewReader(marshaledRes), outArchive)
	if err != nil {
		return fmt.Errorf("unable to write %s: %w", resourceFile, err)
	}

	if resourceBlobReader != nil {
		err = writeFileToTARArchive(resourceBlobFile, resourceBlobReader, outArchive)
		if err != nil {
			return fmt.Errorf("unable to write %s: %w", resourceBlobFile, err)
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

	_, err = io.Copy(tempfile, contentReader)
	if err != nil {
		return fmt.Errorf("unable to write content to file: %w", err)
	}

	_, err = tempfile.Seek(0, 0)
	if err != nil {
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

	if err = outArchive.WriteHeader(&header); err != nil {
		return fmt.Errorf("unable to write tar header: %w", err)
	}

	_, err = io.Copy(outArchive, tempfile)
	if err != nil {
		return fmt.Errorf("unable to write file to tar archive: %w", err)
	}

	return nil
}

// ReadTARArchive reads the component descriptor, resource and resource blob from a TAR archive.
// The resource blob reader can be nil. If a non-nil value is returned, it must be closed by the caller.
func ReadTARArchive(r *tar.Reader) (*cdv2.ComponentDescriptor, cdv2.Resource, io.ReadCloser, error) {
	var cd *cdv2.ComponentDescriptor
	var res cdv2.Resource
	var f *os.File

	for {
		header, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to read tar header: %w", err)
		}

		switch header.Name {
		case resourceFile:
			res, err = readResource(r)
			if err != nil {
				return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to read %s: %w", resourceFile, err)
			}
		case componentDescriptorFile:
			cd, err = readComponentDescriptor(r)
			if err != nil {
				return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to read %s: %w", componentDescriptorFile, err)
			}
		case resourceBlobFile:
			f, err = ioutil.TempFile("", "")
			if err != nil {
				return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to create tempfile: %w", err)
			}
			_, err = io.Copy(f, r)
			if err != nil {
				return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to read %s: %w", resourceBlobFile, err)
			}
		}
	}

	if f != nil {
		_, err := f.Seek(0, 0)
		if err != nil {
			return nil, cdv2.Resource{}, nil, fmt.Errorf("unable to seek to beginning of file: %w", err)
		}
	}

	return cd, res, f, nil
}

func readResource(r *tar.Reader) (cdv2.Resource, error) {
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

func readComponentDescriptor(r *tar.Reader) (*cdv2.ComponentDescriptor, error) {
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
