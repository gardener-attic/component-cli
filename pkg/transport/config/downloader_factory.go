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

const (
	// LocalOCIBlobDownloaderType defines the type of a local oci blob downloader
	LocalOCIBlobDownloaderType = "LocalOciBlobDownloader"

	// OCIArtifactDownloaderType defines the type of an oci artifact downloader
	OCIArtifactDownloaderType = "OciArtifactDownloader"
)

// NewDownloaderFactory creates a new downloader factory
func NewDownloaderFactory(client ociclient.Client, ocicache cache.Cache) *DownloaderFactory {
	return &DownloaderFactory{
		client: client,
		cache:  ocicache,
	}
}

// DownloaderFactory defines a helper struct for creating downloaders
type DownloaderFactory struct {
	client ociclient.Client
	cache  cache.Cache
}

// Create creates a new downloader defined by a type and a spec
func (f *DownloaderFactory) Create(downloaderType string, spec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	switch downloaderType {
	case LocalOCIBlobDownloaderType:
		return downloaders.NewLocalOCIBlobDownloader(f.client)
	case OCIArtifactDownloaderType:
		return downloaders.NewOCIArtifactDownloader(f.client, f.cache)
	case ExecutableType:
		return createExecutable(spec)
	default:
		return nil, fmt.Errorf("unknown downloader type %s", downloaderType)
	}
}
