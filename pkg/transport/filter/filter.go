package filter

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

type Filter interface {
	Matches(*cdv2.ComponentDescriptor, cdv2.Resource) bool
}
