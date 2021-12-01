// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package utils

import (
	"fmt"
	"os"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/pkg/transport/filters"
)

type RepositoryContextOverride struct {
	Overrides []Override
}

type Override struct {
	Filter            filters.Filter
	RepositoryContext *cdv2.OCIRegistryRepository
}

func ParseRepositoryContextConfig(configPath string) (*RepositoryContextOverride, error) {
	type meta struct {
		Version string `json:"version"`
	}

	type override struct {
		ComponentNameFilterSpec *filters.ComponentNameFilterSpec `json:"componentNameFilterSpec"`
		RepositoryContext       *cdv2.OCIRegistryRepository      `json:"repositoryContext"`
	}

	type repositoryContextOverride struct {
		Meta      meta       `json:"meta"`
		Overrides []override `json:"overrides"`
	}

	repoCtxOverrideCfgYaml, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read config file: %w", err)
	}

	var cfg repositoryContextOverride
	if err := yaml.Unmarshal(repoCtxOverrideCfgYaml, &cfg); err != nil {
		return nil, fmt.Errorf("unable to parse config file: %w", err)
	}

	parsedCfg := RepositoryContextOverride{
		Overrides: []Override{},
	}

	for _, o := range cfg.Overrides {
		f, err := filters.NewComponentNameFilter(*o.ComponentNameFilterSpec)
		if err != nil {
			return nil, fmt.Errorf("unable to create component name filter: %w", err)
		}
		po := Override{
			Filter:            f,
			RepositoryContext: o.RepositoryContext,
		}
		parsedCfg.Overrides = append(parsedCfg.Overrides, po)
	}

	return &parsedCfg, nil
}

func (c *RepositoryContextOverride) GetRepositoryContext(componentName string, defaultRepoCtx cdv2.OCIRegistryRepository) *cdv2.OCIRegistryRepository {
	ctx := defaultRepoCtx
	for _, o := range c.Overrides {
		dummyCd := cdv2.ComponentDescriptor{
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name: componentName,
				},
			},
		}
		if o.Filter.Matches(dummyCd, cdv2.Resource{}) {
			ctx = *o.RepositoryContext
		}
	}
	return &ctx
}
