package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/nkim500/market-digest/internal/db"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply pending SQL migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		home := digestHome()
		ctx := context.Background()
		conn, err := db.Open(ctx, filepath.Join(home, "data", "digest.db"))
		if err != nil {
			return err
		}
		defer conn.Close()
		applied, err := db.Migrate(ctx, conn, filepath.Join(home, "migrations"))
		if err != nil {
			return err
		}
		if len(applied) == 0 {
			fmt.Println("migrate: up to date")
			return nil
		}
		for _, f := range applied {
			fmt.Printf("migrate: applied %s\n", f)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
