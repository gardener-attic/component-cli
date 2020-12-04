// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/gardener/component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/gardener/component-cli/ociclient/credentials/secretserver"
	"github.com/gardener/component-cli/pkg/logger"
	"github.com/gardener/component-cli/pkg/utils"
)

type pushOptions struct {
	// CTFPath is the path to the directory containing the ctf archive.
	CTFPath string
	// AllowPlainHttp allows the fallback to http if the oci registry does not support https
	AllowPlainHttp bool

	// CacheDir defines the oci cache directory
	CacheDir string
	// RegistryConfigPath defines a path to the dockerconfig.json with the oci registry authentication.
	RegistryConfigPath string
	// ConcourseConfigPath is the path to the local concourse config file.
	ConcourseConfigPath string
}

// NewPushCommand creates a new definition command to push definitions
func NewPushCommand(ctx context.Context) *cobra.Command {
	opts := &pushOptions{}
	cmd := &cobra.Command{
		Use:   "push [ctf-path]",
		Args:  cobra.RangeArgs(1, 4),
		Short: "Pushes all archives of a ctf to a remote repository",
		Long: `
Push pushes all component archives and oci artifacts to the defined oci repository.

The oci repository is automatically determined based on the component/artifact descriptor (repositoryContext, component name and version).

Note: Currently only component archives are supoprted. Generic OCI Artifacts will be supported in the future.
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

			fmt.Print("Successfully uploaded ctf\n")
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *pushOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	info, err := fs.Stat(o.CTFPath)
	if err != nil {
		return fmt.Errorf("unable to get info for %s: %w", o.CTFPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf(`%q is a directory. 
It is expected that the given path points to a CTF Archive`, o.CTFPath)
	}

	cache, err := cache.NewCache(log, cache.WithBasePath(o.CacheDir))
	if err != nil {
		return err
	}

	ociOpts := []ociclient.Option{
		ociclient.WithCache{Cache: cache},
		ociclient.WithKnownMediaType(cdoci.ComponentDescriptorConfigMimeType),
		ociclient.WithKnownMediaType(cdoci.ComponentDescriptorTarMimeType),
		ociclient.WithKnownMediaType(cdoci.ComponentDescriptorJSONMimeType),
		ociclient.AllowPlainHttp(o.AllowPlainHttp),
	}
	if len(o.RegistryConfigPath) != 0 {
		keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{o.RegistryConfigPath})
		if err != nil {
			return fmt.Errorf("unable to create keyring for registry at %q: %w", o.RegistryConfigPath, err)
		}
		ociOpts = append(ociOpts, ociclient.WithKeyring(keyring))
	} else {
		keyring, err := secretserver.New().
			WithFS(fs).
			FromPath(o.ConcourseConfigPath).
			WithMinPrivileges(secretserver.ReadWrite).
			Build()
		if err != nil {
			return fmt.Errorf("unable to get credentils from secret server: %s", err.Error())
		}
		if keyring != nil {
			ociOpts = append(ociOpts, ociclient.WithKeyring(keyring))
		}
	}

	ociClient, err := ociclient.NewClient(log, ociOpts...)
	if err != nil {
		return err
	}

	ctfArchive, err := ctf.NewCTF(fs, o.CTFPath)
	if err != nil {
		return fmt.Errorf("unable to open ctf at %q: %s", o.CTFPath, err.Error())
	}

	err = ctfArchive.Walk(func(ca *ctf.ComponentArchive) error {
		manifest, err := cdoci.NewManifestBuilder(cache, ca).Build(ctx)
		if err != nil {
			return fmt.Errorf("unable to build oci artifact for component acrchive: %w", err)
		}

		ref, err := cdoci.OCIRef(ca.ComponentDescriptor.GetEffectiveRepositoryContext(), ca.ComponentDescriptor.GetName(), ca.ComponentDescriptor.GetVersion())
		if err != nil {
			return fmt.Errorf("unable to calculate oci ref for %q: %s", ca.ComponentDescriptor.GetName(), err.Error())
		}
		if err := ociClient.PushManifest(ctx, ref, manifest); err != nil {
			return fmt.Errorf("unable to upload component archive to %q: %s", ref, err.Error())
		}
		log.Info(fmt.Sprintf("Successfully uploaded component archive to %q", ref))
		return nil
	})
	if err != nil {
		return fmt.Errorf("error while reading component archives in ctf: %w", err)
	}

	return ctfArchive.Close()
}

func (o *pushOptions) Complete(args []string) error {
	o.CTFPath = args[0]

	var err error
	o.CacheDir, err = utils.CacheDir()
	if err != nil {
		return fmt.Errorf("unable to get oci cache directory: %w", err)
	}

	if err := o.Validate(); err != nil {
		return err
	}

	return nil
}

// Validate validates push options
func (o *pushOptions) Validate() error {
	if len(o.CTFPath) == 0 {
		return errors.New("a path to the component descriptor must be defined")
	}

	if len(o.CacheDir) == 0 {
		return errors.New("a oci cache directory must be defined")
	}

	// todo: validate references exist
	return nil
}

func (o *pushOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.AllowPlainHttp, "allow-plain-http", false, "allows the fallback to http if the oci registry does not support https")
	fs.StringVar(&o.RegistryConfigPath, "registry-config", "", "path to the dockerconfig.json with the oci registry authentication information")
	fs.StringVar(&o.ConcourseConfigPath, "cc-config", "", "path to the local concourse config file")
}
