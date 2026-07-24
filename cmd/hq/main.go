// Command hq is the HQ CLI entry point. Phase 0 implements read-only
// commands: version, health, context, get, list, and search.
package main

import (
	"os"

	"github.com/ryannmicua/hq-cli/internal/app"
)

func main() {
	code := app.Run(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}
