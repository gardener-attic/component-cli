package oci

import (
	"errors"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type Artifact struct {
	manifest *Manifest
	index    *Index
}

type Manifest struct {
	Desc ocispecv1.Descriptor
	Data *ocispecv1.Manifest
}

type Index struct {
	Manifests   []*Manifest
	Annotations map[string]string
}

func NewManifestArtifact(m *Manifest) *Artifact {
	a := Artifact{
		manifest: m,
	}
	return &a
}

func NewIndexArtifact(i *Index) *Artifact {
	a := Artifact{
		index: i,
	}
	return &a
}

func (a *Artifact) GetManifest() *Manifest {
	return a.manifest
}

func (a *Artifact) GetIndex() *Index {
	return a.index
}

func (a *Artifact) SetManifest(m *Manifest) error {
	if m == nil {
		return errors.New("manifest must not be nil")
	}

	if a.IsIndex() {
		return errors.New("unable to set manifest on index artifact")
	}

	a.manifest = m
	return nil
}

func (a *Artifact) SetIndex(i *Index) error {
	if i == nil {
		return errors.New("index must not be nil")
	}

	if a.IsManifest() {
		return errors.New("unable to set index on manifest artifact")
	}

	a.index = i
	return nil
}

func (a *Artifact) IsManifest() bool {
	return a.manifest != nil
}

func (a *Artifact) IsIndex() bool {
	return a.index != nil
}
