// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier
package uploaders

import (
	"context"
	"fmt"
	"io"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/serialize"
	"github.com/gardener/component-cli/pkg/utils"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

type ociImageUploader struct {
	client         ociclient.Client
	cache          cache.Cache
	targetRepo     string
	keepSourceRepo bool
}

func NewOCIImageUploader(client ociclient.Client, cache cache.Cache, targetRepo string, keepSourceRepo bool) process.ResourceStreamProcessor {
	obj := ociImageUploader{
		client:         client,
		cache:          cache,
		targetRepo:     targetRepo,
		keepSourceRepo: keepSourceRepo,
	}
	return &obj
}

func (u *ociImageUploader) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cd, res, resBlobReader, err := process.ReadProcessorMessage(r)
	if err != nil {
		return fmt.Errorf("unable to read input archive: %w", err)
	}
	defer resBlobReader.Close()

	ociArtifact, err := serialize.DeserializeOCIArtifact(resBlobReader, u.cache)
	if err != nil {
		return fmt.Errorf("unable to deserialize oci artifact: %w", err)
	}

	if res.Access.GetType() != cdv2.OCIRegistryType {
		return fmt.Errorf("unsupported access type: %+v", res.Access)
	}

	if res.Type != cdv2.OCIImageType {
		return fmt.Errorf("unsupported resource type: %s", res.Type)
	}

	ociAccess := &cdv2.OCIRegistryAccess{}
	if err := res.Access.DecodeInto(ociAccess); err != nil {
		return fmt.Errorf("unable to decode resource access: %w", err)
	}

	target, err := utils.TargetOCIArtifactRef(u.targetRepo, ociAccess.ImageReference, u.keepSourceRepo)
	if err != nil {
		return fmt.Errorf("unable to create target oci artifact reference: %w", err)
	}

	if err := u.client.PushOCIArtifact(ctx, target, ociArtifact, ociclient.WithStore(u.cache)); err != nil {
		return fmt.Errorf("unable to push oci artifact: %w", err)
	}

	blobReader, err := serialize.SerializeOCIArtifact(*ociArtifact, u.cache)
	if err != nil {
		return fmt.Errorf("unable to serialize oci artifact: %w", err)
	}
	defer blobReader.Close()

	if err := process.WriteProcessorMessage(*cd, res, blobReader, w); err != nil {
		return fmt.Errorf("unable to write processor message: %w", err)
	}

	return nil
}
