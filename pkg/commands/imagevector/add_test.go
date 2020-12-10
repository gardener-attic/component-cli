// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imagevector_test

import (
	"context"
	"encoding/json"
	"path/filepath"

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

	"github.com/gardener/component-cli/pkg/commands/imagevector"
)

var _ = Describe("Add", func() {

	var testdataFs vfs.FileSystem

	BeforeEach(func() {
		fs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), fs)
	})

	It("should add a image source with tag", func() {

		opts := &imagevector.AddOptions{
			ComponentArchivePath: "./00-component",
			ImageVectorPath:      "./resources/00-tag.yaml",
		}

		Expect(opts.Run(context.TODO(), testlog.NullLogger{}, testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("pause-container"),
			"Version": Equal("3.1"),
		}))
	})

	It("should add a image source with a label", func() {

		opts := &imagevector.AddOptions{
			ComponentArchivePath: "./00-component",
			ImageVectorPath:      "./resources/01-labels.yaml",
		}

		Expect(opts.Run(context.TODO(), testlog.NullLogger{}, testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("pause-container"),
			"Version": Equal("3.1"),
			"Labels": ContainElement(cdv2.Label{
				Name:  "my-label",
				Value: json.RawMessage(`"myval"`),
			}),
		}))
	})

	It("should add a image source with tag and target version", func() {

		opts := &imagevector.AddOptions{
			ComponentArchivePath: "./00-component",
			ImageVectorPath:      "./resources/10-targetversion.yaml",
		}

		Expect(opts.Run(context.TODO(), testlog.NullLogger{}, testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("metrics-server"),
			"Version":       Equal("v0.4.1"),
			"ExtraIdentity": HaveKeyWithValue(imagevector.TagExtraIdentity, "v0.4.1"),
		}))
	})

	It("should add two image sources with different target versions", func() {

		opts := &imagevector.AddOptions{
			ComponentArchivePath: "./00-component",
			ImageVectorPath:      "./resources/11-multi-targetversion.yaml",
		}

		Expect(opts.Run(context.TODO(), testlog.NullLogger{}, testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(2))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("metrics-server"),
			"Version":       Equal("v0.4.1"),
			"ExtraIdentity": HaveKeyWithValue(imagevector.TagExtraIdentity, "v0.4.1"),
		}))

		Expect(cd.Resources[1]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[1].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("metrics-server"),
			"Version":       Equal("v0.3.1"),
			"ExtraIdentity": HaveKeyWithValue(imagevector.TagExtraIdentity, "v0.3.1"),
		}))
	})

	It("should add image sources that match a given pattern as component reference", func() {

		opts := &imagevector.AddOptions{
			ComponentArchivePath: "./00-component",
			ImageVectorPath:      "./resources/20-comp-ref.yaml",
			ParseImageOptions: imagevector.ParseImageOptions{
				ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
			},
		}

		Expect(opts.Run(context.TODO(), testlog.NullLogger{}, testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(0))
		Expect(cd.ComponentReferences).To(HaveLen(1))
		Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("cluster-autoscaler"),
			"ComponentName": Equal("github.com/gardener/autoscaler"),
			"Version":       Equal("v0.10.0"),
		}))

		imageLabelBytes, ok := cd.ComponentReferences[0].GetLabels().Get(imagevector.ImagesLabel)
		Expect(ok).To(BeTrue())
		imageVector := &imagevector.ImageVector{}
		Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
		Expect(imageVector.Images).To(HaveLen(1))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("cluster-autoscaler"),
			"Tag":  PointTo(Equal("v0.10.0")),
		})))
	})

	It("should add two image sources that match a given pattern as component reference", func() {

		opts := &imagevector.AddOptions{
			ComponentArchivePath: "./00-component",
			ImageVectorPath:      "./resources/21-multi-comp-ref.yaml",
			ParseImageOptions: imagevector.ParseImageOptions{
				ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
			},
		}

		Expect(opts.Run(context.TODO(), testlog.NullLogger{}, testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(0))
		Expect(cd.ComponentReferences).To(HaveLen(2))
		Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("cluster-autoscaler"),
			"ComponentName": Equal("github.com/gardener/autoscaler"),
			"Version":       Equal("v0.13.0"),
		}))
		Expect(cd.ComponentReferences[1]).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("cluster-autoscaler"),
			"ComponentName": Equal("github.com/gardener/autoscaler"),
			"Version":       Equal("v0.10.1"),
		}))

		imageLabelBytes, ok := cd.ComponentReferences[1].GetLabels().Get(imagevector.ImagesLabel)
		Expect(ok).To(BeTrue())
		imageVector := &imagevector.ImageVector{}
		Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
		Expect(imageVector.Images).To(HaveLen(1))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("cluster-autoscaler"),
			"Tag":  PointTo(Equal("v0.10.1")),
		})))

		imageLabelBytes, ok = cd.ComponentReferences[0].GetLabels().Get(imagevector.ImagesLabel)
		Expect(ok).To(BeTrue())
		imageVector = &imagevector.ImageVector{}
		Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
		Expect(imageVector.Images).To(HaveLen(1))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("cluster-autoscaler"),
			"Tag":  PointTo(Equal("v0.13.0")),
		})))
	})

	It("should add two image sources that match a given pattern as one component reference", func() {

		opts := &imagevector.AddOptions{
			ComponentArchivePath: "./00-component",
			ImageVectorPath:      "./resources/22-multi-image-comp-ref.yaml",
			ParseImageOptions: imagevector.ParseImageOptions{
				ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
			},
		}

		Expect(opts.Run(context.TODO(), testlog.NullLogger{}, testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(0))
		Expect(cd.ComponentReferences).To(HaveLen(1))
		Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("cluster-autoscaler"),
			"ComponentName": Equal("github.com/gardener/autoscaler"),
			"Version":       Equal("v0.13.0"),
		}))

		imageLabelBytes, ok := cd.ComponentReferences[0].GetLabels().Get(imagevector.ImagesLabel)
		Expect(ok).To(BeTrue())
		imageVector := &imagevector.ImageVector{}
		Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
		Expect(imageVector.Images).To(HaveLen(2))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":       Equal("cluster-autoscaler"),
			"Repository": Equal("eu.gcr.io/gardener-project/gardener/autoscaler/cluster-autoscaler"),
			"Tag":        PointTo(Equal("v0.13.0")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":       Equal("cluster-autoscaler"),
			"Repository": Equal("eu.gcr.io/gardener-project/gardener/autoscaler/old"),
			"Tag":        PointTo(Equal("v0.13.0")),
		})))
	})

	It("should add two image sources that match a given pattern as one component reference", func() {
		opts := &imagevector.AddOptions{
			ParseImageOptions: imagevector.ParseImageOptions{
				GenericDependencies: []string{
					"hyperkube",
				},
			},
		}
		cd := runAdd(testdataFs, "./00-component", "./resources/30-generic.yaml", opts)

		Expect(cd.Resources).To(HaveLen(0))
		Expect(cd.ComponentReferences).To(HaveLen(0))

		imageLabelBytes, ok := cd.GetLabels().Get(imagevector.ImagesLabel)
		Expect(ok).To(BeTrue())
		imageVector := &imagevector.ImageVector{}
		Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
		Expect(imageVector.Images).To(HaveLen(2))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("hyperkube"),
			"Repository":    Equal("k8s.gcr.io/hyperkube"),
			"TargetVersion": PointTo(Equal("< 1.19")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("hyperkube"),
			"Repository":    Equal("eu.gcr.io/gardener-project/hyperkube"),
			"TargetVersion": PointTo(Equal(">= 1.19")),
		})))
	})

})

// runAdd runs the add command
func runAdd(fs vfs.FileSystem, caPath, ivPath string, addOpts ...*imagevector.AddOptions) *cdv2.ComponentDescriptor {
	Expect(len(addOpts) <= 1).To(BeTrue())
	opts := &imagevector.AddOptions{}
	if len(addOpts) == 1 {
		opts = addOpts[0]
	}
	opts.ComponentArchivePath = caPath
	opts.ImageVectorPath = ivPath

	Expect(opts.Run(context.TODO(), testlog.NullLogger{}, fs)).To(Succeed())

	data, err := vfs.ReadFile(fs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
	Expect(err).ToNot(HaveOccurred())

	cd := &cdv2.ComponentDescriptor{}
	Expect(codec.Decode(data, cd)).To(Succeed())
	return cd
}
