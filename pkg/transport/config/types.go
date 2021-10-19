// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import "encoding/json"

type meta struct {
	Version string `json:"version"`
}

type transportConfig struct {
	Meta        meta                   `json:"meta"`
	Uploaders   []UploaderDefinition   `json:"uploaders"`
	Processors  []ProcessorDefinition  `json:"processors"`
	Downloaders []DownloaderDefinition `json:"downloaders"`
	Rules       []Rule                 `json:"rules"`
}

type ExtensionType string

const (
	ExecutableProcessor ExtensionType = "executeable"
)

type BaseProcessorDefinition struct {
	Name string           `json:"name"`
	Type ExtensionType    `json:"type"`
	Spec *json.RawMessage `json:"spec"`
}

type HookDefinition struct {
	BaseProcessorDefinition
}

type FilterDefinition struct {
	Type string           `json:"type"`
	Spec *json.RawMessage `json:"spec"`
}

type DownloaderDefinition struct {
	BaseProcessorDefinition
	Filters []FilterDefinition `json:"filters"`
}

type UploaderDefinition struct {
	BaseProcessorDefinition
	Filters []FilterDefinition `json:"filters"`
}

type ProcessorDefinition struct {
	BaseProcessorDefinition
}

type ProcessorReference struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Rule struct {
	Name            string
	CopyByReference bool                 `json:"copyByReference"`
	Filters         []FilterDefinition   `json:"filters"`
	Processors      []ProcessorReference `json:"processors"`
}
