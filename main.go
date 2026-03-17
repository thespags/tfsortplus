package main

import (
	"os"

	"github.com/thespags/tfsortplus/cmd"
)

func main() {
	var err error

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if err := cmd.RootCmd(cwd).Execute(); err != nil {
		os.Exit(1)
	}
}
