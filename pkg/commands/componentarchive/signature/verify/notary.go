package verify

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

type NotaryVerifyOptions struct {

	// PathToNotaryConfig for notary config with url and jwt key
	PathToNotaryConfig string

	GenericVerifyOptions
}

// NewGetCommand shows definitions and their configuration.
func NewNotaryVerifyCommand(ctx context.Context) *cobra.Command {
	opts := &NotaryVerifyOptions{}
	cmd := &cobra.Command{
		Use:   "notary BASE_URL COMPONENT_NAME VERSION",
		Args:  cobra.ExactArgs(3),
		Short: "[EXPERIMENTAL] fetch the component descriptor from a oci registry and verify its integrity",
		Long: `
[EXPERIMENTAL] fetches the component-descriptor and sign it.
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

func (o *NotaryVerifyOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	verifier, err := signatures.CreateNotaryVerifierFromConfig(o.PathToNotaryConfig)
	if err != nil {
		return fmt.Errorf("failed creating notry verifier: %w", err)
	}
	if err := o.GenericVerifyOptions.VerifyWithVerifier(ctx, log, fs, verifier); err != nil {
		return fmt.Errorf("failed verifying cd: %w", err)
	}
	return nil
}

func (o *NotaryVerifyOptions) Complete(args []string) error {
	if err := o.GenericVerifyOptions.Complete(args); err != nil {
		return err
	}
	if o.PathToNotaryConfig == "" {
		return errors.New("a path to a notary config file must be provided")
	}

	return nil
}

func (o *NotaryVerifyOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.PathToNotaryConfig, "config", "", "path to config file")
	fs.StringVar(&o.SignatureName, "signature-name", "", "name of the signature to verify")
	o.OciOptions.AddFlags(fs)
}
