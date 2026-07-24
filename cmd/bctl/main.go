package main

import (
	"fmt"
	"os"

	"github.com/jamesonstone/beacon/internal/cli"
)

func main() {
	if err := cli.NewBctl().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(cli.ExitCode(err))
	}
}
