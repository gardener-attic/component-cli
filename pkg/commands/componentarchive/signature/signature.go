package signature

import (
	"context"

	"github.com/gardener/component-cli/pkg/commands/componentarchive/signature/sign"
	"github.com/gardener/component-cli/pkg/commands/componentarchive/signature/verify"
	"github.com/spf13/cobra"
)

// NewSignatureCommand creates a new command to interact with signatures.
func NewSignatureCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signature",
		Short: "command to work with signatures and digests in component-descriptors",
	}

	cmd.AddCommand(NewAddDigestsCommand(ctx))
	cmd.AddCommand(NewCheckDigest(ctx))
	cmd.AddCommand(NewCompareCommand(ctx))
	cmd.AddCommand(sign.NewSignCommand(ctx))
	cmd.AddCommand(verify.NewVerifyCommand(ctx))

	return cmd
}
