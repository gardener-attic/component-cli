// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package get_test

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/pkg/commands/componentarchive/get"
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

	It("should get name", func() {
		opts := &get.Options{
			BuilderOptions: componentarchive.BuilderOptions{
				ComponentArchivePath: "./00-component",
			},
			Property: "name",
		}

		Expect(opts.Get(context.TODO(), logr.Discard(), testdataFs)).To(Equal("example.com/component"))
	})

	It("should get version", func() {
		opts := &get.Options{
			BuilderOptions: componentarchive.BuilderOptions{
				ComponentArchivePath: "./00-component",
			},
			Property: "version",
		}

		Expect(opts.Get(context.TODO(), logr.Discard(), testdataFs)).To(Equal("v0.0.0"))
	})
})
