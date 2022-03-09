package sign

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/component-cli/pkg/logger"
	"github.com/gardener/component-cli/pkg/signatures"
)

type SignServerSignOptions struct {
	// PathToConfig path to the config file containing sign server information
	PathToConfig string

	GenericSignOptions
}

// NewSignServerSignCommand shows definitions and their configuration.
func NewSignServerSignCommand(ctx context.Context) *cobra.Command {
	opts := &SignServerSignOptions{}
	cmd := &cobra.Command{
		Use:   "sign-server BASE_URL COMPONENT_NAME VERSION",
		Args:  cobra.ExactArgs(3),
		Short: "fetch the component descriptor from an oci registry and sign it with a signature provided from the sign server",
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

func (o *SignServerSignOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	signer, err := signatures.NewSignServerFromConfigFile(o.PathToConfig)
	if err != nil {
		return fmt.Errorf("failed creating sign server signer: %w", err)
	}
	return o.SignAndUploadWithSigner(ctx, log, fs, signer)
}

func (o *SignServerSignOptions) Complete(args []string) error {
	if err := o.GenericSignOptions.Complete(args); err != nil {
		return err
	}

	if o.PathToConfig == "" {
		return errors.New("a path to config file for sign server must be given as flag")
	}

	return nil
}

func (o *SignServerSignOptions) AddFlags(fs *pflag.FlagSet) {
	o.GenericSignOptions.AddFlags(fs)
	fs.StringVar(&o.PathToConfig, "config", "", "path to config file for sign server")
}
