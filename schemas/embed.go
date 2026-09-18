// Package schemas embeds the JSON Schemas of the tool's public formats.
package schemas

import _ "embed"

// Report is the JSON Schema of the JSON report.
//
//go:embed report.schema.json
var Report []byte

// Config is the JSON Schema of the shallnot.yaml configuration file.
//
//go:embed config.schema.json
var Config []byte

// Spec is the JSON Schema of the YAML spec format.
//
//go:embed spec.schema.json
var Spec []byte
