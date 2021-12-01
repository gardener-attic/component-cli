// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package transport

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	ociopts "github.com/gardener/component-cli/ociclient/options"
	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/gardener/component-cli/pkg/logger"
	transport_config "github.com/gardener/component-cli/pkg/transport/config"
	"github.com/gardener/component-cli/pkg/utils"
)

type Options struct {
	SourceRepository string
	TargetRepository string

	// ComponentName is the unique name of the component in the registry.
	ComponentName string
	// Version is the component Version in the oci registry.
	Version string

	// TransportCfgPath is the path to the transport config file
	TransportCfgPath string
	// RepoCtxOverrideCfgPath is the path to the repository context override config file
	RepoCtxOverrideCfgPath string

	// OCIOptions contains all oci client related options.
	OCIOptions ociopts.Options
}

// NewTransportCommand creates a new transport command.
func NewTransportCommand(ctx context.Context) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use: "transport",
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

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.SourceRepository, "from", "", "source repository base url")
	fs.StringVar(&o.TargetRepository, "to", "", "target repository where the components are copied to")
	fs.StringVar(&o.TransportCfgPath, "transport-cfg", "", "path to the transport config file")
	fs.StringVar(&o.RepoCtxOverrideCfgPath, "repo-ctx-override-cfg", "", "path to the repository context override config file")
	o.OCIOptions.AddFlags(fs)
}

func (o *Options) Complete(args []string) error {
	o.ComponentName = args[0]
	o.Version = args[1]

	cliHomeDir, err := constants.CliHomeDir()
	if err != nil {
		return err
	}
	o.OCIOptions.CacheDir = filepath.Join(cliHomeDir, "components")
	if err := os.MkdirAll(o.OCIOptions.CacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create cache directory %s: %w", o.OCIOptions.CacheDir, err)
	}

	if len(o.ComponentName) == 0 {
		return errors.New("a component name must be defined")
	}
	if len(o.Version) == 0 {
		return errors.New("a component's Version must be defined")
	}

	if len(o.SourceRepository) == 0 {
		return errors.New("the base url must be defined")
	}
	if len(o.TransportCfgPath) == 0 {
		return errors.New("a path to a transport config file must be defined")
	}

	return nil
}

func (o *Options) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	ociClient, _, err := o.OCIOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}

	ociCache, err := cache.NewCache(log, cache.WithBasePath(o.OCIOptions.CacheDir))
	if err != nil {
		return fmt.Errorf("unable to build cache: %w", err)
	}

	if err := cache.InjectCacheInto(ociClient, ociCache); err != nil {
		return fmt.Errorf("unable to inject cache into oci client: %w", err)
	}

	repoCtxOverrideCfg, err := utils.ParseRepositoryContextConfig(o.RepoCtxOverrideCfgPath)
	if err != nil {
		return fmt.Errorf("unable to parse repository context override config file: %w", err)
	}

	transportCfg, err := transport_config.ParseConfig(o.TransportCfgPath)
	if err != nil {
		return fmt.Errorf("unable to parse transport config file: %w", err)
	}

	sourceCtx := cdv2.NewOCIRegistryRepository(o.SourceRepository, "")
	targetCtx := cdv2.NewOCIRegistryRepository(o.TargetRepository, "")

	cds, err := ResolveRecursive(ctx, ociClient, *sourceCtx, o.ComponentName, o.Version, *repoCtxOverrideCfg)
	if err != nil {
		return fmt.Errorf("unable to resolve component: %w", err)
	}

	df := transport_config.NewDownloaderFactory(ociClient, ociCache)
	pf := transport_config.NewProcessorFactory(ociCache)
	uf := transport_config.NewUploaderFactory(ociClient, ociCache, *targetCtx)
	pjf, err := transport_config.NewProcessingJobFactory(*transportCfg, df, pf, uf)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	for _, cd := range cds {
		cd := cd
		wg.Add(1)
		go func() {
			defer wg.Done()
			processedResources, errs := handleResources(ctx, cd, *targetCtx, log, pjf)
			if len(errs) > 0 {
				for _, err := range errs {
					log.Error(err, "")
					return
				}
			}

			cd.Resources = processedResources

			manifest, err := cdoci.NewManifestBuilder(ociCache, ctf.NewComponentArchive(cd, nil)).Build(ctx)
			if err != nil {
				log.Error(err, "unable to build oci artifact for component acrchive")
				return
			}

			ociRef, err := cdoci.OCIRef(*targetCtx, o.ComponentName, o.Version)
			if err != nil {
				log.Error(err, "unable to build component reference")
				return
			}

			if err := ociClient.PushManifest(ctx, ociRef, manifest); err != nil {
				log.Error(err, "unable to push manifest")
				return
			}
		}()
	}

	wg.Wait()

	return nil
}

func handleResources(ctx context.Context, cd *cdv2.ComponentDescriptor, targetCtx cdv2.OCIRegistryRepository, log logr.Logger, processingJobFactory *transport_config.ProcessingJobFactory) ([]cdv2.Resource, []error) {
	wg := sync.WaitGroup{}
	errs := []error{}
	mux := sync.Mutex{}
	processedResources := []cdv2.Resource{}

	for _, resource := range cd.Resources {
		resource := resource

		wg.Add(1)
		go func() {
			defer wg.Done()

			job, err := processingJobFactory.Create(*cd, resource)
			if err != nil {
				errs = append(errs, fmt.Errorf("unable to create processing job: %w", err))
				return
			}

			if err = job.Process(ctx); err != nil {
				errs = append(errs, fmt.Errorf("unable to process resource %+v: %w", resource, err))
				return
			}

			mux.Lock()
			processedResources = append(processedResources, *job.ProcessedResource)
			mux.Unlock()
		}()
	}

	wg.Wait()
	return processedResources, errs
}

func ResolveRecursive(
	ctx context.Context,
	client ociclient.Client,
	defaultRepo cdv2.OCIRegistryRepository,
	componentName,
	componentVersion string,
	repoCtxOverrideCfg utils.RepositoryContextOverride,
) ([]*cdv2.ComponentDescriptor, error) {

	repoCtx := repoCtxOverrideCfg.GetRepositoryContext(componentName, defaultRepo)

	ociRef, err := cdoci.OCIRef(*repoCtx, componentName, componentVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid component reference: %w", err)
	}

	cdresolver := cdoci.NewResolver(client)
	cd, err := cdresolver.Resolve(ctx, repoCtx, componentName, componentVersion)
	if err != nil {
		return nil, fmt.Errorf("unable to to fetch component descriptor %s: %w", ociRef, err)
	}

	cds := []*cdv2.ComponentDescriptor{
		cd,
	}
	for _, ref := range cd.ComponentReferences {
		cds2, err := ResolveRecursive(ctx, client, defaultRepo, ref.ComponentName, ref.Version, repoCtxOverrideCfg)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve ref %+v: %w", ref, err)
		}
		cds = append(cds, cds2...)
	}

	return cds, nil
}
