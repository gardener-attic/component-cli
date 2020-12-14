// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// ResolveTransitiveComponentDescriptors resolves all component descriptors of a gioven root component descriptor
func ResolveTransitiveComponentDescriptors(ctx context.Context, resolver ComponentResolver, root *cdv2.ComponentDescriptor) (*cdv2.ComponentDescriptorList, error) {
	list := &cdv2.ComponentDescriptorList{}
	if err := resolveTransitiveComponentDescriptors(ctx, resolver, list, root); err != nil {
		return nil, err
	}
	return list, nil
}

func resolveTransitiveComponentDescriptors(ctx context.Context, resolver ComponentResolver, list *cdv2.ComponentDescriptorList, root *cdv2.ComponentDescriptor) error {
	repoCtx := root.GetEffectiveRepositoryContext()
	for _, ref := range root.ComponentReferences {
		cd, err := resolver.Resolve(ctx, repoCtx, ref.ComponentName, ref.Version)
		if err != nil {
			return fmt.Errorf("unable to resolve component %q:%q: %w", ref.ComponentName, ref.Version, err)
		}
		list.Components = append(list.Components, *cd)
		// resolve transitive dependencies
		if err := resolveTransitiveComponentDescriptors(ctx, resolver, list, root); err != nil {
			return fmt.Errorf("unable to resolve transitive dependencies of %q:%q: %w", ref.ComponentName, ref.Version, err)
		}
	}
	return nil
}
