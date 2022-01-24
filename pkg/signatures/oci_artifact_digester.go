package signatures

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gardener/component-cli/ociclient"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	cdv2Sign "github.com/gardener/component-spec/bindings-go/apis/v2/signatures"
)

type OciArtifactDigester struct {
	client ociclient.Client
}

func NewOciArtifactDigester(client ociclient.Client) (cdv2Sign.ResourceDigester, error) {
	if client == nil {
		return nil, errors.New("client must not be nil")
	}

	obj := OciArtifactDigester{
		client: client,
	}
	return &obj, nil
}

func (digester OciArtifactDigester) DigestForResource(ctx context.Context, componentDescriptor cdv2.ComponentDescriptor, res cdv2.Resource, hasher cdv2Sign.Hasher) (*cdv2.DigestSpec, error) {
	if res.Access.GetType() != cdv2.OCIRegistryType {
		return nil, fmt.Errorf("unsupported access type: %s", res.Access.Type)
	}

	ociAccess := &cdv2.OCIRegistryAccess{}
	if err := res.Access.DecodeInto(ociAccess); err != nil {
		return nil, fmt.Errorf("unable to decode resource access: %w", err)
	}

	manifest, err := digester.client.GetManifest(ctx, ociAccess.ImageReference)
	if err != nil {
		return nil, fmt.Errorf("failed resolving manifest: %w", err)
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed json marshaling: %w", err)
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
}
