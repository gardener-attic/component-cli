// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package processors

import (
	"context"
	"fmt"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/opencontainers/go-digest"

	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/utils"
)

type blobDigester struct{}

func NewBlobDigester() process.ResourceStreamProcessor {
	obj := blobDigester{}
	return &obj
}

func (p *blobDigester) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cd, res, resBlobReader, err := utils.ReadProcessorMessage(r)
	if err != nil {
		return fmt.Errorf("unable to read processor message: %w", err)
	}
	if resBlobReader != nil {
		defer resBlobReader.Close()
	}

	dgst, err := digest.FromReader(resBlobReader)
	if err != nil {
		return fmt.Errorf("unable to calculate digest: %w", err)
	}
	digestspec := cdv2.DigestSpec{
		Algorithm: dgst.Algorithm().String(),
		Value:     dgst.Encoded(),
	}
	res.Digest = &digestspec

	if _, err := resBlobReader.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("unable to seek to beginning of resource blob file: %w", err)
	}

	if err := utils.WriteProcessorMessage(*cd, res, resBlobReader, w); err != nil {
		return fmt.Errorf("unable to write processor message: %w", err)
	}

	return nil
}
