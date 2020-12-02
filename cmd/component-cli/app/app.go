// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/gardener/component-cli/pkg/commands/componentreferences"
	"github.com/gardener/component-cli/pkg/commands/gardenerci"
	"github.com/gardener/component-cli/pkg/commands/remote"
	"github.com/gardener/component-cli/pkg/commands/resources"
	"github.com/gardener/component-cli/pkg/logger"
	"github.com/gardener/component-cli/pkg/version"

	"github.com/spf13/cobra"
)

func NewComponentsCliCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "components-cli",
		Short: "components cli",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log, err := logger.NewCliLogger()
			if err != nil {
				fmt.Println("unable to setup logger")
				fmt.Println(err.Error())
				os.Exit(1)
			}
			logger.SetLogger(log)
		},
	}

	logger.InitFlags(cmd.PersistentFlags())

	cmd.AddCommand(NewVersionCommand())
	cmd.AddCommand(remote.NewRemoteCommand(ctx))
	cmd.AddCommand(resources.NewResourcesCommand(ctx))
	cmd.AddCommand(componentreferences.NewCompRefCommand(ctx))
	cmd.AddCommand(gardenerci.NewGardenerCICommand(ctx))

	return cmd
}

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "displays the version",
		Run: func(cmd *cobra.Command, args []string) {
			v := version.Get()
			fmt.Printf("%#v", v)
		},
	}
}
