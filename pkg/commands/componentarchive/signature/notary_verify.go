package signature

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdv2Sign "github.com/gardener/component-spec/bindings-go/apis/v2/signatures"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	ociopts "github.com/gardener/component-cli/ociclient/options"
	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/gardener/component-cli/pkg/logger"
)

type NotaryVerifyOptions struct {
	// BaseUrl is the oci registry where the component is stored.
	BaseUrl string
	// ComponentName is the unique name of the component in the registry.
	ComponentName string
	// Version is the component Version in the oci registry.
	Version string

	// SignatureName selects the matching signature to verify
	SignatureName string

	// PathToNotaryConfig for notary config with url and jwt key
	PathToNotaryConfig string

	// OciOptions contains all exposed options to configure the oci client.
	OciOptions ociopts.Options
}

// NewGetCommand shows definitions and their configuration.
func NewNotaryVerifyCommand(ctx context.Context) *cobra.Command {
	opts := &NotaryVerifyOptions{}
	cmd := &cobra.Command{
		Use:   "notary-verify BASE_URL COMPONENT_NAME VERSION",
		Args:  cobra.ExactArgs(3),
		Short: "fetch the component descriptor from a oci registry and verify its integrity",
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

func (o *NotaryVerifyOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	repoCtx := cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: cdv2.OCIRegistryType,
		},
		BaseURL: o.BaseUrl,
	}

	//TODO: disable caching!!!!!!!
	ociClient, _, err := o.OciOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}

	cdresolver := cdoci.NewResolver(ociClient)
	cd, err := cdresolver.Resolve(ctx, &repoCtx, o.ComponentName, o.Version)
	if err != nil {
		return fmt.Errorf("unable to to fetch component descriptor %s:%s: %w", o.ComponentName, o.Version, err)
	}

	verifier, err := CreateNotaryVerifierFromConfig(o.PathToNotaryConfig)
	if err != nil {
		return fmt.Errorf("failed creating notry verifier: %w", err)
	}

	// check if digest is signed by author with public key
	if err = cdv2Sign.VerifySignedComponentDescriptor(cd, verifier, o.SignatureName); err != nil {
		return fmt.Errorf("signature invalid for digest: %w", err)
	}

	// check referenced resources and cds
	err = checkCd(cd, repoCtx, ociClient, context.TODO())
	if err != nil {
		return fmt.Errorf("failed checking cd: %w", err)
	}
	hasher, err := cdv2Sign.HasherForName("sha256")
	if err != nil {
		return fmt.Errorf("failed creating hasher: %w", err)
	}
	hashCd, err := cdv2Sign.HashForComponentDescriptor(*cd, *hasher)
	if err != nil {
		return fmt.Errorf("failed hashing cd %s:%s: %w", cd.Name, cd.Version, err)
	}

	matchingSignature, err := cdv2Sign.SelectSignatureByName(cd, o.SignatureName)
	if err != nil {
		return fmt.Errorf("failed selecting signature %s: %w", o.SignatureName, err)
	}

	if hashCd.HashAlgorithm != matchingSignature.Digest.HashAlgorithm || hashCd.NormalisationAlgorithm != matchingSignature.Digest.NormalisationAlgorithm || hashCd.Value != matchingSignature.Digest.Value {
		return fmt.Errorf("failed verifiying signature: signed normalised digest does not match calculated digest")
	}

	log.Info(fmt.Sprintf("Signature %s is valid and digest of normalised cd matches calculated digest", o.SignatureName))

	return nil
}

func (o *NotaryVerifyOptions) Complete(args []string) error {
	o.BaseUrl = args[0]
	o.ComponentName = args[1]
	o.Version = args[2]

	cliHomeDir, err := constants.CliHomeDir()
	if err != nil {
		return err
	}

	// TODO: disable caching
	o.OciOptions.CacheDir = filepath.Join(cliHomeDir, "components")
	if err := os.MkdirAll(o.OciOptions.CacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create cache directory %s: %w", o.OciOptions.CacheDir, err)
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

	if o.SignatureName == "" {
		return errors.New("a signature name must be provided")
	}
	return nil
}

func (o *NotaryVerifyOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.PathToNotaryConfig, "config", "", "path to config file")
	fs.StringVar(&o.SignatureName, "signature-name", "", "name of the signature to verify")
	o.OciOptions.AddFlags(fs)
}
