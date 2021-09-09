package transport

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	ociopts "github.com/gardener/component-cli/ociclient/options"
	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/gardener/component-cli/pkg/logger"
	"github.com/gardener/component-cli/pkg/transport/pipeline"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"
)

const (
	parallelRuns = 1
	targetCtx    = "o.ingress.js-ek.hubforplay.shoot.canary.k8s-hana.ondemand.com/js-transport-test"
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
	repoCtx := cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: cdv2.OCIRegistryType,
		},
		BaseURL:              o.BaseUrl,
		ComponentNameMapping: cdv2.ComponentNameMapping(o.ComponentNameMapping),
	}
	ociRef, err := cdoci.OCIRef(repoCtx, o.ComponentName, o.Version)
	if err != nil {
		return fmt.Errorf("invalid component reference: %w", err)
	}

	ociClient, _, err := o.OCIOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}

	cdresolver := cdoci.NewResolver(ociClient)
	cd, err := cdresolver.Resolve(ctx, &repoCtx, o.ComponentName, o.Version)
	if err != nil {
		return fmt.Errorf("unable to to fetch component descriptor %s: %w", ociRef, err)
	}

	cds := []*cdv2.ComponentDescriptor{}
	for i := 0; i < parallelRuns; i++ {
		cds = append(cds, cd)
	}

	wg := sync.WaitGroup{}

	targetCtx := cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: cdv2.OCIRegistryType,
		},
		BaseURL:              targetCtx,
		ComponentNameMapping: cdv2.ComponentNameMapping(o.ComponentNameMapping),
	}

	for _, cd := range cds {
		for _, resource := range cd.Resources {
			resource := resource

			wg.Add(1)
			go func() {
				defer wg.Done()

				pip, err := pipeline.NewSequentialPipeline(ociClient, targetCtx)
				if err != nil {
					log.Error(err, "unable to build pipeline")
				}

				processedCD, processedRes, err := pip.Process(ctx, cd, resource)
				if err != nil {
					log.Error(err, "unable to process resource")
				}

				mcd, err := yaml.Marshal(processedCD)
				if err != nil {
					log.Error(err, "unable to marshal cd")
				}

				mres, err := yaml.Marshal(processedRes)
				if err != nil {
					log.Error(err, "unable to marshal res")
				}

				fmt.Println(string(mcd))
				fmt.Println(string(mres))
			}()
		}
	}

	fmt.Println("waiting for goroutines to finish")
	wg.Wait()
	fmt.Println("avg_duration =", pipeline.TotalTime/time.Millisecond/parallelRuns, "ms")
	fmt.Println("main finished")

	return nil
}
