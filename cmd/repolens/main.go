package main

import (
	"os"

	"github.com/ConteMan/repolens/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
