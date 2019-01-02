package main

import (
	"github.com/nomadit/antminerbeat/beat/cmd"
	"os"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
