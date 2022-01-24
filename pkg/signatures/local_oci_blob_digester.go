package signatures

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/gardener/component-cli/ociclient"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"

	cdv2Sign "github.com/gardener/component-spec/bindings-go/apis/v2/signatures"
)

type LocalOciBlobDigester struct {
	client ociclient.Client
}

func NewLocalOciBlobDigester(client ociclient.Client) (cdv2Sign.ResourceDigester, error) {
	if client == nil {
		return nil, errors.New("client must not be nil")
	}

	obj := LocalOciBlobDigester{
		client: client,
	}
	return &obj, nil
}

func (digester LocalOciBlobDigester) DigestForResource(ctx context.Context, componentDescriptor cdv2.ComponentDescriptor, res cdv2.Resource, hasher cdv2Sign.Hasher) (*cdv2.DigestSpec, error) {
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

	resolver := cdoci.NewResolver(digester.client)
	_, blobResolver, err := resolver.ResolveWithBlobResolver(ctx, &repoctx, componentDescriptor.Name, componentDescriptor.Version)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve component descriptor: %w", err)
	}
	if _, err := blobResolver.Resolve(ctx, res, tmpfile); err != nil {
		return nil, fmt.Errorf("unable to to resolve blob: %w", err)
	}

	tmpfile.Seek(0, io.SeekStart)
	hasher.HashFunction.Reset()

	if _, err := io.Copy(hasher.HashFunction, tmpfile); err != nil {
		return nil, fmt.Errorf("unable to hash blob: %w", err)
	}
	return &cdv2.DigestSpec{
		HashAlgorithm:          hasher.AlgorithmName,
		NormalisationAlgorithm: string(cdv2.LocalOciBlobDigestV1),
		Value:                  hex.EncodeToString((hasher.HashFunction.Sum(nil))),
	}, nil
}
