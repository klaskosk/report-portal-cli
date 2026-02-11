package main

import (
	"os"

	"github.com/klaskosk/report-portal-cli/internal/cli"
)

// main executes the CLI root command and exits non-zero on failure.
func main() {
	err := cli.Execute()
	if err != nil {
		os.Exit(1)
	}
}
