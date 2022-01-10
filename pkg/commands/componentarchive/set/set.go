// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package set

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	cdvalidation "github.com/gardener/component-spec/bindings-go/apis/v2/validation"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/pkg/componentarchive"
	"github.com/gardener/component-cli/pkg/logger"
)

// Options defines the options that are used to add resources to a component descriptor
type Options struct {
	componentarchive.BuilderOptions
}

// NewSetCommand creates a command to add additional resources to a component descriptor.
func NewSetCommand(ctx context.Context) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "set COMPONENT_ARCHIVE_PATH [options...]",
		Args:  cobra.MinimumNArgs(1),
		Short: "set some component descriptor properties",
		Long: `
the set command sets some component descriptor properies like the component name and/or version.

The component archive can be specified by the first argument, the flag "--archive" or as env var "COMPONENT_ARCHIVE_PATH".
The component archive is expected to be a filesystem archive.
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

	o.Modify = true
	archive, err := o.BuilderOptions.Build(fs)
	if err != nil {
		return err
	}

	if len(o.Name) != 0 {
		archive.ComponentDescriptor.Name = o.Name
	}
	if len(o.Version) != 0 {
		archive.ComponentDescriptor.Version = o.Version
	}

	if err := cdvalidation.Validate(archive.ComponentDescriptor); err != nil {
		return fmt.Errorf("invalid component descriptor: %w", err)
	}

	data, err := yaml.Marshal(archive.ComponentDescriptor)
	if err != nil {
		return fmt.Errorf("unable to encode component descriptor: %w", err)
	}
	if err := vfs.WriteFile(fs, compDescFilePath, data, 0664); err != nil {
		return fmt.Errorf("unable to write modified comonent descriptor: %w", err)
	}
	log.V(2).Info("Successfully changed component descriptor")
	return nil
}

func (o *Options) Complete(args []string) error {
	if len(args) == 0 {
		return errors.New("at least a component archive path argument has to be defined")
	}
	o.BuilderOptions.ComponentArchivePath = args[0]
	o.BuilderOptions.Default()

	return o.validate()
}

func (o *Options) validate() error {
	return o.BuilderOptions.Validate()
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.BuilderOptions.AddFlags(fs)
}
