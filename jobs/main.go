package main

import (
	"fmt"
	"os"

	"github.com/nkim500/market-digest/jobs/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
