package signature

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/gardener/component-cli/pkg/commands/componentarchive/signature/sign"
	"github.com/gardener/component-cli/pkg/commands/componentarchive/signature/verify"
)

// NewSignatureCommand creates a new command to interact with signatures.
func NewSignatureCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signature",
		Short: "[EXPERIMENTAL] command to work with signatures and digests in component-descriptors",
	}

	cmd.AddCommand(NewAddDigestsCommand(ctx))
	cmd.AddCommand(NewCheckDigest(ctx))
	cmd.AddCommand(sign.NewSignCommand(ctx))
	cmd.AddCommand(verify.NewVerifyCommand(ctx))

	return cmd
}
