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
	"github.com/gardener/component-spec/bindings-go/apis/v2/signatures"
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

	GenerateSignature bool
	SignatureName     string
	PrivateKeyPath    string

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
	fs.BoolVar(&o.GenerateSignature, "sign", false, "sign the uploaded component descriptors")
	fs.StringVar(&o.SignatureName, "signature-name", "", "name of the generated signature")
	fs.StringVar(&o.PrivateKeyPath, "private-key", "", "path to the private key file used for signing")
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
		return errors.New("a source repository must be defined")
	}
	if len(o.TargetRepository) == 0 {
		return errors.New("a target repository must be defined")
	}

	if len(o.TransportCfgPath) == 0 {
		return errors.New("a path to a transport config file must be defined")
	}

	if o.GenerateSignature {
		if o.SignatureName == "" {
			return errors.New("a signature name must be defined")
		}
		if o.PrivateKeyPath == "" {
			return errors.New("a path to a private key file must be defined")
		}
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

	var repoCtxOverride *utils.RepositoryContextOverride
	if o.RepoCtxOverrideCfgPath != "" {
		repoCtxOverride, err = utils.ParseRepositoryContextConfig(o.RepoCtxOverrideCfgPath)
		if err != nil {
			return fmt.Errorf("unable to parse repository context override config file: %w", err)
		}
	}

	transportCfg, err := transport_config.ParseConfig(o.TransportCfgPath)
	if err != nil {
		return fmt.Errorf("unable to parse transport config file: %w", err)
	}

	sourceCtx := cdv2.NewOCIRegistryRepository(o.SourceRepository, "")
	targetCtx := cdv2.NewOCIRegistryRepository(o.TargetRepository, "")

	cds, err := ResolveRecursive(ctx, ociClient, *sourceCtx, o.ComponentName, o.Version, repoCtxOverride)
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
	cdMap := map[string]*cdv2.ComponentDescriptor{}

	for _, cd := range cds {
		cd := cd

		key := fmt.Sprintf("%s:%s", cd.Name, cd.Version)
		if _, ok := cdMap[key]; ok {
			return fmt.Errorf("component descriptor already exists in map: %+v", cd)
		}
		cdMap[key] = cd

		wg.Add(1)
		go func() {
			defer wg.Done()
			processedResources, errs := handleResources(ctx, cd, *targetCtx, log, pjf)
			if len(errs) > 0 {
				for _, err := range errs {
					log.Error(err, "unable to process resource")
					return
				}
			}

			cd.Resources = processedResources
		}()
	}

	wg.Wait()

	if o.GenerateSignature {
		// iterate backwards -> start with "leave" component descriptors w/o dependencies
		for i := len(cds) - 1; i >= 0; i-- {
			cd := cds[i]

			crr := func(context.Context, cdv2.ComponentDescriptor, cdv2.ComponentReference) (*cdv2.DigestSpec, error) {
				key := fmt.Sprintf("%s:%s", cd.Name, cd.Version)
				cd, ok := cdMap[key]
				if !ok {
					return nil, fmt.Errorf("unable to find component descriptor in map")
				}

				signature, err := signatures.SelectSignatureByName(cd, o.SignatureName)
				if err != nil {
					return nil, fmt.Errorf("unable to get signature from component descriptor: %w", err)
				}

				return &signature.Digest, nil
			}

			signer, err := signatures.CreateRsaSignerFromKeyFile(o.PrivateKeyPath)
			if err != nil {
				return fmt.Errorf("unable to create signer: %w", err)
			}

			hasher, err := signatures.HasherForName("SHA256")
			if err != nil {
				return fmt.Errorf("unable to create hasher: %w", err)
			}

			if err := signatures.AddDigestsToComponentDescriptor(ctx, cd, crr, nil); err != nil {
				return fmt.Errorf("unable to add digests to component descriptor: %w", err)
			}

			if err := signatures.SignComponentDescriptor(cd, signer, *hasher, o.SignatureName); err != nil {
				return fmt.Errorf("unable to sign component descriptor: %w", err)
			}
		}
	}

	for _, cd := range cdMap {
		cd := cd

		wg.Add(1)
		go func() {
			defer wg.Done()
			manifest, err := cdoci.NewManifestBuilder(ociCache, ctf.NewComponentArchive(cd, nil)).Build(ctx)
			if err != nil {
				log.Error(err, "unable to build oci artifact for component archive")
				return
			}

			ociRef, err := cdoci.OCIRef(*targetCtx, o.ComponentName, o.Version)
			if err != nil {
				log.Error(err, "unable to build component descriptor oci reference")
				return
			}

			if err := ociClient.PushManifest(ctx, ociRef, manifest); err != nil {
				log.Error(err, "unable to push component descriptor manifest")
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
	errsMux := sync.Mutex{}
	processedResources := []cdv2.Resource{}
	resMux := sync.Mutex{}

	for _, resource := range cd.Resources {
		resource := resource

		wg.Add(1)
		go func() {
			defer wg.Done()

			job, err := processingJobFactory.Create(*cd, resource)
			if err != nil {
				errsMux.Lock()
				errs = append(errs, fmt.Errorf("unable to create processing job: %w", err))
				errsMux.Unlock()
				return
			}

			if err = job.Process(ctx); err != nil {
				errsMux.Lock()
				errs = append(errs, fmt.Errorf("unable to process resource %+v: %w", resource, err))
				errsMux.Unlock()
				return
			}

			resMux.Lock()
			processedResources = append(processedResources, *job.ProcessedResource)
			resMux.Unlock()
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
	repoCtxOverrideCfg *utils.RepositoryContextOverride,
) ([]*cdv2.ComponentDescriptor, error) {

	repoCtx := defaultRepo
	if repoCtxOverrideCfg != nil {
		repoCtx = *repoCtxOverrideCfg.GetRepositoryContext(componentName, defaultRepo)
	}

	ociRef, err := cdoci.OCIRef(repoCtx, componentName, componentVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid component reference: %w", err)
	}

	cdresolver := cdoci.NewResolver(client)
	cd, err := cdresolver.Resolve(ctx, &repoCtx, componentName, componentVersion)
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
