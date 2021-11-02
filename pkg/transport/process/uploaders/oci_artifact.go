// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier
package uploaders

import (
	"context"
	"errors"
	"fmt"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/utils"
)

type ociArtifactUploader struct {
	client         ociclient.Client
	cache          cache.Cache
	baseUrl        string
	keepSourceRepo bool
}

func NewOCIImageUploader(client ociclient.Client, cache cache.Cache, baseUrl string, keepSourceRepo bool) (process.ResourceStreamProcessor, error) {
	if client == nil {
		return nil, errors.New("client must not be nil")
	}

	if cache == nil {
		return nil, errors.New("cache must not be nil")
	}

	if baseUrl == "" {
		return nil, errors.New("baseUrl must not be empty")
	}

	obj := ociArtifactUploader{
		client:         client,
		cache:          cache,
		baseUrl:        baseUrl,
		keepSourceRepo: keepSourceRepo,
	}
	return &obj, nil
}

func (u *ociArtifactUploader) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cd, res, resBlobReader, err := process.ReadProcessorMessage(r)
	if err != nil {
		return fmt.Errorf("unable to read processor message: %w", err)
	}
	defer resBlobReader.Close()

	ociArtifact, err := process.DeserializeOCIArtifact(resBlobReader, u.cache)
	if err != nil {
		return fmt.Errorf("unable to deserialize oci artifact: %w", err)
	}

	if res.Access.GetType() != cdv2.OCIRegistryType {
		return fmt.Errorf("unsupported access type: %s", res.Access.Type)
	}

	if res.Type != cdv2.OCIImageType {
		return fmt.Errorf("unsupported resource type: %s", res.Type)
	}

	ociAccess := &cdv2.OCIRegistryAccess{}
	if err := res.Access.DecodeInto(ociAccess); err != nil {
		return fmt.Errorf("unable to decode resource access: %w", err)
	}

	target, err := utils.TargetOCIArtifactRef(u.baseUrl, ociAccess.ImageReference, u.keepSourceRepo)
	if err != nil {
		return fmt.Errorf("unable to create target oci artifact reference: %w", err)
	}

	if err := u.client.PushOCIArtifact(ctx, target, ociArtifact, ociclient.WithStore(u.cache)); err != nil {
		return fmt.Errorf("unable to push oci artifact: %w", err)
	}

	blobReader, err := process.SerializeOCIArtifact(*ociArtifact, u.cache)
	if err != nil {
		return fmt.Errorf("unable to serialize oci artifact: %w", err)
	}
	defer blobReader.Close()

	if err := process.WriteProcessorMessage(*cd, res, blobReader, w); err != nil {
		return fmt.Errorf("unable to write processor message: %w", err)
	}

	return nil
}
