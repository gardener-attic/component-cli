// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imagevector

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/gardener/component-cli/pkg/logger"
)

// AddOptions defines the options that are used to add resources defined by a image vector to a component descriptor
type AddOptions struct {
	// ComponentArchivePath is the path to the component descriptor
	ComponentArchivePath string
	// ImageVectorPath defines the path to the image vector defined as yaml or json
	ImageVectorPath string

	ParseImageOptions
}

// NewAddCommand creates a command to add additional resources to a component descriptor.
func NewAddCommand(ctx context.Context) *cobra.Command {
	opts := &AddOptions{}
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Adds all resources of a image vector to the component descriptor",
		Long: `
add parses a image vector and generates the corresponding component descriptor resources.

<pre>

images:
- name: pause-container
  sourceRepository: github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile
  repository: gcr.io/google_containers/pause-amd64
  tag: "3.1"

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

func (o *AddOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
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

	parsedResources, err := o.parseImageVector(fs)
	if err != nil {
		return err
	}

	for _, ref := range parsedResources.ComponentReferences {
		if errList := cdvalidation.ValidateComponentReference(field.NewPath(""), ref); len(errList) != 0 {
			return fmt.Errorf("invalid component reference: %w", errList.ToAggregate())
		}
		id := archive.ComponentDescriptor.GetComponentReferenceIndex(ref)
		if id != -1 {
			archive.ComponentDescriptor.ComponentReferences[id] = ref
		} else {
			archive.ComponentDescriptor.ComponentReferences = append(archive.ComponentDescriptor.ComponentReferences, ref)
		}
		log.V(3).Info(fmt.Sprintf("Successfully added component references %q to component descriptor", ref.Name))
	}

	for _, res := range parsedResources.Resources {
		if errList := cdvalidation.ValidateResource(field.NewPath(""), res); len(errList) != 0 {
			return fmt.Errorf("invalid resource: %w", errList.ToAggregate())
		}
		id := archive.ComponentDescriptor.GetResourceIndex(res)
		if id != -1 {
			archive.ComponentDescriptor.Resources[id] = res
		} else {
			archive.ComponentDescriptor.Resources = append(archive.ComponentDescriptor.Resources, res)
		}
		log.V(3).Info(fmt.Sprintf("Successfully added resource %q to component descriptor", res.Name))
	}

	if parsedResources.GenericDependencies != nil {
		archive.ComponentDescriptor.Labels = cdutils.SetRawLabel(archive.ComponentDescriptor.Labels,
			parsedResources.GenericDependencies.Name, parsedResources.GenericDependencies.Value)
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

func (o *AddOptions) Complete(args []string) error {

	// default component path to env var
	if len(o.ComponentArchivePath) == 0 {
		o.ComponentArchivePath = filepath.Dir(os.Getenv(constants.ComponentDescriptorPathEnvName))
	}

	return o.validate()
}

func (o *AddOptions) validate() error {
	if len(o.ComponentArchivePath) == 0 {
		return errors.New("component descriptor path must be provided")
	}
	return nil
}

func (o *AddOptions) AddFlags(set *pflag.FlagSet) {
	set.StringVar(&o.ComponentArchivePath, "comp-desc", "", "path to the component descriptor directory")
	set.StringVar(&o.ImageVectorPath, "image-vector", "", "The path to the resources defined as yaml or json")
	set.StringArrayVar(&o.ParseImageOptions.ComponentReferencePrefixes, "component-prefixes", []string{}, "Specify all prefixes that define a image  from another component")
}

// parseImageVector parses the given image vector and returns a list of all resources.
func (o *AddOptions) parseImageVector(fs vfs.FileSystem) (*CAResources, error) {
	file, err := fs.Open(o.ImageVectorPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open image vector file: %q: %w", o.ImageVectorPath, err)
	}
	defer file.Close()
	return ParseImageVector(file, &o.ParseImageOptions)
}
