// Package plugin embeds the agent skill that the plugin package distributes,
// so that `shallnot init` installs the same text.
package plugin

import _ "embed"

// Skill is skills/shallnot/SKILL.md.
//
//go:embed skills/shallnot/SKILL.md
var Skill []byte
