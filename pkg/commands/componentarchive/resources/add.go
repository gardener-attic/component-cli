// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/apis/v2/cdutils"
	cdvalidation "github.com/gardener/component-spec/bindings-go/apis/v2/validation"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/validation/field"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/gardener/component-cli/pkg/logger"
)

// Options defines the options that are used to add resources to a component descriptor
type Options struct {
	// ComponentArchivePath is the path to the component descriptor
	ComponentArchivePath string

	// either components can be added by a yaml resource template or by input flags
	// ResourceObjectPath defines the path to the resources defined as yaml or json
	ResourceObjectPath string
}

// ResourceOptions contains options that are used to describe a resource
type ResourceOptions struct {
	cdv2.Resource
	Input *BlobInput `json:"input,omitempty"`
}

type BlobInputType string

const (
	FileInputType = "file"
	DirInputType  = "dir"
)

// BlobInput defines a local resource input that should be added to the component descriptor and
// to the resource's access.
type BlobInput struct {
	// Type defines the input type of the blob to be added.
	// Note that a input blob of type "dir" is automatically tarred.
	Type BlobInputType `json:"type"`
	// Path is the path that points to the blob to be added.
	Path string `json:"path"`
	// CompressWithGzip defines that the blob should be automatically compressed using gzip.
	CompressWithGzip *bool `json:"compress,omitempty"`
}

// Compress returns if the blob should be compressed using gzip.
func (i BlobInput) Compress() bool {
	if i.CompressWithGzip == nil {
		return false
	}
	return *i.CompressWithGzip
}

// NewAddCommand creates a command to add additional resources to a component descriptor.
func NewAddCommand(ctx context.Context) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "add [component archive path] [-r resource-path]",
		Args:  cobra.RangeArgs(0, 1),
		Short: "Adds a resource to an component archive",
		Long: `
add generates resources from a resource template and adds it to the given component descriptor in the component archive.
If the resource is already defined (quality by identity) in the component-descriptor it will be overwritten.

The component archive can be specified by the first argument, the flag "--archive" or as env var "COMPONENT_ARCHIVE_PATH".
The component archive is expected to be a filesystem archive. If the archive is given as tar please use the export command.

The resource template can be defined by specifying a file with the template with "resource" or it can be given through stdin.

The resource template is a multidoc yaml file so multiple templates can be defined.

<pre>

---
name: 'myconfig'
type: 'json'
relation: 'local'
input:
  type: "file"
  path: "some/path"
...
---
name: 'myconfig'
type: 'json'
relation: 'local'
input:
  type: "dir"
  path: /my/path
  compress: true # defaults to false
  exclude: "*.txt"
...

</pre>
`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			if err := opts.Run(ctx, logger.Log, osfs.New()); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *Options) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	compDescFilePath := filepath.Join(o.ComponentArchivePath, ctf.ComponentDescriptorFileName)

	// add the input to the ctf format
	archiveFs, err := projectionfs.New(fs, o.ComponentArchivePath)
	if err != nil {
		return fmt.Errorf("unable to create projectionfilesystem: %w", err)
	}
	archive, err := ctf.NewComponentArchiveFromFilesystem(archiveFs)
	if err != nil {
		return fmt.Errorf("unable to parse component archive from %s: %w", compDescFilePath, err)
	}

	resources, err := o.generateResources(fs, archive.ComponentDescriptor)
	if err != nil {
		return err
	}

	for _, resource := range resources {
		if resource.Input != nil {
			log.Info(fmt.Sprintf("add input blob from %q", resource.Input.Path))
			if err := o.addInputBlob(fs, archive, &resource); err != nil {
				return err
			}
		} else {
			if errList := cdvalidation.ValidateResource(field.NewPath(""), resource.Resource); len(errList) != 0 {
				return errList.ToAggregate()
			}
			// validate the resource
			id := archive.ComponentDescriptor.GetResourceIndex(resource.Resource)
			if id != -1 {
				archive.ComponentDescriptor.Resources[id] = cdutils.MergeResources(archive.ComponentDescriptor.Resources[id], resource.Resource)
			} else {
				archive.ComponentDescriptor.Resources = append(archive.ComponentDescriptor.Resources, resource.Resource)
			}
		}

		if err := cdvalidation.Validate(archive.ComponentDescriptor); err != nil {
			return fmt.Errorf("invalid resource: %w", err)
		}

		data, err := yaml.Marshal(archive.ComponentDescriptor)
		if err != nil {
			return fmt.Errorf("unable to encode component descriptor: %w", err)
		}
		if err := vfs.WriteFile(fs, compDescFilePath, data, 06444); err != nil {
			return fmt.Errorf("unable to write modified comonent descriptor: %w", err)
		}
		fmt.Printf("Successfully added resource %q to component descriptor", resource.Name)
	}
	return nil
}

func (o *Options) Complete(args []string) error {

	// default component path to env var
	if len(o.ComponentArchivePath) == 0 {
		o.ComponentArchivePath = filepath.Dir(os.Getenv(constants.ComponentArchivePathEnvName))
	}

	return o.validate()
}

func (o *Options) validate() error {
	if len(o.ComponentArchivePath) == 0 {
		return errors.New("component descriptor path must be provided")
	}
	return nil
}

func (o *Options) AddFlags(set *pflag.FlagSet) {
	set.StringVarP(&o.ComponentArchivePath, "archive", "a", "", "path to the component descriptor directory")

	// specify the resource
	set.StringVarP(&o.ResourceObjectPath, "resource", "r", "", "The path to the resources defined as yaml or json")
}

func (o *Options) generateResources(fs vfs.FileSystem, cd *cdv2.ComponentDescriptor) ([]ResourceOptions, error) {
	resources := make([]ResourceOptions, 0)
	if len(o.ResourceObjectPath) != 0 {
		resourceObjectReader, err := fs.Open(o.ResourceObjectPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read resource object from %s: %w", o.ResourceObjectPath, err)
		}
		defer resourceObjectReader.Close()
		resources, err = generateResourcesFromReader(cd, resourceObjectReader)
		if err != nil {
			return nil, fmt.Errorf("unable to read resources from %s: %w", o.ResourceObjectPath, err)
		}
	}
	stdinResources, err := generateResourcesFromReader(cd, os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("unable to read from stdin: %w", err)
	}
	return append(resources, stdinResources...), nil
}

// generateResourcesFromPath generates a resource given resource options and a resource template file.
func generateResourcesFromReader(cd *cdv2.ComponentDescriptor, reader io.Reader) ([]ResourceOptions, error) {
	resources := make([]ResourceOptions, 0)
	yamldecoder := yamlutil.NewYAMLOrJSONDecoder(reader, 1024)
	for {
		resource := ResourceOptions{}
		if err := yamldecoder.Decode(&resource); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("unable to decode resource: %w", err)
		}

		// default relation to local
		if len(resource.Relation) == 0 {
			resource.Relation = cdv2.LocalRelation
		}
		// automatically set the version to the component descriptors version for local resources
		if resource.Relation == cdv2.LocalRelation && len(resource.Version) == 0 {
			resource.Version = cd.GetVersion()
		}

		if resource.Input != nil && resource.Access != nil {
			return nil, fmt.Errorf("the resources %q input and access is defiend. Only one option is allowed", resource.Name)
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (o *Options) addInputBlob(fs vfs.FileSystem, archive *ctf.ComponentArchive, resource *ResourceOptions) error {
	inputPath := resource.Input.Path
	if !filepath.IsAbs(resource.Input.Path) {
		inputPath = filepath.Join(filepath.Dir(o.ResourceObjectPath), resource.Input.Path)
	}
	inputInfo, err := fs.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("unable to get info for input blob from %q, %w", inputPath, err)
	}

	var (
		blob        io.ReadCloser
		inputDigest string
	)
	// automatically tar the input artifact if it is a directory
	if resource.Input.Type == DirInputType {
		if !inputInfo.IsDir() {
			return fmt.Errorf("resource type is dir but a file was provided")
		}
		blobFs, err := projectionfs.New(fs, inputPath)
		if err != nil {
			return fmt.Errorf("unable to create internal fs for %q: %w", inputPath, err)
		}
		var (
			data bytes.Buffer
		)
		if resource.Input.Compress() {
			gw := gzip.NewWriter(&data)
			if err := TarFileSystem(blobFs, gw); err != nil {
				return fmt.Errorf("unable to tar input artifact: %w", err)
			}
			if err := gw.Close(); err != nil {
				return fmt.Errorf("unable to close gzip writer: %w", err)
			}
		} else {
			if err := TarFileSystem(blobFs, &data); err != nil {
				return fmt.Errorf("unable to tar input artifact: %w", err)
			}
		}
		blob = ioutil.NopCloser(&data)
		inputDigest = digest.FromBytes(data.Bytes()).String()
	} else if resource.Input.Type == FileInputType {
		if inputInfo.IsDir() {
			return fmt.Errorf("resource type is file but a directory was provided")
		}
		// otherwise just open the file
		inputBlob, err := fs.Open(inputPath)
		if err != nil {
			return fmt.Errorf("unable to read input blob from %q: %w", inputPath, err)
		}
		blobDigest, err := digest.FromReader(inputBlob)
		if err != nil {
			return fmt.Errorf("unable to calculate digest for input blob from %q, %w", inputPath, err)
		}
		if _, err := inputBlob.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("unable to reset input file: %s", err)
		}
		blob = inputBlob
		inputDigest = blobDigest.String()
		if resource.Input.Compress() {
			var data bytes.Buffer
			gw := gzip.NewWriter(&data)
			if _, err := io.Copy(gw, inputBlob); err != nil {
				return fmt.Errorf("unable to compress input file %q: %w", inputPath, err)
			}
			if err := gw.Close(); err != nil {
				return fmt.Errorf("unable to close gzip writer: %w", err)
			}
			blob = ioutil.NopCloser(&data)
			inputDigest = digest.FromBytes(data.Bytes()).String()
		}
	} else {
		return fmt.Errorf("unknown input type %q", inputPath)
	}

	err = archive.AddResource(&resource.Resource, ctf.BlobInfo{
		MediaType: resource.Type,
		Digest:    inputDigest,
		Size:      inputInfo.Size(),
	}, blob)
	if err != nil {
		return fmt.Errorf("unable to add input blob to archive: %w", err)
	}
	if err := blob.Close(); err != nil {
		return fmt.Errorf("unable to close input file %q: %w", inputPath, err)
	}
	return nil
}

// TarFileSystem creates a tar archive from a filesystem.
func TarFileSystem(fs vfs.FileSystem, writer io.Writer) error {
	tw := tar.NewWriter(writer)

	err := vfs.Walk(fs, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel("/", path)
		if err != nil {
			return fmt.Errorf("unable to calculate relative path for %s: %w", path, err)
		}
		// ignore the root directory.
		if relPath == "." {
			return nil
		}
		header.Name = relPath
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("unable to write header for %q: %w", path, err)
		}
		if info.IsDir() {
			return nil
		}

		file, err := fs.OpenFile(path, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to open file %q: %w", path, err)
		}
		if _, err := io.Copy(tw, file); err != nil {
			return fmt.Errorf("unable to add file to tar %q: %w", path, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("unable to close file %q: %w", path, err)
		}
		return nil
	})
	return err
}
