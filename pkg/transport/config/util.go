// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/pkg/transport/process"
	"github.com/gardener/component-cli/pkg/transport/process/extensions"
)

const (
	ExecutableType = "Executable"
)

func createExecutable(rawSpec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	type executableSpec struct {
		Bin  string
		Args []string
		Env  map[string]string
	}

	var spec executableSpec
	if err := yaml.Unmarshal(*rawSpec, &spec); err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return extensions.NewUnixDomainSocketExecutable(spec.Bin, spec.Args, spec.Env)
}
