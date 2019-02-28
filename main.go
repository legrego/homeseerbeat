package main

import (
	"os"

	"github.com/legrego/homeseerbeat/cmd"

	_ "github.com/legrego/homeseerbeat/include"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
