// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imagevector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/distribution/reference"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/apis/v2/cdutils"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// LabelPrefix is the prefix for all image vector related labels on component descriptor resources.
const LabelPrefix = "imagevector.gardener.cloud"

// Label creates a new label for a name and append the image vector prefix.
func Label(name string) string {
	return fmt.Sprintf("%s/%s", LabelPrefix, name)
}

// ExtraIdentityKeyPrefix is the prefix for all image vector related extra identities on component descriptor resources.
const ExtraIdentityKeyPrefix = "imagevector-gardener-cloud"

// ExtraIdentityKey creates a new identity key for a name and append the image vector prefix.
func ExtraIdentityKey(name string) string {
	return fmt.Sprintf("%s+%s", ExtraIdentityKeyPrefix, name)
}

var (
	NameLabel             = Label("name")
	RepositoryLabel       = Label("repository")
	SourceRepositoryLabel = Label("source-repository")
	TargetVersionLabel    = Label("target-version")
	RuntimeVersionLabel   = Label("runtime-version")
	ImagesLabel           = Label("images")

	TagExtraIdentity = ExtraIdentityKey("tag")
)

// ImageVector defines a image vector that defines oci images with specific requirements
type ImageVector struct {
	Images []ImageEntry `json:"images"`
}

// ImageEntry defines one image entry of a image vector
type ImageEntry struct {
	// Name defines the name of the image entry
	Name string `json:"name"`
	// SourceRepository is the name of the repository where the image was build from
	SourceRepository string `json:"sourceRepository,omitempty"`
	// Repository defines the image repository
	Repository string `json:"repository"`
	// +optional
	Tag *string `json:"tag,omitempty"`
	// +optional
	RuntimeVersion *string `json:"runtimeVersion,omitempty"`
	// +optional
	TargetVersion *string `json:"targetVersion,omitempty"`
	// +optional
	Labels []cdv2.Label `json:"labels,omitempty"`
}

// ParseImageOptions are options to configure the image vector parsing.
type ParseImageOptions struct {
	// ComponentReferencePrefixes are prefixes that are used to identify images from other components.
	// These images are then not added as direct resources but the source repository is used as the component reference.
	ComponentReferencePrefixes []string

	// GenericDependencies define images that should be untouched and not added as real dependency to the component descriptors.
	// These dependencies are added a specific label to the component descriptor.
	GenericDependencies []string
}

// CAResources are the resources that are returned
type CAResources struct {
	Resources           []cdv2.Resource
	ComponentReferences []cdv2.ComponentReference
	GenericDependencies *cdv2.Label
}

// ParseImageVector parses a image vector and generates the corresponding component descriptor resources.
func ParseImageVector(reader io.Reader, opts *ParseImageOptions) (*CAResources, error) {
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 1024)

	imageVector := &ImageVector{}
	if err := decoder.Decode(imageVector); err != nil {
		return nil, fmt.Errorf("unable to decode image vector: %w", err)
	}

	out := &CAResources{
		Resources:           make([]cdv2.Resource, 0),
		ComponentReferences: make([]cdv2.ComponentReference, 0),
	}
	genericImageVector := &ImageVector{}
	for _, image := range imageVector.Images {
		if entryMatchesPrefix(opts.GenericDependencies, image.Name) {
			genericImageVector.Images = append(genericImageVector.Images, image)
			continue
		}
		if image.Tag == nil {
			continue
		}

		if entryMatchesPrefix(opts.ComponentReferencePrefixes, image.Repository) {
			// add image as component reference
			ref := cdv2.ComponentReference{
				Name:          image.Name,
				ComponentName: image.SourceRepository,
				Version:       *image.Tag,
				ExtraIdentity: map[string]string{
					TagExtraIdentity: *image.Tag,
				},
				Labels: make([]cdv2.Label, 0),
			}
			// add complete image as label
			var err error
			out.ComponentReferences, err = addComponentReference(out.ComponentReferences, ref, image)
			if err != nil {
				return nil, fmt.Errorf("unable to add component reference for %q: %w", image.Name, err)
			}
			continue
		}

		res := cdv2.Resource{
			IdentityObjectMeta: cdv2.IdentityObjectMeta{
				Labels: make([]cdv2.Label, 0),
			},
		}
		res.Name = image.Name
		res.Version = *image.Tag
		res.Type = cdv2.OCIImageType
		res.Relation = cdv2.ExternalRelation

		var err error
		res.Labels, err = cdutils.SetLabel(res.Labels, NameLabel, image.Name)
		if err != nil {
			return nil, fmt.Errorf("unable to add name label to resource for image %q: %w", image.Name, err)
		}

		for _, label := range image.Labels {
			res.Labels = cdutils.SetRawLabel(res.Labels, label.Name, label.Value)
		}

		if len(image.Repository) != 0 {
			res.Labels, err = cdutils.SetLabel(res.Labels, RepositoryLabel, image.Repository)
			if err != nil {
				return nil, fmt.Errorf("unable to add repository label to resource for image %q: %w", image.Name, err)
			}
		}
		if len(image.SourceRepository) != 0 {
			res.Labels, err = cdutils.SetLabel(res.Labels, SourceRepositoryLabel, image.SourceRepository)
			if err != nil {
				return nil, fmt.Errorf("unable to add source repository label to resource for image %q: %w", image.Name, err)
			}
		}
		if image.TargetVersion != nil {
			res.Labels, err = cdutils.SetLabel(res.Labels, TargetVersionLabel, image.TargetVersion)
			if err != nil {
				return nil, fmt.Errorf("unable to add target version label to resource for image %q: %w", image.Name, err)
			}
		}
		if image.RuntimeVersion != nil {
			res.Labels, err = cdutils.SetLabel(res.Labels, RuntimeVersionLabel, image.RuntimeVersion)
			if err != nil {
				return nil, fmt.Errorf("unable to add target version label to resource for image %q: %w", image.Name, err)
			}
		}

		// set the tag as identity
		cdutils.SetExtraIdentityField(&res.IdentityObjectMeta, TagExtraIdentity, *image.Tag)

		// todo: also consider digests
		ociImageAccess := cdv2.NewOCIRegistryAccess(image.Repository + ":" + *image.Tag)
		uObj, err := cdutils.ToUnstructuredTypedObject(cdv2.DefaultJSONTypedObjectCodec, ociImageAccess)
		if err != nil {
			return nil, fmt.Errorf("unable to create oci registry access for %q: %w", image.Name, err)
		}
		res.Access = uObj
		out.Resources = append(out.Resources, res)
	}

	// parse label
	if len(genericImageVector.Images) != 0 {
		genericImageVectorBytes, err := json.Marshal(genericImageVector)
		if err != nil {
			return nil, fmt.Errorf("unable to parse generic image vector: %w", err)
		}
		out.GenericDependencies = &cdv2.Label{
			Name:  ImagesLabel,
			Value: genericImageVectorBytes,
		}
	}

	return out, nil
}

// addComponentReference adds the given component to the list of component references.
// if the component is already declared, the given image entry is appended to the images label
func addComponentReference(refs []cdv2.ComponentReference, new cdv2.ComponentReference, entry ImageEntry) ([]cdv2.ComponentReference, error) {
	for i, ref := range refs {
		if ref.Name == new.Name && ref.Version == new.Version {

			// parse current images and add the image
			imageVector := &ImageVector{
				Images: []ImageEntry{entry},
			}
			data, ok := ref.GetLabels().Get(ImagesLabel)
			if ok {
				if err := json.Unmarshal(data, imageVector); err != nil {
					return nil, err
				}
				imageVector.Images = append(imageVector.Images, entry)
			}
			var err error
			ref.Labels, err = cdutils.SetLabel(ref.Labels, ImagesLabel, imageVector)
			if err != nil {
				return nil, err
			}
			refs[i] = ref
			return refs, nil
		}
	}

	imageVector := ImageVector{
		Images: []ImageEntry{entry},
	}
	var err error
	new.Labels, err = cdutils.SetLabel(new.Labels, ImagesLabel, imageVector)
	if err != nil {
		return nil, err
	}
	return append(refs, new), nil
}

// ParseComponentDescriptor parses a component descriptor and returns the defined image vector
func ParseComponentDescriptor(cd *cdv2.ComponentDescriptor, list *cdv2.ComponentDescriptorList) (*ImageVector, error) {
	imageVector := &ImageVector{}

	// parse all images from the component descriptors resources
	images, err := parseImagesFromResources(cd.Resources)
	if err != nil {
		return nil, err
	}
	imageVector.Images = append(imageVector.Images, images...)

	images, err = parseImagesFromComponentReferences(cd, list)
	if err != nil {
		return nil, err
	}
	imageVector.Images = append(imageVector.Images, images...)

	return imageVector, nil
}

// parseImagesFromResources parse all images from the component descriptors resources
func parseImagesFromResources(resources []cdv2.Resource) ([]ImageEntry, error) {
	images := make([]ImageEntry, 0)
	for _, res := range resources {
		if res.GetType() != cdv2.OCIImageType || res.Access.GetType() != cdv2.OCIRegistryType {
			continue
		}
		var name string
		if ok, err := getLabel(res.GetLabels(), NameLabel, &name); !ok || err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to get name for %q: %w", res.GetName(), err)
			}
			continue
		}

		entry := ImageEntry{
			Name: string(name),
		}

		if err := parseResourceAccess(&entry, res); err != nil {
			return nil, err
		}

		// set additional information
		var targetVersion string
		if ok, err := getLabel(res.GetLabels(), TargetVersionLabel, &targetVersion); ok || err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to get target version for %q: %w", res.GetName(), err)
			}
			entry.TargetVersion = &targetVersion
		}
		var runtimeVersion string
		if ok, err := getLabel(res.GetLabels(), RuntimeVersionLabel, &runtimeVersion); ok || err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to get target version for %q: %w", res.GetName(), err)
			}
			entry.RuntimeVersion = &runtimeVersion
		}

		images = append(images, entry)
	}
	return images, nil
}

// parseImagesFromComponentReferences parse all images from the component descriptors references
func parseImagesFromComponentReferences(ca *cdv2.ComponentDescriptor, list *cdv2.ComponentDescriptorList) ([]ImageEntry, error) {
	ctx := context.Background()
	defer ctx.Done()
	images := make([]ImageEntry, 0)

	for _, ref := range ca.ComponentReferences {

		// only resolve the component reference if a images.yaml is defined
		imageVector := &ImageVector{}
		if ok, err := getLabel(ref.GetLabels(), ImagesLabel, imageVector); !ok || err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to parse images label from component reference %q: %w", ref.GetName(), err)
			}
			continue
		}

		refCD, err := list.GetComponent(ref.ComponentName, ref.Version)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve component descriptor %q: %w", ref.GetName(), err)
		}

		// find the matching resource by name and version
		for _, image := range imageVector.Images {
			foundResources, err := refCD.GetResourcesByName(image.Name)
			if err != nil {
				return nil, fmt.Errorf("unable to find images for %q in component refernce %q: %w", image.Name, ref.GetName(), err)
			}
			for _, res := range foundResources {
				if res.GetVersion() != *image.Tag {
					continue
				}
				if err := parseResourceAccess(&image, res); err != nil {
					return nil, fmt.Errorf("unable to find images for %q in component refernce %q: %w", image.Name, ref.GetName(), err)
				}
				images = append(images, image)
			}
		}

	}

	return images, nil
}

// parseResourceAccess parses a resource's access and sets the repository and tag on the given image entry.
// Currently only access of type 'ociRegistry' is supported.
func parseResourceAccess(imageEntry *ImageEntry, res cdv2.Resource) error {
	access := &cdv2.OCIRegistryAccess{}
	if err := cdv2.NewCodec(nil, nil, nil).Decode(res.Access.Raw, access); err != nil {
		return fmt.Errorf("unable to decode ociRegistry access: %w", err)
	}

	ref, err := reference.Parse(access.ImageReference)
	if err != nil {
		return fmt.Errorf("unable to parse image reference %q: %w", access.ImageReference, err)
	}

	named, ok := ref.(reference.Named)
	if !ok {
		return fmt.Errorf("unable to get repository for %q", ref.String())
	}
	imageEntry.Repository = named.Name()

	switch r := ref.(type) {
	case reference.Tagged:
		tag := r.Tag()
		imageEntry.Tag = &tag
	case reference.Digested:
		tag := r.Digest().String()
		imageEntry.Tag = &tag
	}
	return nil
}

func getLabel(labels cdv2.Labels, name string, into interface{}) (bool, error) {
	val, ok := labels.Get(name)
	if !ok {
		return false, nil
	}

	if err := json.Unmarshal(val, into); err != nil {
		return false, err
	}
	return true, nil
}

func entryMatchesPrefix(prefixes []string, val string) bool {
	for _, pref := range prefixes {
		if strings.HasPrefix(val, pref) {
			return true
		}
	}
	return false
}
