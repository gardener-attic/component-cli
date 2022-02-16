// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Copy copies a oci artifact from one location to a target ref.
// The artifact is copied without any modification.
// This function does directly stream the blobs from the upstream it does not use any cache.
func Copy(ctx context.Context, client Client, srcRef, tgtRef string) error {
	desc, rawManifest, err := client.GetManifestRaw(ctx, srcRef)
	if err != nil {
		return fmt.Errorf("unable to get manifest: %w", err)
	}

	store := GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		return client.Fetch(ctx, srcRef, desc, writer)
	})

	if IsMultiArchImage(desc.MediaType) {
		index := ocispecv1.Index{}
		if err := json.Unmarshal(rawManifest, &index); err != nil {
			return fmt.Errorf("unable to unmarshal image index: %w", err)
		}

		for _, manifestDesc := range index.Manifests {
			buf := bytes.NewBuffer([]byte{})
			if err := client.Fetch(ctx, srcRef, manifestDesc, buf); err != nil {
				return fmt.Errorf("unable to get manifest: %w", err)
			}

			if err := client.PushManifestRaw(ctx, tgtRef, manifestDesc, buf.Bytes(), WithStore(store)); err != nil {
				return fmt.Errorf("unable to push manifest: %w", err)
			}
		}
	}

	if err := client.PushManifestRaw(ctx, tgtRef, desc, rawManifest, WithStore(store)); err != nil {
		return fmt.Errorf("unable to push manifest: %w", err)
	}

	return nil
}

// GenericStore is a helper struct to implement a custom oci blob store.
type GenericStore func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error

func (s GenericStore) Get(desc ocispecv1.Descriptor) (io.ReadCloser, error) {
	ctx := context.Background()
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		defer ctx.Done()
		if err := s(ctx, desc, writer); err != nil {
			_ = reader.CloseWithError(err)
		}
	}()
	return reader, nil
}
