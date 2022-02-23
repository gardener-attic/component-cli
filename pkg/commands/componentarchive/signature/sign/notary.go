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

type NotarySignOptions struct {
	// PathToNotaryConfig for notary config with url and jwt key
	PathToNotaryConfig string

	GenericSignOptions
}

func NewNotarySignCommand(ctx context.Context) *cobra.Command {
	opts := &NotarySignOptions{}
	cmd := &cobra.Command{
		Use:   "notary-sign BASE_URL COMPONENT_NAME VERSION",
		Args:  cobra.ExactArgs(3),
		Short: "[EXPERIMENTAL] fetch the component descriptor from a oci registry and sign it with notary",
		Long: `
[EXPERIMENTAL] fetches the component-descriptor and sign it with notary.
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

func (o *NotarySignOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	signer, err := signatures.CreateNotarySignerFromConfig(o.PathToNotaryConfig)
	if err != nil {
		return fmt.Errorf("failed creating rsa signer: %w", err)
	}
	return o.SignAndUploadWithSigner(ctx, log, fs, signer)
}

func (o *NotarySignOptions) Complete(args []string) error {
	if err := o.GenericSignOptions.Complete(args); err != nil {
		return err
	}
	if o.PathToNotaryConfig == "" {
		return errors.New("a path to private key file must be given as --keyfile flag")
	}

	return nil
}

func (o *NotarySignOptions) AddFlags(fs *pflag.FlagSet) {
	o.GenericSignOptions.AddFlags(fs)
	fs.StringVar(&o.PathToNotaryConfig, "config", "", "path to config file")
}
