package signature

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdv2Sign "github.com/gardener/component-spec/bindings-go/apis/v2/signatures"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"gopkg.in/yaml.v2"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	ociopts "github.com/gardener/component-cli/ociclient/options"
	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/gardener/component-cli/pkg/logger"
)

type VerifyOptions struct {
	// BaseUrl is the oci registry where the component is stored.
	BaseUrl string
	// ComponentName is the unique name of the component in the registry.
	ComponentName string
	// Version is the component Version in the oci registry.
	Version string

	// SignatureName selects the matching signature to verify
	SignatureName string

	// PathToPublicKey for RSA signing
	PathToPublicKey string

	// OciOptions contains all exposed options to configure the oci client.
	OciOptions ociopts.Options
}

// NewGetCommand shows definitions and their configuration.
func NewRSAVerifyCommand(ctx context.Context) *cobra.Command {
	opts := &VerifyOptions{}
	cmd := &cobra.Command{
		Use:   "rsa-verify BASE_URL COMPONENT_NAME VERSION",
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

func (o *VerifyOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
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

	verifier, err := cdv2Sign.CreateRsaVerifierFromKeyFile(o.PathToPublicKey)
	if err != nil {
		return fmt.Errorf("failed creating rsa verifier: %w", err)
	}

	// check if digest is signed by author with public key
	if err = cdv2Sign.VerifySignedComponentDescriptor(cd, verifier, o.SignatureName); err != nil {
		return fmt.Errorf("signature invalid for digest: %w", err)
	}

	// check if digest matches the component-descriptor
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

func checkCd(cd *cdv2.ComponentDescriptor, repoContext cdv2.OCIRegistryRepository, ociClient cdoci.Client, ctx context.Context) error {
	for _, reference := range cd.ComponentReferences {
		ociRef, err := cdoci.OCIRef(repoContext, reference.Name, reference.Version)
		if err != nil {
			return fmt.Errorf("invalid component reference: %w", err)
		}

		cdresolver := cdoci.NewResolver(ociClient)
		childCd, err := cdresolver.Resolve(ctx, &repoContext, reference.ComponentName, reference.Version)
		if err != nil {
			return fmt.Errorf("unable to to fetch component descriptor %s: %w", ociRef, err)
		}

		digest, err := recursivelyCheckCds(childCd, repoContext, ociClient, ctx)
		if err != nil {
			return fmt.Errorf("unable to resolve component reference to %s:%s: %w", reference.ComponentName, reference.Version, err)
		}
		if reference.Digest == nil || reference.Digest.HashAlgorithm == "" || reference.Digest.NormalisationAlgorithm == "" || reference.Digest.Value == "" {
			return fmt.Errorf("component reference is missing digest %s:%s", reference.ComponentName, reference.Version)
		} else {
			if reference.Digest.HashAlgorithm != digest.HashAlgorithm || reference.Digest.NormalisationAlgorithm != digest.NormalisationAlgorithm || reference.Digest.Value != digest.Value {
				return fmt.Errorf("component reference digest is different to stored one %s:%s", reference.ComponentName, reference.Version)
			}
		}
	}
	for _, resource := range cd.Resources {
		switch resource.Access.Type {
		case cdv2.OCIRegistryType:
			hasher, err := cdv2Sign.HasherForName("sha256")
			if err != nil {
				return fmt.Errorf("failed creating hasher: %w", err)
			}
			ociRegistryAccess := cdv2.OCIRegistryAccess{}
			resource.Access.DecodeInto(&ociRegistryAccess)
			//TODO: make stable, use oci digest for tag
			manifest, err := ociClient.GetManifest(ctx, ociRegistryAccess.ImageReference)
			if err != nil {
				return fmt.Errorf("failed resolving manifest: %w", err)
			}
			manifestBytes, err := yaml.Marshal(manifest)
			if err != nil {
				return fmt.Errorf("failed converting manifest back to yaml bytes: %w", err)
			}

			hasher.HashFunction.Reset()
			if _, err = hasher.HashFunction.Write(manifestBytes); err != nil {
				return fmt.Errorf("failed hashing yaml, %w", err)

			}
			digest := &cdv2.DigestSpec{
				HashAlgorithm:          hasher.AlgorithmName,
				NormalisationAlgorithm: string(cdv2.ManifestDigestV1),
				Value:                  hex.EncodeToString((hasher.HashFunction.Sum(nil))),
			}
			if resource.Digest == nil || resource.Digest.HashAlgorithm == "" || resource.Digest.NormalisationAlgorithm == "" || resource.Digest.Value == "" {
				return fmt.Errorf("resource is missing digest %s:%s", resource.Name, resource.Version)
			} else {
				if resource.Digest.HashAlgorithm != digest.HashAlgorithm || resource.Digest.NormalisationAlgorithm != digest.NormalisationAlgorithm || resource.Digest.Value != digest.Value {
					return fmt.Errorf("resource digest is different to stored one %s:%s", resource.Name, resource.Version)
				}
			}

		default:
			return fmt.Errorf("access type %s not supported", resource.Access.Type)
		}
	}
	return nil
}

func recursivelyCheckCds(cd *cdv2.ComponentDescriptor, repoContext cdv2.OCIRegistryRepository, ociClient cdoci.Client, ctx context.Context) (*cdv2.DigestSpec, error) {
	for referenceIndex, reference := range cd.ComponentReferences {
		ociRef, err := cdoci.OCIRef(repoContext, reference.Name, reference.Version)
		if err != nil {
			return nil, fmt.Errorf("invalid component reference: %w", err)
		}

		cdresolver := cdoci.NewResolver(ociClient)
		childCd, err := cdresolver.Resolve(ctx, &repoContext, reference.ComponentName, reference.Version)
		if err != nil {
			return nil, fmt.Errorf("unable to to fetch component descriptor %s: %w", ociRef, err)
		}

		digest, err := recursivelyCheckCds(childCd, repoContext, ociClient, ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve component reference to %s:%s: %w", reference.ComponentName, reference.Version, err)
		}
		reference.Digest = digest
		cd.ComponentReferences[referenceIndex] = reference
	}
	for resourceIndex, resource := range cd.Resources {
		switch resource.Access.Type {
		case cdv2.OCIRegistryType:
			hasher, err := cdv2Sign.HasherForName("sha256")
			if err != nil {
				return nil, fmt.Errorf("failed creating hasher: %w", err)
			}
			ociRegistryAccess := cdv2.OCIRegistryAccess{}
			resource.Access.DecodeInto(&ociRegistryAccess)
			//TODO: make stable, use oci digest for tag
			manifest, err := ociClient.GetManifest(ctx, ociRegistryAccess.ImageReference)
			if err != nil {
				return nil, fmt.Errorf("failed resolving manifest: %w", err)
			}
			manifestBytes, err := yaml.Marshal(manifest)
			if err != nil {
				return nil, fmt.Errorf("failed converting manifest back to yaml bytes: %w", err)
			}

			hasher.HashFunction.Reset()
			if _, err = hasher.HashFunction.Write(manifestBytes); err != nil {
				return nil, fmt.Errorf("failed hashing yaml, %w", err)

			}
			digest := &cdv2.DigestSpec{
				HashAlgorithm:          hasher.AlgorithmName,
				NormalisationAlgorithm: string(cdv2.ManifestDigestV1),
				Value:                  hex.EncodeToString((hasher.HashFunction.Sum(nil))),
			}
			resource.Digest = digest
			cd.Resources[resourceIndex] = resource

		default:
			return nil, fmt.Errorf("access type %s not supported", resource.Access.Type)
		}
	}
	hasher, err := cdv2Sign.HasherForName("sha256")
	if err != nil {
		return nil, fmt.Errorf("failed creating hasher: %w", err)
	}
	hashCd, err := cdv2Sign.HashForComponentDescriptor(*cd, *hasher)
	if err != nil {
		return nil, fmt.Errorf("failed hashing cd %s:%s: %w", cd.Name, cd.Version, err)
	}
	return hashCd, nil
}

func (o *VerifyOptions) Complete(args []string) error {
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

	if o.PathToPublicKey == "" {
		return errors.New("a path to public key file must be given as --keyfile flag")
	}

	if o.SignatureName == "" {
		return errors.New("a signature name must be provided")
	}
	return nil
}

func (o *VerifyOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.PathToPublicKey, "keyfile", "", "path to public key file")
	fs.StringVar(&o.SignatureName, "signature-name", "", "name of the signature to verify")
	o.OciOptions.AddFlags(fs)
}
