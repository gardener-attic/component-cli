// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package utils_test

import (
	"archive/tar"
	"bytes"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/component-cli/pkg/testutils"
	"github.com/gardener/component-cli/pkg/utils"
)

var _ = Describe("utils", func() {

	Context("WriteFileToTARArchive", func() {

		It("should write file", func() {
			fname := "testfile"
			content := []byte("testcontent")

			archiveBuf := bytes.NewBuffer([]byte{})
			tw := tar.NewWriter(archiveBuf)

			Expect(utils.WriteFileToTARArchive(fname, bytes.NewReader(content), tw)).To(Succeed())
			Expect(tw.Close()).To(Succeed())

			tr := tar.NewReader(archiveBuf)
			fheader, err := tr.Next()
			Expect(err).ToNot(HaveOccurred())
			Expect(fheader.Name).To(Equal(fname))

			actualContentBuf := bytes.NewBuffer([]byte{})
			_, err = io.Copy(actualContentBuf, tr)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualContentBuf.Bytes()).To(Equal(content))

			_, err = tr.Next()
			Expect(err).To(Equal(io.EOF))
		})

		It("should write empty file", func() {
			fname := "testfile"

			archiveBuf := bytes.NewBuffer([]byte{})
			tw := tar.NewWriter(archiveBuf)

			Expect(utils.WriteFileToTARArchive(fname, bytes.NewReader([]byte{}), tw)).To(Succeed())
			Expect(tw.Close()).To(Succeed())

			tr := tar.NewReader(archiveBuf)
			fheader, err := tr.Next()
			Expect(err).ToNot(HaveOccurred())
			Expect(fheader.Name).To(Equal(fname))

			actualContentBuf := bytes.NewBuffer([]byte{})
			contentLenght, err := io.Copy(actualContentBuf, tr)
			Expect(err).ToNot(HaveOccurred())
			Expect(contentLenght).To(Equal(int64(0)))

			_, err = tr.Next()
			Expect(err).To(Equal(io.EOF))
		})

		It("should return error if filename is empty", func() {
			tw := tar.NewWriter(bytes.NewBuffer([]byte{}))
			inputReader := bytes.NewReader([]byte{})
			Expect(utils.WriteFileToTARArchive("", inputReader, tw)).To(MatchError("filename must not be empty"))
		})

		It("should return error if inputReader is nil", func() {
			tw := tar.NewWriter(bytes.NewBuffer([]byte{}))
			Expect(utils.WriteFileToTARArchive("testfile", nil, tw)).To(MatchError("inputReader must not be nil"))
		})

		It("should return error if outArchive is nil", func() {
			inputReader := bytes.NewReader([]byte{})
			Expect(utils.WriteFileToTARArchive("testfile", inputReader, nil)).To(MatchError("outputWriter must not be nil"))
		})

	})

	Context("FilterTARArchive", func() {

		It("should filter archive", func() {
			removePatterns := []string{
				"second/*",
			}

			inputFiles := map[string][]byte{
				"first/testfile":    []byte("some-content"),
				"second/testfile":   []byte("more-content"),
				"second/testfile-2": []byte("other-content"),
			}

			expectedFiles := map[string][]byte{
				"first/testfile": []byte("some-content"),
			}

			archive := testutils.CreateTARArchive(inputFiles)

			outBuf := bytes.NewBuffer([]byte{})
			Expect(utils.FilterTARArchive(archive, outBuf, removePatterns)).To(Succeed())

			testutils.CheckTARArchive(outBuf, expectedFiles)
		})

		It("should return error if inputReader is nil", func() {
			outWriter := bytes.NewBuffer([]byte{})
			Expect(utils.FilterTARArchive(nil, outWriter, []string{})).To(MatchError("inputReader must not be nil"))
		})

		It("should return error if outputWriter is nil", func() {
			inputReader := bytes.NewReader([]byte{})
			Expect(utils.FilterTARArchive(inputReader, nil, []string{})).To(MatchError("outputWriter must not be nil"))
		})

	})

})
