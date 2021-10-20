// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package filter

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

type Filter interface {
	// Matches receives a component descriptor and a resource and returns whether they match the filter criteria
	Matches(*cdv2.ComponentDescriptor, cdv2.Resource) bool
}
