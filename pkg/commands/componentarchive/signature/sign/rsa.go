package sign

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

type RSASignOptions struct {
	// PathToPrivateKey for RSA signing
	PathToPrivateKey string

	GenericSignOptions
}

// NewGetCommand shows definitions and their configuration.
func NewRSASignCommand(ctx context.Context) *cobra.Command {
	opts := &RSASignOptions{}
	cmd := &cobra.Command{
		Use:   "rsa BASE_URL COMPONENT_NAME VERSION",
		Args:  cobra.ExactArgs(3),
		Short: "fetch the component descriptor from an oci registry and sign it using RSASSA-PKCS1-V1_5-SIGN",
		Long: `
fetches the component-descriptor and sign it.
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

func (o *RSASignOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	signer, err := cdv2Sign.CreateRsaSignerFromKeyFile(o.PathToPrivateKey)
	if err != nil {
		return fmt.Errorf("failed creating rsa signer: %w", err)
	}
	return o.SignAndUploadWithSigner(ctx, log, fs, signer)
}

func (o *RSASignOptions) Complete(args []string) error {
	if err := o.GenericSignOptions.Complete(args); err != nil {
		return err
	}

	if o.PathToPrivateKey == "" {
		return errors.New("a path to private key file must be given as --keyfile flag")
	}

	return nil
}

func (o *RSASignOptions) AddFlags(fs *pflag.FlagSet) {
	o.GenericSignOptions.AddFlags(fs)
	fs.StringVar(&o.PathToPrivateKey, "private-key", "", "path to private key file used for signing")
}
