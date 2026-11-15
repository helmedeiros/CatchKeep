// Command catchkeep runs the CatchKeep backend.
package main

import (
	"fmt"
	"os"

	"github.com/helmedeiros/catchkeep/backend/internal/buildinfo"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(buildinfo.Version())
		return
	}
	fmt.Fprintln(os.Stderr, "usage: catchkeep version")
	os.Exit(2)
}
