// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"fmt"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/pkg/transport/process"
)

func NewUploaderFactory(client ociclient.Client) *UploaderFactory {
	return &UploaderFactory{
		client: client,
	}
}

type UploaderFactory struct {
	client ociclient.Client
}

func (f *UploaderFactory) Create(typ string, spec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	switch typ {
	case "executable":
		return createExecutable(spec)
	default:
		return nil, fmt.Errorf("unable to create uploader: unknown type %s", typ)
	}
}
