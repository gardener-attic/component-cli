// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"fmt"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/downloaders"
)

func NewDownloaderFactory(client ociclient.Client, ocicache cache.Cache) *DownloaderFactory {
	return &DownloaderFactory{
		client: client,
		cache:  ocicache,
	}
}

type DownloaderFactory struct {
	client ociclient.Client
	cache  cache.Cache
}

func (f *DownloaderFactory) Create(typ string, spec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	switch typ {
	case "localOciBlobDL":
		return downloaders.NewLocalOCIBlobDownloader(f.client), nil
	case "ociImageDL":
		return downloaders.NewOCIImageDownloader(f.client, f.cache), nil
	case "executable":
		return createExecutable(spec)
	default:
		return nil, fmt.Errorf("unknown downloader type %s", typ)
	}
}
