// Package version identifies the build.
package version

// Name is the tool's name as it appears in reports.
const Name = "shallnot"

// Version is set at build time with -ldflags "-X .../internal/version.Version=v1.2.3".
var Version = "dev"
