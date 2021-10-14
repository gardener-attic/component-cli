// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"fmt"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/downloaders"
)

func NewDownloaderFactory(client ociclient.Client) *DownloaderFactory {
	return &DownloaderFactory{
		client: client,
	}
}

type DownloaderFactory struct {
	client ociclient.Client
}

func (f *DownloaderFactory) Create(typ string, spec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	switch typ {
	case "localOCIBlob":
		return downloaders.NewLocalOCIBlobDownloader(f.client), nil
	case "executable":
		return createExecutable(spec)
	default:
		return nil, fmt.Errorf("unable to create downloader: unknown type %s", typ)
	}
}
