package main

import (
	"os"

	"github.com/klaskosk/report-portal-cli/internal/cli"
)

func main() {
	err := cli.Execute()
	if err != nil {
		os.Exit(1)
	}
}
