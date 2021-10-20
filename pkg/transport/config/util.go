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

const (
	ExecutableType = "executable"
)

func createExecutable(rawSpec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	type executableSpec struct {
		Bin  string
		Args []string
		Env  []string
	}

	var spec executableSpec
	if err := yaml.Unmarshal(*rawSpec, &spec); err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return extensions.NewUDSExecutable(spec.Bin, spec.Args, spec.Env)
}
