package main

import (
	"os"

	"github.com/xavier-castro/smsc/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr))
}
