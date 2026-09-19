// Package plugin embeds the agent skills that the plugin package distributes,
// so that `shallnot init` installs the same text.
package plugin

import (
	"embed"
	"path"
)

//go:embed skills
var skillFiles embed.FS

const (
	skillsDirectory = "skills"
	skillFileName   = "SKILL.md"
)

// skillOrder is the reading order of the skills: the convention first, then
// what comes before it (requirements) and after it (review).
var skillOrder = []string{"shallnot", "shallnot-plan", "shallnot-review"}

// Skill is one packaged skill.
type Skill struct {
	Name    string
	Content []byte
}

// Skills returns the packaged skills in reading order.
func Skills() []Skill {
	skills := make([]Skill, 0, len(skillOrder))
	for _, name := range skillOrder {
		content, err := skillFiles.ReadFile(path.Join(skillsDirectory, name, skillFileName))
		if err != nil {
			// The files are embedded at build time: a missing one is a packaging defect.
			panic(err)
		}
		skills = append(skills, Skill{Name: name, Content: content})
	}
	return skills
}
