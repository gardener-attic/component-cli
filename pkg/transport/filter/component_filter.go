package filter

import (
	"encoding/json"
	"fmt"
	"regexp"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type componentFilter struct {
	includeComponentNames []*regexp.Regexp
}

func (f componentFilter) Matches(cd *cdv2.ComponentDescriptor, r cdv2.Resource) bool {
	var matches bool
	for _, icn := range f.includeComponentNames {
		if matches = icn.MatchString(cd.Name); matches {
			break
		}
	}
	return matches
}

func CreateComponentFilterFromConfig(rawConfig *json.RawMessage) (Filter, error) {
	type RawConfigStruct struct {
		IncludeComponentNames []string `json:"includeComponentNames"`
	}
	var config RawConfigStruct
	err := yaml.Unmarshal(*rawConfig, &config)
	if err != nil {
		return nil, fmt.Errorf("unable to parse filter configuration for ComponentFilter %w", err)
	}
	icnRegexps := []*regexp.Regexp{}
	for _, icn := range config.IncludeComponentNames {
		icnRegexp, err := regexp.Compile(icn)
		if err != nil {
			return nil, fmt.Errorf("unable to parse regexp %s: %w", icn, err)
		}
		icnRegexps = append(icnRegexps, icnRegexp)
	}

	filter := componentFilter{
		includeComponentNames: icnRegexps,
	}
	return &filter, nil
}

func NewComponentFilter(includeComponentNames ...string) (Filter, error) {
	icnRegexps := []*regexp.Regexp{}
	for _, icn := range includeComponentNames {
		icnRegexp, err := regexp.Compile(icn)
		if err != nil {
			return nil, fmt.Errorf("unable to parse regexp %s: %w", icn, err)
		}
		icnRegexps = append(icnRegexps, icnRegexp)
	}

	filter := componentFilter{
		includeComponentNames: icnRegexps,
	}

	return &filter, nil
}
