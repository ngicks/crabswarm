// Package main is the entry point for the crabhook CLI.
package main

import (
	"os"

	"github.com/ngicks/crabswarm/hook/cmd/crabhook/internal"
)

func main() {
	if err := internal.Execute(); err != nil {
		os.Exit(1)
	}
}
