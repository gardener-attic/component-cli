// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package set_test

import (
	"context"
	"path/filepath"
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"github.com/gardener/component-cli/pkg/commands/componentarchive/set"
	"github.com/gardener/component-cli/pkg/componentarchive"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resources Test Suite")
}

var _ = Describe("Set", func() {

	var testdataFs vfs.FileSystem

	BeforeEach(func() {
		fs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), fs)
	})

	It("should set name", func() {
		opts := &set.Options{
			BuilderOptions: componentarchive.BuilderOptions{
				ComponentArchivePath: "./00-component",
				Name:                 "xxxx.xx/name",
			},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("ubuntu"),
			"Version":       Equal("v0.0.1"),
			"Type":          Equal("ociImage"),
			"ExtraIdentity": HaveLen(0),
		}))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ResourceRelation("external")),
		}))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("type", "ociRegistry"))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("imageReference", "ubuntu:18.0"))

		Expect(cd.Name).To(Equal("xxxx.xx/name"))
		Expect(cd.Version).To(Equal("v0.0.0"))
	})

	It("should set version", func() {
		opts := &set.Options{
			BuilderOptions: componentarchive.BuilderOptions{
				ComponentArchivePath: "./00-component",
				Version:              "v1",
			},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("ubuntu"),
			"Version":       Equal("v0.0.1"),
			"Type":          Equal("ociImage"),
			"ExtraIdentity": HaveLen(0),
		}))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ResourceRelation("external")),
		}))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("type", "ociRegistry"))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("imageReference", "ubuntu:18.0"))

		Expect(cd.Name).To(Equal("example.com/component"))
		Expect(cd.Version).To(Equal("v1"))
	})

})
