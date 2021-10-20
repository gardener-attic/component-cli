// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package filter

import (
	"fmt"
	"regexp"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

type componentNameFilter struct {
	includeComponentNames []*regexp.Regexp
}

func (f componentNameFilter) Matches(cd *cdv2.ComponentDescriptor, r cdv2.Resource) bool {
	var matches bool
	for _, icn := range f.includeComponentNames {
		if matches = icn.MatchString(cd.Name); matches {
			break
		}
	}
	return matches
}

func NewComponentNameFilter(includeComponentNames ...string) (Filter, error) {
	icnRegexps := []*regexp.Regexp{}
	for _, icn := range includeComponentNames {
		icnRegexp, err := regexp.Compile(icn)
		if err != nil {
			return nil, fmt.Errorf("unable to parse regexp %s: %w", icn, err)
		}
		icnRegexps = append(icnRegexps, icnRegexp)
	}

	filter := componentNameFilter{
		includeComponentNames: icnRegexps,
	}

	return &filter, nil
}
