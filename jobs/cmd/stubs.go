package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var ErrNotImplemented = errors.New("not implemented — see modes/momentum.md, modes/sector.md for context")

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "fetch-prices",
		Short: "(stub) Fetch EOD prices for watchlist — planned for momentum mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "fetch-prices is not implemented yet. See modes/momentum.md.")
			return ErrNotImplemented
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "compute-momentum",
		Short: "(stub) Compute momentum signals over prices table",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "compute-momentum is not implemented yet.")
			return ErrNotImplemented
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "report-sector <sector>",
		Short: "(stub) Generate a sector snapshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "report-sector is not implemented yet.")
			return ErrNotImplemented
		},
	})
}
