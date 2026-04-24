// Package cmd assembles the cobra command tree for the jobs binary.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "jobs",
	Short: "market-digest jobs binary — fetch, compute, alert",
	Long: `market-digest jobs binary.
All subcommands read/write data/digest.db (resolved relative to DIGEST_HOME).
Exit codes: 0 ok, 1 error, 2 noop.`,
}

// Execute runs the root command. Called from main.
func Execute() error {
	return rootCmd.Execute()
}

// digestHome returns DIGEST_HOME or CWD.
func digestHome() string {
	if h := os.Getenv("DIGEST_HOME"); h != "" {
		return h
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "jobs: cannot resolve cwd:", err)
		os.Exit(1)
	}
	return cwd
}
