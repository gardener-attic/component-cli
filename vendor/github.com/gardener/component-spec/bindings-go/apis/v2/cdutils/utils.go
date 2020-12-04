// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils

import v2 "github.com/gardener/component-spec/bindings-go/apis/v2"

// MergeResources merges two resources whereas the second one will overwrite defined attributes.
// Labels are merged by their name not by their index.
func MergeResources(a, b v2.Resource) v2.Resource {
	a.IdentityObjectMeta = MergeIdentityObjectMeta(a.IdentityObjectMeta, b.IdentityObjectMeta)
	if len(b.Relation) != 0 {
		a.Relation = b.Relation
	}
	if b.Access != nil {
		a.Access = b.Access
	}
	if len(b.SourceRef) != 0 {
		a.SourceRef = b.SourceRef
	}
	return a
}

// MergeSources merges two sources whereas the second one will overwrite defined attributes.
// Labels are merged by their name not by their index.
func MergeSources(a, b v2.Source) v2.Source {
	a.IdentityObjectMeta = MergeIdentityObjectMeta(a.IdentityObjectMeta, b.IdentityObjectMeta)
	if b.Access != nil {
		a.Access = b.Access
	}
	return a
}

// MergeIdentityObjectMeta merges two identity objects.
// Labels are merged by their name not by their index.
func MergeIdentityObjectMeta(a, b v2.IdentityObjectMeta) v2.IdentityObjectMeta {
	if StringDefined(b.Name) {
		a.Name = b.Name
	}
	if StringDefined(b.Version) {
		a.Version = b.Version
	}
	if StringDefined(b.Type) {
		a.Type = b.Type
	}

	for _, label := range b.Labels {
		if idx := GetLabelIdx(b.Labels, label.Name); idx != -1 {
			a.Labels[idx] = label
		} else {
			a.Labels = append(a.Labels, label)
		}
	}

	for key, val := range b.ExtraIdentity {
		a.ExtraIdentity[key] = val
	}

	return a
}

// GetLabel returns the label with a given name
func GetLabel(labels []v2.Label, name string) (v2.Label, bool) {
	for _, label := range labels {
		if label.Name == name {
			return label, true
		}
	}
	return v2.Label{}, false
}

// GetLabelIdx returns the index with a given name.
// if the label with the given name does not exist, -1 is returned.
func GetLabelIdx(labels []v2.Label, name string) int {
	for i, label := range labels {
		if label.Name == name {
			return i
		}
	}
	return -1
}

// StringDefined validates if a string is defined
func StringDefined(s string) bool {
	return len(s) != 0
}
