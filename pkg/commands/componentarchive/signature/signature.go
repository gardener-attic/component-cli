package signature

import (
	"context"

	"github.com/spf13/cobra"
)

// NewSignatureCommand creates a new command to interact with signatures.
func NewSignatureCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signature",
		Short: "command to work with signatures and digests in component-descriptors",
	}

	cmd.AddCommand(NewAddDigestsCommand(ctx))
	cmd.AddCommand(NewRSAVerifyCommand(ctx))
	cmd.AddCommand(NewNotaryVerifyCommand(ctx))
	cmd.AddCommand(NewCompareCommand(ctx))

	return cmd
}
