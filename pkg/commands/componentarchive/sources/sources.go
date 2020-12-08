// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"context"

	"github.com/spf13/cobra"
)

// NewSourceCommand creates a new command to to modify sources of a component descriptor.
func NewSourceCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "source",
		Aliases: []string{"src"},
		Short:   "command to modify sources of a component descriptor",
	}
	cmd.AddCommand(NewAddCommand(ctx))
	return cmd
}
