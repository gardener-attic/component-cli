// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package uploaders

import (
	"encoding/json"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/extensions"
)

const (
	// LocalOCIBlobUploaderType defines the type of a local oci blob uploader
	LocalOCIBlobUploaderType = "LocalOciBlobUploader"

	// OCIArtifactUploaderType defines the type of an oci artifact uploader
	OCIArtifactUploaderType = "OciArtifactUploader"
)

// NewUploaderFactory creates a new uploader factory
// How to add a new uploader (without using extension mechanism):
// - Add Go file to uploaders package which contains the source code of the new uploader
// - Add string constant for new uploader type -> will be used in UploaderFactory.Create()
// - Add source code for creating new uploader to UploaderFactory.Create() method
func NewUploaderFactory(client ociclient.Client, ocicache cache.Cache, targetCtx cdv2.OCIRegistryRepository, log logr.Logger) *UploaderFactory {
	return &UploaderFactory{
		client:    client,
		cache:     ocicache,
		targetCtx: targetCtx,
		log:       log,
	}
}

// UploaderFactory defines a helper struct for creating uploaders
type UploaderFactory struct {
	client    ociclient.Client
	cache     cache.Cache
	targetCtx cdv2.OCIRegistryRepository
	log       logr.Logger
}

// Create creates a new uploader defined by a type and a spec
func (f *UploaderFactory) Create(uploaderType string, spec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	switch uploaderType {
	case LocalOCIBlobUploaderType:
		return NewLocalOCIBlobUploader(f.client, f.targetCtx)
	case OCIArtifactUploaderType:
		return f.createOCIArtifactUploader(spec)
	case extensions.ExecutableType:
		return extensions.CreateExecutable(spec, f.log)
	default:
		return nil, fmt.Errorf("unknown uploader type %s", uploaderType)
	}
}

func (f *UploaderFactory) createOCIArtifactUploader(rawSpec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	type uploaderSpec struct {
		BaseUrl        string `json:"baseUrl"`
		KeepSourceRepo bool   `json:"keepSourceRepo"`
	}

	var spec uploaderSpec
	if err := yaml.Unmarshal(*rawSpec, &spec); err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return NewOCIArtifactUploader(f.client, f.cache, spec.BaseUrl, spec.KeepSourceRepo)
}
