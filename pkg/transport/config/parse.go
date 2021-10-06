package config

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/pkg/logger"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// ParsingOptions defines all options that are used
type ParsingOptions struct {
	ConfigPath string
}

func NewConfigParseCommand(ctx context.Context) *cobra.Command {
	opts := &ParsingOptions{}
	cmd := &cobra.Command{
		Use:   "parse PATH_TO_PROCESSING_CFG",
		Args:  cobra.RangeArgs(1, 2),
		Short: "Parses a processing config.",
		Long: `
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

func (o *ParsingOptions) AddFlags(fs *pflag.FlagSet) {
}

func (o *ParsingOptions) Complete(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("a path to a config file is required")
	}
	o.ConfigPath = args[0]
	return nil
}

func (o *ParsingOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	rawConfig, err := os.ReadFile(o.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed reading config file %w", err)
	}

	var config Config
	err = yaml.Unmarshal(rawConfig, &config)
	if err != nil {
		return fmt.Errorf("failed parsing config %w", err)
	}

	compiler, err := CompileFromConfig(&config)
	if err != nil {
		return fmt.Errorf("failed creating lookup table %w", err)
	}
	fmt.Println(compiler.lookup)

	cd := []cdv2.ComponentDescriptor{
		cdv2.ComponentDescriptor{
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name: "ComponentDescirptor1",
				},
				Resources: []cdv2.Resource{
					cdv2.Resource{
						IdentityObjectMeta: cdv2.IdentityObjectMeta{
							Name: "MyResource",
						},
					},
				},
			},
		},
		cdv2.ComponentDescriptor{
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name: "ComponentDescirptor2",
				},
				Resources: []cdv2.Resource{
					cdv2.Resource{
						IdentityObjectMeta: cdv2.IdentityObjectMeta{
							Name: "MyResource",
						},
					},
				},
			},
		},
	}

	pipeline, err := compiler.CreateResourcePipeline(cd)
	if err != nil {
		return fmt.Errorf("failed creating pipeline %w", err)
	}
	fmt.Println(pipeline)
	return nil
}
