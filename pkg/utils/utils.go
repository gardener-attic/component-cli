// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/pkg/commands/constants"
)

// PrintPrettyYaml prints the given objects as yaml if enabled.
func PrintPrettyYaml(obj interface{}, enabled bool) {
	if !enabled {
		return
	}
	data, err := yaml.Marshal(obj)
	if err != nil {
		fmt.Printf("unable to serialize object as yaml: %s", err.Error())
		return
	}
	fmt.Println(string(data))
}

// GetFileType returns the mimetype of a file.
func GetFileType(fs vfs.FileSystem, path string) (string, error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	// see http://golang.org/pkg/net/http/#DetectContentType for the 512 bytes
	buf := make([]byte, 512)
	_, err = file.Read(buf)
	if err != nil {
		return "", err
	}
	return http.DetectContentType(buf), nil
}

// CacheDir returns the cache dir for the current cli command
func CacheDir() (string, error) {
	defaultCacheDir := os.Getenv(cache.CacheDirEnvName)
	if len(defaultCacheDir) != 0 {
		return defaultCacheDir, nil
	}

	cliHomeDir, err := constants.CliHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(cliHomeDir, "components")
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("unable to create cache directory %s: %w", cacheDir, err)
	}

	return cacheDir, nil
}

// CleanMarkdownUsageFunc removes markdown tags from the long usage of the command.
// With this func it is possible to generate the markdown docs but still have readable commandline help func.
// Note: currently only "<pre>" tags are removed
func CleanMarkdownUsageFunc(cmd *cobra.Command) {
	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, s []string) {
		cmd.Long = strings.ReplaceAll(cmd.Long, "<pre>", "")
		cmd.Long = strings.ReplaceAll(cmd.Long, "</pre>", "")
		defaultHelpFunc(cmd, s)
	})
}

// RawJSON converts an arbitrary value to json.RawMessage
func RawJSON(value interface{}) (*json.RawMessage, error) {
	jsonval, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return (*json.RawMessage)(&jsonval), nil
}

// Gzip applies gzip compression to an arbitrary byte slice
func Gzip(data []byte, compressionLevel int) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	gzipWriter, err := gzip.NewWriterLevel(buf, compressionLevel)
	if err != nil {
		return nil, fmt.Errorf("unable to create gzip writer: %w", err)
	}
	defer gzipWriter.Close()

	_, err = gzipWriter.Write(data)
	if err != nil {
		return nil, fmt.Errorf("unable to write to stream: %w", err)
	}

	err = gzipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("unable to close writer: %w", err)
	}

	return buf.Bytes(), nil
}

var chars = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

// RandomString creates a new random string with the given length.
func RandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
