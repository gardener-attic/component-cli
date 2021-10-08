// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

//go:generate go run -mod=vendor ../../../hack/generate-docs ../../../docs/reference

package app

import (
	"context"
	"fmt"
	"os"

	cachecmd "github.com/gardener/component-cli/pkg/commands/cache"
	"github.com/gardener/component-cli/pkg/commands/componentarchive"
	"github.com/gardener/component-cli/pkg/commands/ctf"
	"github.com/gardener/component-cli/pkg/commands/imagevector"
	"github.com/gardener/component-cli/pkg/commands/oci"
	"github.com/gardener/component-cli/pkg/commands/transport"
	"github.com/gardener/component-cli/pkg/logcontext"
	"github.com/gardener/component-cli/pkg/logger"
	"github.com/gardener/component-cli/pkg/transport/config"
	"github.com/gardener/component-cli/pkg/version"

	"github.com/spf13/cobra"
)

func NewComponentsCliCommand(ctx context.Context) *cobra.Command {
	ctx, _ = logcontext.NewContext(ctx)
	cmd := &cobra.Command{
		Use:     "component-cli",
		Short:   "component cli",
		Version: version.Get().String(),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log, err := logger.NewCliLogger()
			if err != nil {
				fmt.Println("unable to setup logger")
				fmt.Println(err.Error())
				os.Exit(1)
			}
			logger.SetLogger(logcontext.New(ctx, log))
		},
	}

	logger.InitFlags(cmd.PersistentFlags())

	cmd.AddCommand(NewVersionCommand())
	cmd.AddCommand(ctf.NewCTFCommand(ctx))
	cmd.AddCommand(componentarchive.NewComponentArchiveCommand(ctx))
	cmd.AddCommand(imagevector.NewImageVectorCommand(ctx))
	cmd.AddCommand(oci.NewOCICommand(ctx))
	cmd.AddCommand(cachecmd.NewCacheCommand(ctx))
	cmd.AddCommand(transport.NewTransportCommand(ctx))
	cmd.AddCommand(config.NewConfigParseCommand(ctx))
	cmd.AddCommand(transport.NewTestCommand(ctx))

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
