package main

import (
	"os"

	"github.com/Nerver-zip/pomo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
