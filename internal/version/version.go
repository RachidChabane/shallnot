// Package version identifies the build.
package version

import "runtime/debug"

// Name is the tool's name as it appears in reports.
const Name = "shallnot"

// unreleased is the version of a build that carries no version information.
const unreleased = "dev"

// injected is set at build time with -ldflags "-X .../internal/version.injected=v1.2.3".
var injected = ""

// Version is the release the binary was built from: the injected one, else the
// module version recorded by `go install module@version`, else "dev".
var Version = resolve()

func resolve() string {
	if injected != "" {
		return injected
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return unreleased
}
