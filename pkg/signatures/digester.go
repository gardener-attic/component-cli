package signatures

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/gardener/component-cli/ociclient"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/apis/v2/signatures"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
)

type Digester struct {
	ociClient ociclient.Client
	hasher    signatures.Hasher
}

func NewDigester(ociClient ociclient.Client, hasher signatures.Hasher) *Digester {
	return &Digester{
		ociClient: ociClient,
		hasher:    hasher,
	}

}

func (d *Digester) DigestForResource(ctx context.Context, cd cdv2.ComponentDescriptor, res cdv2.Resource) (*cdv2.DigestSpec, error) {

	switch res.Access.Type {
	case cdv2.OCIRegistryType:
		return d.digestForOciArtifact(ctx, cd, res)
	case cdv2.LocalOCIBlobType:
		return d.digestForLocalOciBlob(ctx, cd, res)

	default:
		return nil, fmt.Errorf("access type %s not supported", res.Access.Type)
	}
}

func (d *Digester) digestForLocalOciBlob(ctx context.Context, componentDescriptor cdv2.ComponentDescriptor, res cdv2.Resource) (*cdv2.DigestSpec, error) {
	if res.Access.GetType() != cdv2.LocalOCIBlobType {
		return nil, fmt.Errorf("unsupported access type: %s", res.Access.Type)
	}

	repoctx := cdv2.OCIRegistryRepository{}
	if err := componentDescriptor.GetEffectiveRepositoryContext().DecodeInto(&repoctx); err != nil {
		return nil, fmt.Errorf("unable to decode repository context: %w", err)
	}

	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, fmt.Errorf("unable to create tempfile: %w", err)
	}
	defer tmpfile.Close()

	resolver := cdoci.NewResolver(d.ociClient)
	_, blobResolver, err := resolver.ResolveWithBlobResolver(ctx, &repoctx, componentDescriptor.Name, componentDescriptor.Version)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve component descriptor: %w", err)
	}
	if _, err := blobResolver.Resolve(ctx, res, tmpfile); err != nil {
		return nil, fmt.Errorf("unable to to resolve blob: %w", err)
	}

	tmpfile.Seek(0, io.SeekStart)
	d.hasher.HashFunction.Reset()

	if _, err := io.Copy(d.hasher.HashFunction, tmpfile); err != nil {
		return nil, fmt.Errorf("unable to hash blob: %w", err)
	}
	return &cdv2.DigestSpec{
		HashAlgorithm:          d.hasher.AlgorithmName,
		NormalisationAlgorithm: string(cdv2.LocalOciBlobDigestV1),
		Value:                  hex.EncodeToString((d.hasher.HashFunction.Sum(nil))),
	}, nil
}

func (d *Digester) digestForOciArtifact(ctx context.Context, componentDescriptor cdv2.ComponentDescriptor, res cdv2.Resource) (*cdv2.DigestSpec, error) {
	if res.Access.GetType() != cdv2.OCIRegistryType {
		return nil, fmt.Errorf("unsupported access type: %s", res.Access.Type)
	}

	ociAccess := &cdv2.OCIRegistryAccess{}
	if err := res.Access.DecodeInto(ociAccess); err != nil {
		return nil, fmt.Errorf("unable to decode resource access: %w", err)
	}

	manifest, err := d.ociClient.GetManifest(ctx, ociAccess.ImageReference)
	if err != nil {
		return nil, fmt.Errorf("failed resolving manifest: %w", err)
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed json marshaling: %w", err)
	}

	d.hasher.HashFunction.Reset()
	if _, err = d.hasher.HashFunction.Write(manifestBytes); err != nil {
		return nil, fmt.Errorf("failed hashing yaml, %w", err)
	}

	return &cdv2.DigestSpec{
		HashAlgorithm:          d.hasher.AlgorithmName,
		NormalisationAlgorithm: string(cdv2.ManifestDigestV1),
		Value:                  hex.EncodeToString((d.hasher.HashFunction.Sum(nil))),
	}, nil
}