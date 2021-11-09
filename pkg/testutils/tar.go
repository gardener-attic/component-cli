// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package testutils

import (
	"archive/tar"
	"bytes"
	"io"
	"time"

	. "github.com/onsi/gomega"
)

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

func CheckTARArchive(r io.Reader, expectedFiles map[string][]byte) {
	tr := tar.NewReader(r)

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

		expectedContent, ok := expectedFiles[header.Name]
		Expect(ok).To(BeTrue())
		Expect(actualContentBuf.Bytes()).To(Equal(expectedContent))

		delete(expectedFiles, header.Name)
	}

	Expect(expectedFiles).To(BeEmpty())
}
