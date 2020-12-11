// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/apis/v2/cdutils"
	cdvalidation "github.com/gardener/component-spec/bindings-go/apis/v2/validation"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/validation/field"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/gardener/component-cli/pkg/commands/componentarchive/input"
	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/gardener/component-cli/pkg/logger"
)

// Options defines the options that are used to add resources to a component descriptor
type Options struct {
	// ComponentArchivePath is the path to the component descriptor
	ComponentArchivePath string

	// either components can be added by a yaml resource template or by input flags

	// SourceObjectPath defines the path to the resources defined as yaml or json
	SourceObjectPath string
}

// SourceOptions contains options that are used to describe a source
type SourceOptions struct {
	cdv2.Source
	Input *input.BlobInput `json:"input,omitempty"`
}

// NewAddCommand creates a command to add additional resources to a component descriptor.
func NewAddCommand(ctx context.Context) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Adds a source to a component descriptor",
		Long: `
add adds sources to the defined component descriptor.
The sources can be defined in a file or given through stdin.

The source definitions are expected to be a multidoc yaml of the following form

<pre>

---
name: 'myrepo'
type: 'git'
access:
  type: "git"
  repository: github.com/gardener/component-cli
...
---
name: 'myconfig'
type: 'json'
input:
  type: "file"
  path: "some/path"
...
---
name: 'myothersrc'
type: 'json'
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

	archiveFs, err := projectionfs.New(fs, o.ComponentArchivePath)
	if err != nil {
		return fmt.Errorf("unable to create projectionfilesystem: %w", err)
	}
	archive, err := ctf.NewComponentArchiveFromFilesystem(archiveFs)
	if err != nil {
		return fmt.Errorf("unable to parse component archive from %s: %w", compDescFilePath, err)
	}

	sources, err := o.generateSources(fs)
	if err != nil {
		return err
	}

	for _, src := range sources {
		if src.Input != nil {
			log.Info(fmt.Sprintf("add input blob from %q", src.Input.Path))
			if err := o.addInputBlob(fs, archive, src); err != nil {
				return err
			}
		} else {
			id := archive.ComponentDescriptor.GetSourceIndex(src.Source)
			if id != -1 {
				mergedSrc := cdutils.MergeSources(archive.ComponentDescriptor.Sources[id], src.Source)
				if errList := cdvalidation.ValidateSource(field.NewPath(""), mergedSrc); len(errList) != 0 {
					return fmt.Errorf("invalid component reference: %w", errList.ToAggregate())
				}
				archive.ComponentDescriptor.Sources[id] = mergedSrc
			} else {
				if errList := cdvalidation.ValidateSource(field.NewPath(""), src.Source); len(errList) != 0 {
					return fmt.Errorf("invalid component reference: %w", errList.ToAggregate())
				}
				archive.ComponentDescriptor.Sources = append(archive.ComponentDescriptor.Sources, src.Source)
			}
		}
		log.V(3).Info(fmt.Sprintf("Successfully added source %q to component descriptor", src.Name))
	}

	if err := cdvalidation.Validate(archive.ComponentDescriptor); err != nil {
		return fmt.Errorf("invalid component descriptor: %w", err)
	}

	data, err := yaml.Marshal(archive.ComponentDescriptor)
	if err != nil {
		return fmt.Errorf("unable to encode component descriptor: %w", err)
	}
	if err := vfs.WriteFile(fs, compDescFilePath, data, 06444); err != nil {
		return fmt.Errorf("unable to write modified comonent descriptor: %w", err)
	}
	fmt.Printf("Successfully added component references to component descriptor")
	return nil
}

func (o *Options) Complete(args []string) error {

	// default component path to env var
	if len(o.ComponentArchivePath) == 0 {
		o.ComponentArchivePath = filepath.Dir(os.Getenv(constants.ComponentDescriptorPathEnvName))
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
	set.StringVar(&o.ComponentArchivePath, "comp-desc", "", "path to the component descriptor directory")

	// specify the resource
	set.StringVarP(&o.SourceObjectPath, "resource", "r", "", "The path to the resources defined as yaml or json")
}

// generateSources parses component references from the given path and stdin.
func (o *Options) generateSources(fs vfs.FileSystem) ([]SourceOptions, error) {
	sources := make([]SourceOptions, 0)
	if len(o.SourceObjectPath) != 0 {
		resourceObjectReader, err := fs.Open(o.SourceObjectPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read resource object from %s: %w", o.SourceObjectPath, err)
		}
		defer resourceObjectReader.Close()
		sources, err = generateSourcesFromReader(resourceObjectReader)
		if err != nil {
			return nil, fmt.Errorf("unable to read sources from %s: %w", o.SourceObjectPath, err)
		}
	}

	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("unable to read from stdin: %w", err)
	}
	if (stdinInfo.Mode()&os.ModeNamedPipe != 0) || stdinInfo.Size() != 0 {
		stdinSources, err := generateSourcesFromReader(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("unable to read from stdin: %w", err)
		}
		sources = append(sources, stdinSources...)
	}
	return sources, nil
}

// generateSourcesFromReader generates a resource given resource options and a resource template file.
func generateSourcesFromReader(reader io.Reader) ([]SourceOptions, error) {
	sources := make([]SourceOptions, 0)
	yamldecoder := yamlutil.NewYAMLOrJSONDecoder(reader, 1024)
	for {
		src := SourceOptions{}
		if err := yamldecoder.Decode(&src); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("unable to decode src: %w", err)
		}
		sources = append(sources, src)
	}

	return sources, nil
}

func (o *Options) addInputBlob(fs vfs.FileSystem, archive *ctf.ComponentArchive, src SourceOptions) error {
	blob, err := src.Input.Read(fs, o.SourceObjectPath)
	if err != nil {
		return err
	}

	err = archive.AddSource(&src.Source, ctf.BlobInfo{
		MediaType: src.Type,
		Digest:    blob.Digest,
		Size:      blob.Size,
	}, blob.Reader)
	if err != nil {
		blob.Reader.Close()
		return fmt.Errorf("unable to add input blob to archive: %w", err)
	}
	if err := blob.Reader.Close(); err != nil {
		return fmt.Errorf("unable to close input file: %w", err)
	}
	return nil
}
