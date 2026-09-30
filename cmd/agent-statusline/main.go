package main

import (
	"fmt"
	"os"

	"github.com/joshi008/agent-statusline/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "agent-statusline:", err)
		os.Exit(1)
	}
}
