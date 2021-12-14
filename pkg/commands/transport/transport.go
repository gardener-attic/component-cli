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

	DryRun bool

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
	fs.BoolVar(&o.DryRun, "dry-run", false, "only download component descriptors and perform matching of resources against transport config file. no component descriptors are uploaded, no resources are down/uploaded")
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
	if o.DryRun {
		log.Info("dry-run: no component descriptors are uploaded, no resources are down/uploaded")
	}

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
		repoCtxOverride, err = utils.ParseRepositoryContextOverrideConfig(o.RepoCtxOverrideCfgPath)
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

	cds, err := ResolveRecursive(ctx, ociClient, *sourceCtx, o.ComponentName, o.Version, repoCtxOverride, log)
	if err != nil {
		return fmt.Errorf("unable to resolve component: %w", err)
	}

	df := transport_config.NewDownloaderFactory(ociClient, ociCache)
	pf := transport_config.NewProcessorFactory(ociCache)
	uf := transport_config.NewUploaderFactory(ociClient, ociCache, *targetCtx)
	pjf, err := transport_config.NewProcessingJobFactory(*transportCfg, df, pf, uf)
	if err != nil {
		return fmt.Errorf("unable to create processing job factory: %w", err)
	}

	if o.DryRun {
		for _, cd := range cds {
			componentLog := log.WithValues("component-name", cd.Name, "component-version", cd.Version)
			for _, res := range cd.Resources {
				resourceLog := componentLog.WithValues("resource-name", res.Name, "resource-version", res.Version)
				job, err := pjf.Create(*cd, res)
				if err != nil {
					resourceLog.Error(err, "unable to create processing job")
					return err
				}
				resourceLog.Info("matched resource", "matching", job.GetMatching())
			}
		}
		return nil
	}

	wg := sync.WaitGroup{}
	cdLookup := map[string]*cdv2.ComponentDescriptor{}
	errs := []error{}
	errsMux := sync.Mutex{}

	for _, cd := range cds {
		cd := cd
		componentLog := log.WithValues("component-name", cd.Name, "component-version", cd.Version)

		key := fmt.Sprintf("%s:%s", cd.Name, cd.Version)
		if _, ok := cdLookup[key]; ok {
			err := errors.New("component descriptor already exists in map")
			componentLog.Error(err, "unable to add component descriptor to map")
			return err
		}
		cdLookup[key] = cd

		wg.Add(1)
		go func() {
			defer wg.Done()
			processedResources, err := processResources(ctx, cd, *targetCtx, componentLog, pjf)
			if err != nil {
				errsMux.Lock()
				errs = append(errs, err)
				errsMux.Unlock()
				return
			}

			cd.Resources = processedResources
		}()
	}

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("%d errors occurred during resource processing", len(errs))
	}

	if o.GenerateSignature {
		signer, err := signatures.CreateRsaSignerFromKeyFile(o.PrivateKeyPath)
		if err != nil {
			return fmt.Errorf("unable to create signer: %w", err)
		}

		hasher, err := signatures.HasherForName("SHA256")
		if err != nil {
			return fmt.Errorf("unable to create hasher: %w", err)
		}

		crr := func(ctx context.Context, cd cdv2.ComponentDescriptor, ref cdv2.ComponentReference) (*cdv2.DigestSpec, error) {
			key := fmt.Sprintf("%s:%s", ref.Name, ref.Version)
			cd2, ok := cdLookup[key]
			if !ok {
				return nil, fmt.Errorf("unable to find component descriptor in map: %w", err)
			}

			signature, err := signatures.SelectSignatureByName(cd2, o.SignatureName)
			if err != nil {
				return nil, fmt.Errorf("unable to get signature: %w", err)
			}

			return &signature.Digest, nil
		}

		// iterate backwards -> start with "leave" component descriptors w/o dependencies
		for i := len(cds) - 1; i >= 0; i-- {
			cd := cds[i]
			componentLog := log.WithValues("component-name", cd.Name, "component-version", cd.Version)

			if err := signatures.AddDigestsToComponentDescriptor(ctx, cd, crr, nil); err != nil {
				componentLog.Error(err, "unable to add digests to component descriptor")
				return err
			}

			if err := signatures.SignComponentDescriptor(cd, signer, *hasher, o.SignatureName); err != nil {
				componentLog.Error(err, "unable to sign component descriptor")
				return err
			}
		}
	}

	for _, cd := range cdLookup {
		cd := cd
		componentLog := log.WithValues("component-name", cd.Name, "component-version", cd.Version)

		wg.Add(1)
		go func() {
			defer wg.Done()
			manifest, err := cdoci.NewManifestBuilder(ociCache, ctf.NewComponentArchive(cd, nil)).Build(ctx)
			if err != nil {
				componentLog.Error(err, "unable to build oci artifact for component archive")
				return
			}

			ociRef, err := cdoci.OCIRef(*targetCtx, o.ComponentName, o.Version)
			if err != nil {
				componentLog.Error(err, "unable to build component descriptor oci reference")
				return
			}

			if err := ociClient.PushManifest(ctx, ociRef, manifest); err != nil {
				componentLog.Error(err, "unable to push component descriptor manifest")
				return
			}
		}()
	}

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("%d errors occurred during component descriptor uploading", len(errs))
	}

	return nil
}

func processResources(
	ctx context.Context,
	cd *cdv2.ComponentDescriptor,
	targetCtx cdv2.OCIRegistryRepository,
	log logr.Logger,
	processingJobFactory *transport_config.ProcessingJobFactory,
) ([]cdv2.Resource, error) {
	wg := sync.WaitGroup{}
	errs := []error{}
	errsMux := sync.Mutex{}
	processedResources := []cdv2.Resource{}
	resMux := sync.Mutex{}

	for _, resource := range cd.Resources {
		resource := resource
		resourceLog := log.WithValues("resource-name", resource.Name, "resource-version", resource.Version)

		wg.Add(1)
		go func() {
			defer wg.Done()

			job, err := processingJobFactory.Create(*cd, resource)
			if err != nil {
				resourceLog.Error(err, "unable to create processing job")
				errsMux.Lock()
				errs = append(errs, err)
				errsMux.Unlock()
				return
			}

			resourceLog.V(5).Info("matched resource", "matching", job.GetMatching())

			if err = job.Process(ctx); err != nil {
				resourceLog.Error(err, "unable to process resource")
				errsMux.Lock()
				errs = append(errs, err)
				errsMux.Unlock()
				return
			}

			resMux.Lock()
			processedResources = append(processedResources, *job.ProcessedResource)
			resMux.Unlock()
		}()
	}

	wg.Wait()

	if len(errs) > 0 {
		return nil, errors.New("unable to process resources")
	}

	return processedResources, nil
}

func ResolveRecursive(
	ctx context.Context,
	client ociclient.Client,
	defaultRepo cdv2.OCIRegistryRepository,
	componentName,
	componentVersion string,
	repoCtxOverrideCfg *utils.RepositoryContextOverride,
	log logr.Logger,
) ([]*cdv2.ComponentDescriptor, error) {
	componentLog := log.WithValues("component-name", componentName, "component-version", componentVersion)

	repoCtx := defaultRepo
	if repoCtxOverrideCfg != nil {
		repoCtx = *repoCtxOverrideCfg.GetRepositoryContext(componentName, defaultRepo)
		componentLog.V(7).Info("repository context after override", "repository-context", repoCtx)
	}

	cdresolver := cdoci.NewResolver(client)
	cd, err := cdresolver.Resolve(ctx, &repoCtx, componentName, componentVersion)
	if err != nil {
		componentLog.Error(err, "unable to fetch component descriptor")
		return nil, err
	}

	cds := []*cdv2.ComponentDescriptor{
		cd,
	}
	for _, ref := range cd.ComponentReferences {
		cds2, err := ResolveRecursive(ctx, client, defaultRepo, ref.ComponentName, ref.Version, repoCtxOverrideCfg, log)
		if err != nil {
			componentLog.Error(err, "unable to resolve ref", "ref", ref)
			return nil, err
		}
		cds = append(cds, cds2...)
	}

	return cds, nil
}
