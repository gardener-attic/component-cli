// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentreferences_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf"
	testlog "github.com/go-logr/logr/testing"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"github.com/gardener/component-cli/pkg/commands/componentreferences"
	"github.com/gardener/component-cli/pkg/commands/resources"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ComponentReferences Test Suite")
}

var _ = Describe("Add", func() {

	var testdataFs vfs.FileSystem

	BeforeEach(func() {
		var err error
		testdataFs, err = projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
	})

	Context("From template", func() {
		It("should add a resource defined by a file", func() {
			fs := layerfs.New(memoryfs.New(), testdataFs)
			opts := &componentreferences.Options{
				ComponentDescriptorPath:      "./00-component",
				ComponentReferenceObjectPath: "./resources/00-res.yaml",
			}

			Expect(opts.Run(context.TODO(), testlog.NullLogger{}, fs)).To(Succeed())

			data, err := vfs.ReadFile(fs, filepath.Join(opts.ComponentDescriptorPath, ctf.ComponentDescriptorFileName))
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			Expect(cd.ComponentReferences).To(HaveLen(1))
			Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("ubuntu"),
				"ComponentName": Equal("github.com/gardener/ubuntu"),
				"Version":       Equal("v0.0.1"),
			}))
		})

	})

	Context("From arguments", func() {
		var opts *componentreferences.Options

		BeforeEach(func() {
			opts = &componentreferences.Options{
				ComponentDescriptorPath: "./00-component",
				ComponentReferenceOptions: componentreferences.ComponentReferenceOptions{
					Name:          "testres",
					ComponentName: "testname",
					Version:       "v0.0.1",
				},
			}
		})

		It("should add a resource defined by a file", func() {
			fs := layerfs.New(memoryfs.New(), testdataFs)

			Expect(opts.Run(context.TODO(), testlog.NullLogger{}, fs)).To(Succeed())

			data, err := vfs.ReadFile(fs, filepath.Join(opts.ComponentDescriptorPath, ctf.ComponentDescriptorFileName))
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			Expect(cd.ComponentReferences).To(HaveLen(1))
			Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("testres"),
				"ComponentName": Equal("testname"),
				"Version":       Equal("v0.0.1"),
			}))
		})

		It("should set extra identity labels for a resource", func() {
			fs := layerfs.New(memoryfs.New(), testdataFs)
			opts.ComponentReferenceOptions.ExtraIdentity = []string{
				"myid=myotherid",
			}

			Expect(opts.Run(context.TODO(), testlog.NullLogger{}, fs)).To(Succeed())

			data, err := vfs.ReadFile(fs, filepath.Join(opts.ComponentDescriptorPath, ctf.ComponentDescriptorFileName))
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			Expect(cd.ComponentReferences).To(HaveLen(1))
			Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
				"ExtraIdentity": HaveKeyWithValue("myid", "myotherid"),
			}))
		})

		It("should set labels for a resource", func() {
			fs := layerfs.New(memoryfs.New(), testdataFs)
			opts.ComponentReferenceOptions.Labels = []string{
				"mylabel=abc",
				"mysecondlabel={\"key\": true}",
			}

			Expect(opts.Run(context.TODO(), testlog.NullLogger{}, fs)).To(Succeed())

			data, err := vfs.ReadFile(fs, filepath.Join(opts.ComponentDescriptorPath, ctf.ComponentDescriptorFileName))
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			Expect(cd.ComponentReferences).To(HaveLen(1))
			Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
				"Labels": ContainElements(
					MatchFields(0, Fields{
						"Name":  Equal("mylabel"),
						"Value": Equal(json.RawMessage("\"abc\"")),
					}),
					MatchFields(0, Fields{
						"Name":  Equal("mysecondlabel"),
						"Value": Equal(json.RawMessage("{\"key\":true}")),
					}),
				),
			}))
		})

	})

	It("should throw an error if an invalid resource is defined", func() {
		fs := layerfs.New(memoryfs.New(), testdataFs)
		opts := &resources.Options{
			ComponentArchivePath: "./00-component",
			ResourceObjectPath:   "./resources/10-res-invalid.yaml",
		}

		Expect(opts.Run(context.TODO(), testlog.NullLogger{}, fs)).To(HaveOccurred())

		data, err := vfs.ReadFile(fs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())
		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())
		Expect(cd.ComponentReferences).To(HaveLen(0))
	})

})
