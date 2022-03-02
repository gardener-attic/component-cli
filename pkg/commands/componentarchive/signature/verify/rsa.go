package verify

import (
	"context"
	"errors"
	"fmt"
	"os"

	cdv2Sign "github.com/gardener/component-spec/bindings-go/apis/v2/signatures"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/component-cli/pkg/logger"
)

type VerifyOptions struct {
	// PathToPublicKey for RSA verification
	PathToPublicKey string

	GenericVerifyOptions
}

func NewRSAVerifyCommand(ctx context.Context) *cobra.Command {
	opts := &VerifyOptions{}
	cmd := &cobra.Command{
		Use:   "rsa BASE_URL COMPONENT_NAME VERSION",
		Args:  cobra.ExactArgs(3),
		Short: "fetch the component descriptor from an oci registry and verify its integrity based on a RSASSA-PKCS1-V1_5-SIGN signature",
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

func (o *VerifyOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	verifier, err := cdv2Sign.CreateRsaVerifierFromKeyFile(o.PathToPublicKey)
	if err != nil {
		return fmt.Errorf("failed creating rsa verifier: %w", err)
	}

	if err := o.GenericVerifyOptions.VerifyWithVerifier(ctx, log, fs, verifier); err != nil {
		return fmt.Errorf("failed verifying cd: %w", err)
	}
	return nil
}

func (o *VerifyOptions) Complete(args []string) error {
	if err := o.GenericVerifyOptions.Complete(args); err != nil {
		return err
	}
	if o.PathToPublicKey == "" {
		return errors.New("a path to public key file must be given as flag")
	}

	return nil
}

func (o *VerifyOptions) AddFlags(fs *pflag.FlagSet) {
	o.GenericVerifyOptions.AddFlags(fs)
	fs.StringVar(&o.PathToPublicKey, "public-key", "", "path to public key file")
}
