// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentreferences

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
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

const KeyValueAssignment = "="

// Options defines the options that are used to add resources to a component descriptor
type Options struct {
	// ComponentDescriptorPath is the path to the component descriptor
	ComponentDescriptorPath string
	// OutputPath is the path where the modified component descriptor should be written to.
	// BY default the component-descriptor input path is the output path
	OutputPath string

	// either components can be added by a yaml resource template or by input flags

	// ComponentReferenceObjectPath defines the path to the resources defined as yaml or json
	ComponentReferenceObjectPath string
	// ComponentReferenceOptions options contains configuration for a resources that is defined by flags
	ComponentReferenceOptions ComponentReferenceOptions

	// InputPath specifies the path to the blob that should be added
	InputPath string
	// CreateTar configures if the input path should be automatically tarred
	CreateTar bool
	// CompressWithGzip configures if the input path should be automatically gzipped
	CompressWithGzip bool
}

// ComponentReferenceOptions contains options that are used to describe a resource
type ComponentReferenceOptions struct {
	Name          string
	ComponentName string
	Version       string
	ExtraIdentity []string
	Labels        []string
}

// NewAddCommand creates a command to add additional resources to a component descriptor.
func NewAddCommand(ctx context.Context) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:     "add",
		Example: "component-cli add",
		Short:   "add a resource to an existing component descriptor",
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
	compDescFilePath := filepath.Join(o.ComponentDescriptorPath, ctf.ComponentDescriptorFileName)

	// add the input to the ctf format
	archiveFs, err := projectionfs.New(fs, o.ComponentDescriptorPath)
	if err != nil {
		return fmt.Errorf("unable to create projectionfilesystem: %w", err)
	}
	archive, err := ctf.NewComponentArchiveFromFilesystem(archiveFs)
	if err != nil {
		return fmt.Errorf("unable to parse component archive from %s: %w", compDescFilePath, err)
	}

	ref, err := o.generateComponentReference(fs, archive.ComponentDescriptor)
	if err != nil {
		return err
	}

	if errList := cdvalidation.ValidateComponentReference(field.NewPath(""), ref); len(errList) != 0 {
		return errList.ToAggregate()
	}
	// validate the resource
	id := archive.ComponentDescriptor.GetComponentReferenceIndex(ref)
	if id != -1 {
		archive.ComponentDescriptor.ComponentReferences[id] = ref
	} else {
		archive.ComponentDescriptor.ComponentReferences = append(archive.ComponentDescriptor.ComponentReferences, ref)
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
	fmt.Printf("Successfully added resource %q to component descriptor", ref.Name)
	return nil
}

func (o *Options) Complete(args []string) error {

	// default component path to env var
	if len(o.ComponentDescriptorPath) == 0 {
		o.ComponentDescriptorPath = filepath.Dir(os.Getenv(constants.ComponentDescriptorPathEnvName))
	}
	if len(o.OutputPath) == 0 {
		o.OutputPath = o.ComponentDescriptorPath
	}

	return o.validate()
}

func (o *Options) validate() error {
	if len(o.ComponentDescriptorPath) == 0 {
		return errors.New("component descriptor path must be provided")
	}
	return nil
}

func (o *Options) AddFlags(set *pflag.FlagSet) {
	set.StringVar(&o.ComponentDescriptorPath, "comp-desc", "", "path to the component descriptor directory")
	set.StringVarP(&o.OutputPath, "out", "o", "", "path where the modified component descriptor should be written to")

	// specify the resource
	set.StringVarP(&o.ComponentReferenceObjectPath, "resource", "r", "", "The path to the resources defined as yaml or json")
	// specify the resource as component
	set.StringVar(&o.ComponentReferenceOptions.Name, "name", "", "logical name of the reference to be added")
	set.StringVar(&o.ComponentReferenceOptions.ComponentName, "componentName", "", "name of the reference to be added")
	set.StringVar(&o.ComponentReferenceOptions.Version, "version", "", "version of the reference to be added")
	set.StringArrayVar(&o.ComponentReferenceOptions.ExtraIdentity, "identity", []string{},
		"Key-value string pairs that define the extra identity of the resource. The key-value pairs are of the form '<key>=<value>'")
	set.StringArrayVar(&o.ComponentReferenceOptions.Labels, "label", []string{},
		"Key-value string pairs that define additional labels. The key-value pairs are of the form '<key>=<value>' ")
}

// generateResource generates a resource given resource options and a resource template file.
func (o *Options) generateComponentReference(fs vfs.FileSystem, cd *cdv2.ComponentDescriptor) (cdv2.ComponentReference, error) {
	componentReference := cdv2.ComponentReference{}
	if len(o.ComponentReferenceObjectPath) != 0 {
		refData, err := vfs.ReadFile(fs, o.ComponentReferenceObjectPath)
		if err != nil {
			return cdv2.ComponentReference{}, fmt.Errorf("unable to read componentReference object from %s: %w", o.ComponentReferenceObjectPath, err)
		}
		if err := yaml.Unmarshal(refData, &componentReference); err != nil {
			return cdv2.ComponentReference{}, fmt.Errorf("unable to decode componentReference from %s: %w", o.ComponentReferenceObjectPath, err)
		}
	}

	// set componentReference options if defined
	if len(o.ComponentReferenceOptions.Name) != 0 {
		componentReference.Name = o.ComponentReferenceOptions.Name
	}
	if len(o.ComponentReferenceOptions.ComponentName) != 0 {
		componentReference.ComponentName = o.ComponentReferenceOptions.ComponentName
	}
	if len(o.ComponentReferenceOptions.Version) != 0 {
		componentReference.Version = o.ComponentReferenceOptions.Version
	}

	for _, extraID := range o.ComponentReferenceOptions.ExtraIdentity {
		keyVal := strings.Split(extraID, KeyValueAssignment)
		if len(keyVal) > 2 {
			return cdv2.ComponentReference{}, fmt.Errorf("extra identity key-value pair %q must consist of a key and avlue separated by %q", extraID, KeyValueAssignment)
		}
		if componentReference.ExtraIdentity == nil {
			componentReference.ExtraIdentity = cdv2.Identity{}
		}
		componentReference.ExtraIdentity[keyVal[0]] = strings.Join(keyVal[1:], KeyValueAssignment)
	}
	for _, label := range o.ComponentReferenceOptions.Labels {
		keyVal := strings.Split(label, KeyValueAssignment)
		if len(keyVal) > 2 {
			return cdv2.ComponentReference{}, fmt.Errorf("label key-value pair %q must consist of a key and avlue separated by %q", label, KeyValueAssignment)
		}
		if componentReference.Labels == nil {
			componentReference.Labels = make([]cdv2.Label, 0)
		}
		value := json.RawMessage{}
		if err := yaml.Unmarshal([]byte(strings.Join(keyVal[1:], KeyValueAssignment)), &value); err != nil {
			return cdv2.ComponentReference{}, fmt.Errorf("unable to parse label value of %q: %w", keyVal[1:], err)
		}
		componentReference.Labels = append(componentReference.Labels, cdv2.Label{
			Name:  keyVal[0],
			Value: value,
		})
	}
	return componentReference, nil
}
