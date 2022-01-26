package signature

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdv2Sign "github.com/gardener/component-spec/bindings-go/apis/v2/signatures"
	"github.com/gardener/component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/opencontainers/go-digest"
	"gopkg.in/yaml.v2"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/component-cli/ociclient"
	ociCache "github.com/gardener/component-cli/ociclient/cache"
	ociopts "github.com/gardener/component-cli/ociclient/options"
	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/gardener/component-cli/pkg/components"
	"github.com/gardener/component-cli/pkg/logger"
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

func UploadCDPreservingLocalOciBlobs(ctx context.Context, cd v2.ComponentDescriptor, targetRepository cdv2.OCIRegistryRepository, ociClient ociclient.ExtendedClient, cache ociCache.Cache, blobResolver ctf.BlobResolver, log logr.Logger) error {
	manifest, err := cdoci.NewManifestBuilder(cache, ctf.NewComponentArchive(&cd, nil)).Build(ctx)
	if err != nil {
		return fmt.Errorf("unable to build oci artifact for component acrchive: %w", err)
	}
	if err := cdv2.InjectRepositoryContext(&cd, &targetRepository); err != nil {
		return fmt.Errorf("unble to inject target repository: %w", err)
	}

	// add all localOciBlobs to the layers
	var layers []ocispecv1.Descriptor
	blobToResource := map[string]*cdv2.Resource{}

	for _, res := range cd.Resources {
		if res.Access.Type == cdv2.LocalOCIBlobType {
			localBlob := &cdv2.LocalOCIBlobAccess{}
			if err := res.Access.DecodeInto(localBlob); err != nil {
				return fmt.Errorf("unable to decode resource %s: %w", res.Name, err)
			}
			blobInfo, err := blobResolver.Info(ctx, res)
			if err != nil {
				return fmt.Errorf("unable to get blob info for resource %s: %w", res.Name, err)
			}
			d, err := digest.Parse(blobInfo.Digest)
			if err != nil {
				return fmt.Errorf("unable to parse digest for resource %s: %w", res.Name, err)
			}
			layers = append(layers, ocispecv1.Descriptor{
				MediaType: blobInfo.MediaType,
				Digest:    d,
				Size:      blobInfo.Size,
				Annotations: map[string]string{
					"resource": res.Name,
				},
			})
			blobToResource[blobInfo.Digest] = res.DeepCopy()

		}
	}
	manifest.Layers = append(manifest.Layers, layers...)

	ref, err := components.OCIRef(&targetRepository, cd.Name, cd.Version)
	if err != nil {
		return fmt.Errorf("invalid component reference: %w", err)
	}

	store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		log := log.WithValues("digest", desc.Digest.String(), "mediaType", desc.MediaType)
		res, ok := blobToResource[desc.Digest.String()]
		if !ok {
			// default to cache
			log.V(5).Info("copying resource from cache")
			rc, err := cache.Get(desc)
			if err != nil {
				return err
			}
			defer func() {
				if err := rc.Close(); err != nil {
					log.Error(err, "unable to close blob reader")
				}
			}()
			if _, err := io.Copy(writer, rc); err != nil {
				return err
			}
			return nil
		}
		log.V(5).Info("copying resource", "resource", res.Name)
		_, err := blobResolver.Resolve(ctx, *res, writer)
		return err
	})
	log.V(3).Info("Upload component.", "ref", ref)
	if err := ociClient.PushManifest(ctx, ref, manifest, ociclient.WithStore(store)); err != nil {
		return fmt.Errorf("failed pushing manifest: %w", err)
	}
	return nil

}

func (o *SignOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
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

	if err := recursivelyAddDigestsToCd(cd, repoCtx, ociClient, context.TODO()); err != nil {
		return fmt.Errorf("failed adding digests to cd: %w", err)
	}
	signer, err := cdv2Sign.CreateRsaSignerFromKeyFile(o.PathToPrivateKey)
	if err != nil {
		return fmt.Errorf("failed creating rsa signer: %w", err)
	}
	hasher, err := cdv2Sign.HasherForName("sha256")
	if err != nil {
		return fmt.Errorf("failed creating hasher: %w", err)
	}

	if err := cdv2Sign.SignComponentDescriptor(cd, signer, *hasher, o.SignatureName); err != nil {
		return fmt.Errorf("failed signing component descriptor: %w", err)
	}
	logger.Log.Info("CD Signed - Uploading to %s %s %s", o.UploadBaseUrlForSigned, o.ComponentName, o.Version)
	targetRepoCtx := cdv2.NewOCIRegistryRepository(o.UploadBaseUrlForSigned, "")

	if err := UploadCDPreservingLocalOciBlobs(ctx, *cd, *targetRepoCtx, ociClient, cache, blobResolver, log); err != nil {
		return fmt.Errorf("failed uploading cd: %w", err)
	}

	return nil
}

func recursivelyAddDigestsToCd(cd *cdv2.ComponentDescriptor, repoContext cdv2.OCIRegistryRepository, ociClient cdoci.Client, ctx context.Context) error {
	cdResolver := func(c context.Context, cd cdv2.ComponentDescriptor, cr cdv2.ComponentReference) (*cdv2.DigestSpec, error) {
		ociRef, err := cdoci.OCIRef(repoContext, cr.Name, cr.Version)
		if err != nil {
			return nil, fmt.Errorf("invalid component reference: %w", err)
		}

		cdresolver := cdoci.NewResolver(ociClient)
		childCd, err := cdresolver.Resolve(ctx, &repoContext, cr.ComponentName, cr.Version)
		if err != nil {
			return nil, fmt.Errorf("unable to to fetch component descriptor %s: %w", ociRef, err)
		}
		err = recursivelyAddDigestsToCd(childCd, repoContext, ociClient, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed resolving referenced cd %s:%s: %w", cr.Name, cr.Version, err)
		}
		hasher, err := cdv2Sign.HasherForName("sha256")
		if err != nil {
			return nil, fmt.Errorf("failed creating hasher: %w", err)
		}
		hashCd, err := cdv2Sign.HashForComponentDescriptor(*childCd, *hasher)
		if err != nil {
			return nil, fmt.Errorf("failed hashing referenced cd %s:%s: %w", cr.Name, cr.Version, err)
		}
		return hashCd, nil

	}
	resResolver := func(c context.Context, cd cdv2.ComponentDescriptor, cr cdv2.Resource) (*cdv2.DigestSpec, error) {
		switch cr.Access.Type {
		case cdv2.OCIRegistryType:
			hasher, err := cdv2Sign.HasherForName("sha256")
			if err != nil {
				return nil, fmt.Errorf("failed creating hasher: %w", err)
			}
			ociRegistryAccess := cdv2.OCIRegistryAccess{}
			cr.Access.DecodeInto(&ociRegistryAccess)
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
			return &cdv2.DigestSpec{
				HashAlgorithm:          hasher.AlgorithmName,
				NormalisationAlgorithm: string(cdv2.ManifestDigestV1),
				Value:                  hex.EncodeToString((hasher.HashFunction.Sum(nil))),
			}, nil
		default:
			return nil, fmt.Errorf("access type %s not supported", cr.Access.Type)
		}

	}
	err := cdv2Sign.AddDigestsToComponentDescriptor(context.TODO(), cd, cdResolver, resResolver)
	if err != nil {
		return fmt.Errorf("failed adding digests to cd %s:%s: %w", cd.Name, cd.Version, err)
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
	o.OciOptions.AddFlags(fs)
}
