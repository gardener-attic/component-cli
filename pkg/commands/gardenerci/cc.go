// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gardenerci

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/gardener/component-cli/pkg/commands/constants"
)

// NewGardenerCICommand creates a new gardener ci command to interact with the gardener-ci specifics.
func NewGardenerCICommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "gardener-ci",
		Aliases: []string{"cc"},
	}

	cmd.AddCommand(NewInitCommand(ctx))
	return cmd
}

// NewInitCommand creates a new init command that prepares the gardener-ci component-descriptor step to be used with this tool.
// It basically copies the base component descriptor from $BASE_DEFINITION_PATH to COMPONENT_DESCRIPTOR_PATH.
func NewInitCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "init",
		Args: cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			// copy component descriptor from base path to output
			basePath := os.Getenv(constants.BaseDefinitionPathEnvName)
			outputPath := os.Getenv(constants.ComponentDescriptorPathEnvName)

			if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
				fmt.Printf("unable to create directories for %q: %s", outputPath, err.Error())
				os.Exit(1)
			}

			out, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
			if err != nil {
				fmt.Printf("unable to open component descriptor file %q: %s", outputPath, err.Error())
				os.Exit(1)
			}
			defer out.Close()
			base, err := os.Open(basePath)
			if err != nil {
				fmt.Printf("unable to open base component descriptor file %q: %s", basePath, err.Error())
				os.Exit(1)
			}
			defer base.Close()

			if _, err := io.Copy(out, base); err != nil {
				fmt.Printf("unable to copy base component descriptor %q to output component descriptor %q: %s", basePath, outputPath, err.Error())
				os.Exit(1)
			}
			fmt.Printf("Successfully copied component descriptor")
		},
	}

	return cmd
}
