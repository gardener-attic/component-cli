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
	"sigs.k8s.io/yaml"
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
	case "localOCIBlob":
		return downloaders.NewLocalOCIBlobDownloader(f.client), nil
	case "ociImage":
		return f.createOCIImageDownloader(spec)
	case "executable":
		return createExecutable(spec)
	default:
		return nil, fmt.Errorf("unknown downloader type %s", typ)
	}
}

func (f *DownloaderFactory) createOCIImageDownloader(rawSpec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	type downloaderSpec struct {
		BaseUrl        string `json:"baseUrl"`
		KeepSourceRepo bool   `json:"keepSourceRepo"`
	}

	var spec downloaderSpec
	err := yaml.Unmarshal(*rawSpec, &spec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return downloaders.NewOCIImageDownloader(f.client, f.cache), nil
}
