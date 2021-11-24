// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package testutils

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"time"

	. "github.com/onsi/gomega"
)

// CreateTARArchive creates a TAR archive which contains a defined set of files
func CreateTARArchive(files map[string][]byte) *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})
	tw := tar.NewWriter(buf)
	defer tw.Close()

	for filename, content := range files {
		h := tar.Header{
			Name:    filename,
			Size:    int64(len(content)),
			Mode:    0600,
			ModTime: time.Now(),
		}

		Expect(tw.WriteHeader(&h)).To(Succeed())
		_, err := tw.Write(content)
		Expect(err).ToNot(HaveOccurred())
	}

	return buf
}

// CheckTARArchive checks that a TAR archive contains a defined set of files
func CheckTARArchive(archiveReader io.Reader, expectedFiles map[string][]byte) {
	tr := tar.NewReader(archiveReader)

	expectedFilesCopy := map[string][]byte{}
	for key, value := range expectedFiles {
		expectedFilesCopy[key] = value
	}

	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			Expect(err).ToNot(HaveOccurred())
		}

		actualContentBuf := bytes.NewBuffer([]byte{})
		_, err = io.Copy(actualContentBuf, tr)
		Expect(err).ToNot(HaveOccurred())

		expectedContent, ok := expectedFilesCopy[header.Name]
		Expect(ok).To(BeTrue(), fmt.Sprintf("file \"%s\" is not included in expected files", header.Name))
		Expect(actualContentBuf.Bytes()).To(Equal(expectedContent))

		delete(expectedFilesCopy, header.Name)
	}

	Expect(expectedFilesCopy).To(BeEmpty(), fmt.Sprintf("unable to find all expected files in TAR archive. missing files = %+v", expectedFilesCopy))
}
