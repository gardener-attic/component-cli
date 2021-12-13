// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package processors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/opencontainers/go-digest"

	"github.com/gardener/component-cli/pkg/transport/process"
	processutils "github.com/gardener/component-cli/pkg/transport/process/utils"
)

type ociManifestDigester struct{}

func NewOCIManifestDigester() process.ResourceStreamProcessor {
	obj := ociManifestDigester{}
	return &obj
}

func (f *ociManifestDigester) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cd, res, blobreader, err := processutils.ReadProcessorMessage(r)
	if err != nil {
		return fmt.Errorf("unable to read archive: %w", err)
	}
	if blobreader == nil {
		return errors.New("resource blob must not be nil")
	}
	defer blobreader.Close()

	manifest, index, err := processutils.GetManifestOrIndexFromSerializedOCIArtifact(blobreader)
	if err != nil {
		return fmt.Errorf("unable to deserialize oci artifact: %w", err)
	}

	var content []byte
	if manifest != nil {
		content, err = json.Marshal(manifest)
		if err != nil {
			return fmt.Errorf("unable to marshal manifest: %w", err)
		}
	} else if index != nil {
		content, err = json.Marshal(index)
		if err != nil {
			return fmt.Errorf("unable to marshal image index: %w", err)
		}
	} else {
		return errors.New("")
	}

	dgst := digest.FromBytes(content)
	digestspec := cdv2.DigestSpec{
		Algorithm: dgst.Algorithm().String(),
		Value:     dgst.Encoded(),
	}
	res.Digest = &digestspec

	if _, err := blobreader.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("unable to seek to beginning of resource blob file: %w", err)
	}

	if err = processutils.WriteProcessorMessage(*cd, res, blobreader, w); err != nil {
		return fmt.Errorf("unable to write archive: %w", err)
	}

	return nil
}
