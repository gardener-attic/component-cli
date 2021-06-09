// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imagevector_test

import (
	"context"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/gardener/component-cli/pkg/components"

	ivcmd "github.com/gardener/component-cli/pkg/commands/imagevector"
	"github.com/gardener/component-cli/pkg/imagevector"
)

var _ = Describe("GenerateOverwrite", func() {

	var testdataFs vfs.FileSystem

	BeforeEach(func() {
		fs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), fs)
	})

	It("should generate a simple image with tag from a component descriptor", func() {
		imageVector := runGenerateOverwrite(testdataFs, "./01-component/component-descriptor.yaml")

		Expect(imageVector.Images).To(HaveLen(3))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("pause-container"),
			"Tag":  PointTo(Equal("3.1")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("pause-container"),
			"Tag":  PointTo(Equal("sha256:eb9086d472747453ad2d5cfa10f80986d9b0afb9ae9c4256fe2887b029566d06")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("gardenlet"),
			"Tag":  PointTo(Equal("v0.0.0")),
		})))
	})

	It("should generate a image source with a target version", func() {
		runAdd(testdataFs, "./00-component/component-descriptor.yaml", "./resources/10-targetversion.yaml")
		imageVector := runGenerateOverwrite(testdataFs, "./00-component/component-descriptor.yaml")
		Expect(imageVector.Images).To(HaveLen(1))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("metrics-server"),
			"Tag":           PointTo(Equal("v0.4.1")),
			"TargetVersion": PointTo(Equal(">= 1.11")),
		})))
	})

	It("should generate image sources from component references", func() {
		opts := &ivcmd.AddOptions{
			ParseImageOptions: imagevector.ParseImageOptions{
				ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
			},
		}
		runAdd(testdataFs, "./00-component/component-descriptor.yaml", "./resources/21-multi-comp-ref.yaml", opts)

		getOpts := &ivcmd.GenerateOverwriteOptions{}
		getOpts.ComponentDescriptorsPath = []string{
			"./02-autoscaler-0.10.1/component-descriptor.yaml",
			"./03-autoscaler-0.13.0/component-descriptor.yaml",
		}
		imageVector := runGenerateOverwrite(testdataFs, "./00-component/component-descriptor.yaml", getOpts)
		Expect(imageVector.Images).To(HaveLen(2))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("cluster-autoscaler"),
			"Tag":           PointTo(Equal("v0.13.0")),
			"TargetVersion": PointTo(Equal(">= 1.16")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("cluster-autoscaler"),
			"Tag":           PointTo(Equal("v0.10.1")),
			"TargetVersion": PointTo(Equal("< 1.16")),
		})))
	})

	It("should generate image sources from generic images", func() {
		addOpts := &ivcmd.AddOptions{
			ParseImageOptions: imagevector.ParseImageOptions{
				GenericDependencies: []string{
					"hyperkube",
				},
			},
		}
		runAdd(testdataFs, "./00-component/component-descriptor.yaml", "./resources/30-generic.yaml", addOpts)

		getOpts := &ivcmd.GenerateOverwriteOptions{}
		getOpts.ComponentDescriptorsPath = []string{"./04-generic-images/component-descriptor.yaml"}
		imageVector := runGenerateOverwrite(testdataFs, "./00-component/component-descriptor.yaml", getOpts)
		Expect(imageVector.Images).To(HaveLen(3))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("hyperkube"),
			"Repository":    Equal("eu.gcr.io/gardener-project/hyperkube"),
			"Tag":           PointTo(Equal("v1.19.2")),
			"TargetVersion": PointTo(Equal("= v1.19.2")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("hyperkube"),
			"Repository":    Equal("k8s.gcr.io/hyperkube"),
			"Tag":           PointTo(Equal("v1.18.6")),
			"TargetVersion": PointTo(Equal("= v1.18.6")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("hyperkube"),
			"Repository":    Equal("k8s.gcr.io/hyperkube"),
			"Tag":           PointTo(Equal("v1.17.10")),
			"TargetVersion": PointTo(Equal("= v1.17.10")),
		})))
	})

})

func runGenerateOverwrite(fs vfs.FileSystem, caPath string, getOpts ...*ivcmd.GenerateOverwriteOptions) *imagevector.ImageVector {
	Expect(len(getOpts) <= 1).To(BeTrue())
	opts := &ivcmd.GenerateOverwriteOptions{}
	if len(getOpts) == 1 {
		opts = getOpts[0]
	}
	opts.ComponentDescriptorPath = caPath
	opts.ImageVectorPath = "./out/imagevector.yaml"
	Expect(opts.Complete(nil)).To(Succeed())

	// fake local cache with given component descriptor
	data, err := vfs.ReadFile(fs, caPath)
	Expect(err).ToNot(HaveOccurred())
	cd := &cdv2.ComponentDescriptor{}
	Expect(codec.Decode(data, cd)).To(Succeed())
	Expect(os.Setenv(constants.ComponentRepositoryCacheDirEnvVar, "/tmp/components")).To(Succeed())
	cdCachePath := components.LocalCachePath(cd.GetEffectiveRepositoryContext(), cd.Name, cd.Version)
	Expect(fs.MkdirAll(filepath.Dir(cdCachePath), os.ModePerm)).To(Succeed())
	Expect(vfs.WriteFile(fs, cdCachePath, data, os.ModePerm)).To(Succeed())

	Expect(opts.Run(context.TODO(), logr.Discard(), fs)).To(Succeed())

	data, err = vfs.ReadFile(fs, opts.ImageVectorPath)
	Expect(err).ToNot(HaveOccurred())

	imageVector := &imagevector.ImageVector{}
	Expect(yaml.Unmarshal(data, imageVector)).To(Succeed())
	return imageVector
}
