package transport

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/ociclient"
	ociopts "github.com/gardener/component-cli/ociclient/options"
	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/gardener/component-cli/pkg/logger"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/downloaders"
	"github.com/gardener/component-cli/pkg/transport/process/extensions"
	"github.com/gardener/component-cli/pkg/transport/process/uploaders"
)

const (
	parallelRuns = 1
	targetCtxUrl = "eu.gcr.io/gardener-project/test/jschicktanz/target"
)

type Options struct {
	// BaseUrl is the oci registry where the component is stored.
	BaseUrl string
	// ComponentName is the unique name of the component in the registry.
	ComponentName string
	// Version is the component Version in the oci registry.
	Version string

	ComponentNameMapping string

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
	fs.StringVar(&o.ComponentNameMapping, "component-name-mapping", string(cdv2.OCIRegistryURLPathMapping), "[OPTIONAL] repository context name mapping")
	o.OCIOptions.AddFlags(fs)
}

func (o *Options) Complete(args []string) error {
	// todo: validate args
	o.BaseUrl = args[0]
	o.ComponentName = args[1]
	o.Version = args[2]

	cliHomeDir, err := constants.CliHomeDir()
	if err != nil {
		return err
	}
	o.OCIOptions.CacheDir = filepath.Join(cliHomeDir, "components")
	if err := os.MkdirAll(o.OCIOptions.CacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create cache directory %s: %w", o.OCIOptions.CacheDir, err)
	}

	if len(o.BaseUrl) == 0 {
		return errors.New("the base url must be defined")
	}
	if len(o.ComponentName) == 0 {
		return errors.New("a component name must be defined")
	}
	if len(o.Version) == 0 {
		return errors.New("a component's Version must be defined")
	}

	return nil
}

func (o *Options) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	ociClient, _, err := o.OCIOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}

	cds, err := ResolveRecursive(ctx, ociClient, o.BaseUrl, o.ComponentName, o.Version, o.ComponentNameMapping)
	if err != nil {
		return fmt.Errorf("unable to resolve component: %w", err)
	}

	targetCtx := cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: cdv2.OCIRegistryType,
		},
		BaseURL:              targetCtxUrl,
		ComponentNameMapping: cdv2.ComponentNameMapping(o.ComponentNameMapping),
	}

	wg := sync.WaitGroup{}
	for _, cd := range cds {
		cd := cd
		wg.Add(1)
		go func() {
			defer wg.Done()
			processedResources, errs := handleResources(ctx, cd, targetCtx, log, ociClient)
			if len(errs) > 0 {
				for _, err := range errs {
					log.Error(err, "")
				}
			}

			cd.Resources = processedResources
		}()
	}

	fmt.Println("waiting for goroutines to finish")
	wg.Wait()
	fmt.Println("main finished")

	return nil
}

func handleResources(ctx context.Context, cd *cdv2.ComponentDescriptor, targetCtx cdv2.OCIRegistryRepository, log logr.Logger, ociClient ociclient.Client) ([]cdv2.Resource, []error) {
	wg := sync.WaitGroup{}
	errs := []error{}
	mux := sync.Mutex{}
	processedResources := []cdv2.Resource{}

	for _, resource := range cd.Resources {
		resource := resource

		wg.Add(1)
		go func() {
			defer wg.Done()

			procs, err := createProcessors(ociClient, targetCtx)
			if err != nil {
				errs = append(errs, fmt.Errorf("unable to create processors: %w", err))
				return
			}

			pip := process.NewResourceProcessingPipeline(procs...)
			if err != nil {
				errs = append(errs, fmt.Errorf("unable to create pipeline: %w", err))
				return
			}

			// TODO: do we allow modifications of the component descriptor?
			// If so, how do we merge the possibly different output of multiple resource pipelines?
			processedCD, processedRes, err := pip.Process(ctx, *cd, resource)
			if err != nil {
				errs = append(errs, fmt.Errorf("unable to process resource: %w", err))
				return
			}

			mux.Lock()
			processedResources = append(processedResources, processedRes)
			mux.Unlock()

			mcd, err := yaml.Marshal(processedCD)
			if err != nil {
				errs = append(errs, fmt.Errorf("unable to marshal cd: %w", err))
				return
			}

			mres, err := yaml.Marshal(processedRes)
			if err != nil {
				errs = append(errs, fmt.Errorf("unable to marshal res: %w", err))
				return
			}

			fmt.Println(string(mcd))
			fmt.Println(string(mres))
		}()
	}

	wg.Wait()
	return processedResources, errs
}

func ResolveRecursive(ctx context.Context, client ociclient.Client, baseUrl, componentName, componentVersion, componentNameMapping string) ([]*cdv2.ComponentDescriptor, error) {
	repoCtx := cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: cdv2.OCIRegistryType,
		},
		BaseURL:              baseUrl,
		ComponentNameMapping: cdv2.ComponentNameMapping(componentNameMapping),
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
		cds2, err := ResolveRecursive(ctx, client, baseUrl, ref.ComponentName, ref.Version, componentNameMapping)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve ref %+v: %w", ref, err)
		}
		cds = append(cds, cds2...)
	}

	return cds, nil
}

func createProcessors(client ociclient.Client, targetCtx cdv2.OCIRegistryRepository) ([]process.ResourceStreamProcessor, error) {
	procBins := []string{
		"/Users/i500806/dev/pipeman/bin/processor_1",
		"/Users/i500806/dev/pipeman/bin/processor_2",
		"/Users/i500806/dev/pipeman/bin/processor_3",
	}

	procs := []process.ResourceStreamProcessor{
		downloaders.NewLocalOCIBlobDownloader(client),
	}

	for _, procBin := range procBins {
		exec, err := extensions.NewStdIOExecutable(procBin, []string{}, []string{})
		if err != nil {
			return nil, err
		}
		procs = append(procs, exec)
	}

	procs = append(procs, uploaders.NewLocalOCIBlobUploader(client, targetCtx))

	return procs, nil
}
