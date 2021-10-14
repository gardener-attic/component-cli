// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"fmt"

	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/extensions"
	"sigs.k8s.io/yaml"
)

type executableSpec struct {
	Bin  string
	Args []string
	Env  []string
}

func createExecutable(spec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	var specstr executableSpec
	if err := yaml.Unmarshal(*spec, &specstr); err != nil {
		return nil, fmt.Errorf("unable to parse downloader spec: %w", err)
	}
	return extensions.NewUDSExecutable(specstr.Bin, specstr.Args, specstr.Env)
}
