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
	"github.com/gardener/component-cli/pkg/componentarchive"
	"github.com/gardener/component-cli/pkg/components"
	"github.com/gardener/component-cli/pkg/logger"
	"github.com/gardener/component-cli/pkg/signatures"
)

type NotarySignOptions struct {
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

	//UploadBaseUrlForSigned is the repository context to which the signed cd will be pushed
	UploadBaseUrlForSigned string

	// OciOptions contains all exposed options to configure the oci client.
	OciOptions ociopts.Options
}

// NewGetCommand shows definitions and their configuration.
func NewNotarySignCommand(ctx context.Context) *cobra.Command {
	opts := &NotarySignOptions{}
	cmd := &cobra.Command{
		Use:   "notary-sign BASE_URL COMPONENT_NAME VERSION",
		Args:  cobra.ExactArgs(3),
		Short: "fetch the component descriptor from a oci registry and sign it with notary",
		Long: `
fetches the component-descriptor and sign it with notary.
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
	repoCtx := cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: cdv2.OCIRegistryType,
		},
		BaseURL: o.BaseUrl,
	}

	//TODO: disable caching!!!!!!!
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

	_, err = signatures.RecursivelyAddDigestsToCd(cd, repoCtx, ociClient, blobResolvers, context.TODO(), []string{})
	if err != nil {
		return fmt.Errorf("failed adding adding digests to cd: %w", err)
	}
	signer, err := CreateNotarySignerFromConfig(o.PathToNotaryConfig)
	if err != nil {
		return fmt.Errorf("failed creating notary signer: %w", err)
	}
	hasher, err := cdv2Sign.HasherForName("sha256")
	if err != nil {
		return fmt.Errorf("failed creating hasher: %w", err)
	}

	if err = cdv2Sign.SignComponentDescriptor(cd, signer, *hasher, o.SignatureName); err != nil {
		return fmt.Errorf("failed signing component-descriptor: %w", err)
	}
	logger.Log.Info(fmt.Sprintf("CD Signed - Uploading to %s %s %s", o.UploadBaseUrlForSigned, o.ComponentName, o.Version))

	builderOptions := componentarchive.BuilderOptions{
		BaseUrl:   o.UploadBaseUrlForSigned,
		Name:      o.ComponentName,
		Version:   o.Version,
		Overwrite: true,
	}
	archive, err := builderOptions.Build(fs)
	if err != nil {
		return fmt.Errorf("unable to build component archive: %w", err)
	}
	archive.ComponentDescriptor = cd
	// update repository context
	if len(o.BaseUrl) != 0 {
		if err := cdv2.InjectRepositoryContext(archive.ComponentDescriptor, cdv2.NewOCIRegistryRepository(o.UploadBaseUrlForSigned, "")); err != nil {
			return fmt.Errorf("unable to add repository context to component descriptor: %w", err)
		}
	}
	manifest, err := cdoci.NewManifestBuilder(cache, archive).Build(ctx)
	if err != nil {
		return fmt.Errorf("unable to build oci artifact for component acrchive: %w", err)
	}
	ref, err := components.OCIRef(archive.ComponentDescriptor.GetEffectiveRepositoryContext(), archive.ComponentDescriptor.Name, archive.ComponentDescriptor.Version)
	if err != nil {
		return fmt.Errorf("invalid component reference: %w", err)
	}
	if err := ociClient.PushManifest(ctx, ref, manifest); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Successfully uploaded component descriptor at %q", ref))

	return nil
}

func (o *NotarySignOptions) Complete(args []string) error {
	// todo: validate args
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

	if o.PathToNotaryConfig == "" {
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

func (o *NotarySignOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.SignatureName, "signature-name", "", "name of the signature to verify")
	fs.StringVar(&o.PathToNotaryConfig, "config", "", "path to config file")
	fs.StringVar(&o.UploadBaseUrlForSigned, "upload-base-url", "", "version to upload signed cd")
	o.OciOptions.AddFlags(fs)
}
