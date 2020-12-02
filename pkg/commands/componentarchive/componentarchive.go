// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive

import (
	"context"

	"github.com/spf13/cobra"
)

// NewComponentArchiveCommand creates a new component archive command.
func NewComponentArchiveCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "component-archive",
		Aliases: []string{"componentarchive", "ca", "archive"},
	}
	cmd.AddCommand(NewExportCommand(ctx))
	return cmd
}
