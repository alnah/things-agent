package main

import (
	"os"

	app "github.com/alnah/things-agent/internal/app"
)

func main() {
	cmd := app.NewRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
