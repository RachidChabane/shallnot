// Command shallnot is a spec-to-test traceability gate.
package main

import (
	"os"

	"github.com/RachidChabane/shallnot/internal/cli"
)

func main() {
	os.Exit(int(cli.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)))
}
