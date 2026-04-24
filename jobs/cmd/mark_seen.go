package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/nkim500/market-digest/internal/db"
)

var markSeenCmd = &cobra.Command{
	Use:   "mark-seen <alert-id>",
	Short: "Mark an alert as seen",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("parse id: %w", err)
		}
		ctx := context.Background()
		conn, err := db.Open(ctx, filepath.Join(digestHome(), "data", "digest.db"))
		if err != nil {
			return err
		}
		defer conn.Close()
		res, err := conn.ExecContext(ctx,
			`UPDATE alerts SET seen_ts=? WHERE id=? AND seen_ts IS NULL`,
			time.Now().Unix(), id,
		)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			fmt.Println("no change (already seen or unknown id)")
			return nil
		}
		fmt.Printf("alert %d marked seen\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(markSeenCmd)
}
