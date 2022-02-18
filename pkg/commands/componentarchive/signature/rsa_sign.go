package signature

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdv2Sign "github.com/gardener/component-spec/bindings-go/apis/v2/signatures"
	"github.com/gardener/component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	ociopts "github.com/gardener/component-cli/ociclient/options"
	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/gardener/component-cli/pkg/logger"
	"github.com/gardener/component-cli/pkg/signatures"
)

type SignOptions struct {
	// BaseUrl is the oci registry where the component is stored.
	BaseUrl string
	// ComponentName is the unique name of the component in the registry.
	ComponentName string
	// Version is the component Version in the oci registry.
	Version string

	// SignatureName selects the matching signature to verify
	SignatureName string

	// PathToPrivateKey for RSA signing
	PathToPrivateKey string

	UploadBaseUrlForSigned string

	//RecursiveSigning to sign and upload all referenced components
	RecursiveSigning bool

	//SkipSigning to skip signing and only add digests to cds
	SkipSigning bool

	//SkipAccessTypes defines the access types that will be ignored for signing
	SkipAccessTypes []string

	// OciOptions contains all exposed options to configure the oci client.
	OciOptions ociopts.Options
}

// NewGetCommand shows definitions and their configuration.
func NewRSASignCommand(ctx context.Context) *cobra.Command {
	opts := &SignOptions{}
	cmd := &cobra.Command{
		Use:   "rsa-sign BASE_URL COMPONENT_NAME VERSION",
		Args:  cobra.ExactArgs(3),
		Short: "fetch the component descriptor from a oci registry and sign it",
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

func (o *SignOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	repoCtx := cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: cdv2.OCIRegistryType,
		},
		BaseURL: o.BaseUrl,
	}

	ociClient, cache, err := o.OciOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}

	cdresolver := cdoci.NewResolver(ociClient)
	cd, blobResolver, err := cdresolver.ResolveWithBlobResolver(ctx, &repoCtx, o.ComponentName, o.Version)
	if err != nil {
		return fmt.Errorf("unable to to fetch component descriptor %s:%s: %w", o.ComponentName, o.Version, err)
	}

	blobResolvers := map[string]ctf.BlobResolver{}
	blobResolvers[fmt.Sprintf("%s:%s", cd.Name, cd.Version)] = blobResolver

	signedCds, err := signatures.RecursivelyAddDigestsToCd(cd, repoCtx, ociClient, blobResolvers, context.TODO(), o.SkipAccessTypes)
	if err != nil {
		return fmt.Errorf("failed adding digests to cd: %w", err)
	}

	targetRepoCtx := cdv2.NewOCIRegistryRepository(o.UploadBaseUrlForSigned, "")

	if o.RecursiveSigning {
		for _, signedCd := range signedCds {
			if !o.SkipSigning {
				signer, err := cdv2Sign.CreateRsaSignerFromKeyFile(o.PathToPrivateKey)
				if err != nil {
					return fmt.Errorf("failed creating rsa signer: %w", err)
				}
				hasher, err := cdv2Sign.HasherForName("sha256")
				if err != nil {
					return fmt.Errorf("failed creating hasher: %w", err)
				}

				if err := cdv2Sign.SignComponentDescriptor(signedCd, signer, *hasher, o.SignatureName); err != nil {
					return fmt.Errorf("failed signing component descriptor: %w", err)
				}
				logger.Log.Info(fmt.Sprintf("CD Signed %s %s", o.ComponentName, o.Version))
			}

			logger.Log.Info(fmt.Sprintf("Uploading to %s %s %s", o.UploadBaseUrlForSigned, signedCd.Name, signedCd.Version))

			if err := signatures.UploadCDPreservingLocalOciBlobs(ctx, *signedCd, *targetRepoCtx, ociClient, cache, blobResolvers, log); err != nil {
				return fmt.Errorf("failed uploading cd: %w", err)
			}
		}
	} else {
		if err := signatures.UploadCDPreservingLocalOciBlobs(ctx, *cd, *targetRepoCtx, ociClient, cache, blobResolvers, log); err != nil {
			return fmt.Errorf("failed uploading cd: %w", err)
		}
	}

	return nil
}

func (o *SignOptions) Complete(args []string) error {
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

	if o.PathToPrivateKey == "" {
		return errors.New("a path to private key file must be given as --keyfile flag")
	}

	if o.UploadBaseUrlForSigned == "" {
		return errors.New("a new upload-base-url is required to upload component-desriptor")
	}
	if o.SignatureName == "" {
		return errors.New("a signature name must be provided")
	}
	return nil
}

func (o *SignOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.SignatureName, "signature-name", "", "name of the signature to verify")
	fs.StringVar(&o.PathToPrivateKey, "keyfile", "", "path to private key file")
	fs.StringVar(&o.UploadBaseUrlForSigned, "upload-base-url", "", "target repository context to upload the signed cd")
	fs.StringSliceVar(&o.SkipAccessTypes, "skip-access-types", []string{}, "comma separeted list of access types that will not be digested and signed")
	fs.BoolVar(&o.RecursiveSigning, "recursive", false, "recursively sign and upload all referenced cds")
	fs.BoolVar(&o.SkipSigning, "skip-signing", false, "skip the signing to only add digests")
	o.OciOptions.AddFlags(fs)
}
