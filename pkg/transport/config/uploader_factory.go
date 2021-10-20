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
	"github.com/gardener/component-cli/pkg/transport/process/uploaders"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"sigs.k8s.io/yaml"
)

const (
	LocalOCIBlobUploaderType = "localOciBlobUL"
	OCIImageUploaderType     = "ociImageUL"
)

func NewUploaderFactory(client ociclient.Client, ocicache cache.Cache, targetCtx cdv2.OCIRegistryRepository) *UploaderFactory {
	return &UploaderFactory{
		client:    client,
		cache:     ocicache,
		targetCtx: targetCtx,
	}
}

type UploaderFactory struct {
	client    ociclient.Client
	cache     cache.Cache
	targetCtx cdv2.OCIRegistryRepository
}

func (f *UploaderFactory) Create(typ string, spec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	switch typ {
	case LocalOCIBlobUploaderType:
		return uploaders.NewLocalOCIBlobUploader(f.client, f.targetCtx), nil
	case OCIImageUploaderType:
		return f.createOCIImageUploader(spec)
	case ExecutableType:
		return createExecutable(spec)
	default:
		return nil, fmt.Errorf("unknown uploader type %s", typ)
	}
}

func (f *UploaderFactory) createOCIImageUploader(rawSpec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	type uploaderSpec struct {
		BaseUrl        string `json:"baseUrl"`
		KeepSourceRepo bool   `json:"keepSourceRepo"`
	}

	var spec uploaderSpec
	err := yaml.Unmarshal(*rawSpec, &spec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return uploaders.NewOCIImageUploader(f.client, f.cache, spec.BaseUrl, spec.KeepSourceRepo), nil
}
