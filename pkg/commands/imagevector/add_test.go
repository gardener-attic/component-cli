// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imagevector_test

import (
	"context"
	"encoding/json"

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

	ivcmd "github.com/gardener/component-cli/pkg/commands/imagevector"
	"github.com/gardener/component-cli/pkg/imagevector"
)

var _ = Describe("Add", func() {

	var testdataFs vfs.FileSystem

	BeforeEach(func() {
		fs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), fs)
	})

	It("should add a image source with tag", func() {

		opts := &ivcmd.AddOptions{
			ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
			ImageVectorPath:         "./resources/00-tag.yaml",
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("pause-container"),
			"Version":       Equal("3.1"),
			"ExtraIdentity": HaveKeyWithValue(imagevector.TagExtraIdentity, "3.1"),
			"Labels": ContainElements(
				cdv2.Label{
					Name:  imagevector.NameLabel,
					Value: json.RawMessage(`"pause-container"`),
				},
				cdv2.Label{
					Name:  imagevector.RepositoryLabel,
					Value: json.RawMessage(`"gcr.io/google_containers/pause-amd64"`),
				},
				cdv2.Label{
					Name:  imagevector.SourceRepositoryLabel,
					Value: json.RawMessage(`"github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile"`),
				},
			),
		}))
		Expect(cd.Resources[0].Access.Object).To(MatchKeys(IgnoreExtras, Keys{
			"imageReference": Equal("gcr.io/google_containers/pause-amd64:3.1"),
		}))
	})

	It("should add a image source with a digest as tag", func() {

		opts := &ivcmd.AddOptions{
			ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
			ImageVectorPath:         "./resources/03-sha.yaml",
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("pause-container"),
			"Version":       Equal("v0.0.0"),
			"ExtraIdentity": HaveKeyWithValue(imagevector.TagExtraIdentity, "sha256:179e67c248007299e05791db36298c41cbf0992372204a68473e12795a51b06b"),
			"Labels": ContainElements(
				cdv2.Label{
					Name:  imagevector.NameLabel,
					Value: json.RawMessage(`"pause-container"`),
				},
				cdv2.Label{
					Name:  imagevector.RepositoryLabel,
					Value: json.RawMessage(`"gcr.io/google_containers/pause-amd64"`),
				},
				cdv2.Label{
					Name:  imagevector.SourceRepositoryLabel,
					Value: json.RawMessage(`"github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile"`),
				},
			),
		}))
		Expect(cd.Resources[0].Access.Object).To(MatchKeys(IgnoreExtras, Keys{
			"imageReference": Equal("gcr.io/google_containers/pause-amd64@sha256:179e67c248007299e05791db36298c41cbf0992372204a68473e12795a51b06b"),
		}))
	})

	It("should add a image source with a label", func() {

		opts := &ivcmd.AddOptions{
			ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
			ImageVectorPath:         "./resources/01-labels.yaml",
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
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
			"Labels": ContainElements(
				cdv2.Label{
					Name:  "my-label",
					Value: json.RawMessage(`"myval"`),
				},
				cdv2.Label{
					Name:  imagevector.NameLabel,
					Value: json.RawMessage(`"pause-container"`),
				},
				cdv2.Label{
					Name:  imagevector.RepositoryLabel,
					Value: json.RawMessage(`"gcr.io/google_containers/pause-amd64"`),
				},
				cdv2.Label{
					Name:  imagevector.SourceRepositoryLabel,
					Value: json.RawMessage(`"github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile"`),
				},
			),
		}))
	})

	It("should add imagevector labels for inline image definitions", func() {

		opts := &ivcmd.AddOptions{
			ComponentDescriptorPath: "./05-inline/component-descriptor.yaml",
			ImageVectorPath:         "./resources/02-inline.yaml",
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.LocalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("gardenlet"),
			"Version": Equal("v0.0.0"),
			"Labels": ContainElements(
				cdv2.Label{
					Name:  imagevector.NameLabel,
					Value: json.RawMessage(`"gardenlet"`),
				},
				cdv2.Label{
					Name:  imagevector.RepositoryLabel,
					Value: json.RawMessage(`"eu.gcr.io/gardener-project/gardener/gardenlet"`),
				},
				cdv2.Label{
					Name:  imagevector.SourceRepositoryLabel,
					Value: json.RawMessage(`"github.com/gardener/gardener"`),
				},
			),
		}))
	})

	It("should add a image source with tag and target version", func() {

		opts := &ivcmd.AddOptions{
			ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
			ImageVectorPath:         "./resources/10-targetversion.yaml",
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
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

		opts := &ivcmd.AddOptions{
			ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
			ImageVectorPath:         "./resources/11-multi-targetversion.yaml",
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
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

	Context("ComponentReferences", func() {
		It("should add image sources that match a given pattern as component reference", func() {
			opts := &ivcmd.AddOptions{
				ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
				ImageVectorPath:         "./resources/20-comp-ref.yaml",
				ParseImageOptions: imagevector.ParseImageOptions{
					ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
				},
			}

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

			data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			Expect(cd.Resources).To(HaveLen(0))
			Expect(cd.ComponentReferences).To(HaveLen(1))
			Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("cluster-autoscaler"),
				"ComponentName": Equal("github.com/gardener/autoscaler"),
				"Version":       Equal("v0.10.0"),
				"ExtraIdentity": HaveKeyWithValue("imagevector-gardener-cloud+tag", "v0.10.0"),
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

		It("should add image sources with the component reference label as component reference", func() {
			opts := &ivcmd.AddOptions{
				ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
				ImageVectorPath:         "./resources/23-comp-ref-label.yaml",
			}

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

			data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			Expect(cd.Resources).To(HaveLen(0))
			Expect(cd.ComponentReferences).To(HaveLen(1))
			Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("cluster-autoscaler"),
				"ComponentName": Equal("github.com/gardener/autoscaler"),
				"Version":       Equal("v0.10.0"),
				"ExtraIdentity": HaveKeyWithValue("imagevector-gardener-cloud+tag", "v0.10.0"),
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

		It("should add image sources with the component reference label and overwrites as component reference", func() {
			opts := &ivcmd.AddOptions{
				ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
				ImageVectorPath:         "./resources/24-comp-ref-label.yaml",
			}

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

			data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			Expect(cd.Resources).To(HaveLen(0))
			Expect(cd.ComponentReferences).To(HaveLen(1))
			Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("cla"),
				"ComponentName": Equal("example.com/autoscaler"),
				"Version":       Equal("v0.0.1"),
				"ExtraIdentity": HaveKeyWithValue("imagevector-gardener-cloud+tag", "v0.0.1"),
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

		It("should not add image sources that match a given pattern as component reference but has a ignore label", func() {
			opts := &ivcmd.AddOptions{
				ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
				ImageVectorPath:         "./resources/25-comp-ref-ignore-label.yaml",
				ParseImageOptions: imagevector.ParseImageOptions{
					ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
				},
			}

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

			data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			Expect(cd.Resources).To(HaveLen(1))
			Expect(cd.ComponentReferences).To(HaveLen(0))
		})

		It("should not add a image sources that match a given pattern as component reference but is excluded", func() {

			opts := &ivcmd.AddOptions{
				ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
				ImageVectorPath:         "./resources/20-comp-ref.yaml",
				ParseImageOptions: imagevector.ParseImageOptions{
					ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
					ExcludeComponentReference:  []string{"cluster-autoscaler"},
				},
			}

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

			data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			Expect(cd.ComponentReferences).To(HaveLen(0))
			Expect(cd.Resources).To(HaveLen(1))
			Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
				"Relation": Equal(cdv2.ExternalRelation),
			}))
			Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("cluster-autoscaler"),
				"Version":       Equal("v0.10.0"),
				"ExtraIdentity": HaveKeyWithValue(imagevector.TagExtraIdentity, "v0.10.0"),
			}))
		})

		It("should add two image sources that match a given pattern as component reference", func() {

			opts := &ivcmd.AddOptions{
				ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
				ImageVectorPath:         "./resources/21-multi-comp-ref.yaml",
				ParseImageOptions: imagevector.ParseImageOptions{
					ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
				},
			}

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

			data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
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

			opts := &ivcmd.AddOptions{
				ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
				ImageVectorPath:         "./resources/22-multi-image-comp-ref.yaml",
				ParseImageOptions: imagevector.ParseImageOptions{
					ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
				},
			}

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

			data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			Expect(cd.Resources).To(HaveLen(0))
			Expect(cd.ComponentReferences).To(HaveLen(1))
			Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("cluster-autoscaler"),
				"ComponentName": Equal("github.com/gardener/autoscaler"),
				"Version":       Equal("v0.13.0"),
				"ExtraIdentity": And(HaveKey(imagevector.TagExtraIdentity), Not(HaveKey("name"))),
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
	})

	Context("Generic Dependencies", func() {

		It("should add generic sources that match a given generic dependency name", func() {
			opts := &ivcmd.AddOptions{
				ParseImageOptions: imagevector.ParseImageOptions{
					GenericDependencies: []string{
						"hyperkube",
					},
				},
			}
			cd := runAdd(testdataFs, "./00-component/component-descriptor.yaml", "./resources/30-generic.yaml", opts)

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

		It("should add generic sources that match a given generic dependency name defined by a list of dependencies", func() {
			opts := &ivcmd.AddOptions{
				GenericDependencies: "hyperkube",
			}
			cd := runAdd(testdataFs, "./00-component/component-descriptor.yaml", "./resources/30-generic.yaml", opts)

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

		It("should add generic sources that are labeled", func() {
			opts := &ivcmd.AddOptions{}
			cd := runAdd(testdataFs, "./00-component/component-descriptor.yaml", "./resources/31-generic-labels.yaml", opts)

			Expect(cd.Resources).To(HaveLen(0))
			Expect(cd.ComponentReferences).To(HaveLen(0))

			imageLabelBytes, ok := cd.GetLabels().Get(imagevector.ImagesLabel)
			Expect(ok).To(BeTrue())
			imageVector := &imagevector.ImageVector{}
			Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
			Expect(imageVector.Images).To(HaveLen(1))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("hyperkube"),
				"Repository":    Equal("k8s.gcr.io/hyperkube"),
				"TargetVersion": PointTo(Equal("< 1.19")),
			})))
		})

	})

})

// runAdd runs the add command
func runAdd(fs vfs.FileSystem, caPath, ivPath string, addOpts ...*ivcmd.AddOptions) *cdv2.ComponentDescriptor {
	Expect(len(addOpts) <= 1).To(BeTrue())
	opts := &ivcmd.AddOptions{}
	if len(addOpts) == 1 {
		opts = addOpts[0]
	}
	opts.ComponentDescriptorPath = caPath
	opts.ImageVectorPath = ivPath
	Expect(opts.Complete(nil)).To(Succeed())

	Expect(opts.Run(context.TODO(), logr.Discard(), fs)).To(Succeed())

	data, err := vfs.ReadFile(fs, opts.ComponentDescriptorPath)
	Expect(err).ToNot(HaveOccurred())

	cd := &cdv2.ComponentDescriptor{}
	Expect(codec.Decode(data, cd)).To(Succeed())
	return cd
}
