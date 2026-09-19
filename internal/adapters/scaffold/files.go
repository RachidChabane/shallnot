package scaffold

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/RachidChabane/shallnot/internal/adapters/config"
	"github.com/RachidChabane/shallnot/plugin"
)

const (
	specsDirectory = "specs"
	agentsPath     = "AGENTS.md"
	claudePath     = "CLAUDE.md"
	skillPath      = ".claude/skills/shallnot/SKILL.md"
	conftestPath   = "conftest.py"

	sectionBegin = "<!-- shallnot:begin -->"
	sectionEnd   = "<!-- shallnot:end -->"
	agentsImport = "@" + agentsPath
)

// configFile writes a starter shallnot.yaml and never rewrites an existing one: the project owns it.
func configFile(directory string, detected []Runner) provisioner {
	return provisioner{path: config.DefaultFileName, desired: func(current []byte) ([]byte, error) {
		if current != nil {
			return current, nil
		}
		var text strings.Builder
		fmt.Fprintf(&text, "version: %d\nspecs:\n  - %s\n", config.SupportedVersion, specsDirectory)
		writeList(&text, "tests", testRoots(directory, detected))
		writeList(&text, "results", collect(detected, func(runner Runner) string { return runner.Results }))
		writeList(&text, "test_commands", collect(detected, func(runner Runner) string { return runner.TestCommand }))
		return []byte(text.String()), nil
	}}
}

func writeList(text *strings.Builder, key string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(text, "%s:\n", key)
	for _, value := range values {
		fmt.Fprintf(text, "  - %s\n", yamlScalar(value))
	}
}

var plainYAMLScalar = regexp.MustCompile(`^[A-Za-z0-9_./-][A-Za-z0-9_./ =@:-]*$`)

// yamlScalar quotes a value unless it is safe as a plain YAML scalar.
func yamlScalar(value string) string {
	if plainYAMLScalar.MatchString(value) && !strings.Contains(value, ": ") {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func collect(detected []Runner, field func(Runner) string) []string {
	var values []string
	for _, runner := range detected {
		values = append(values, field(runner))
	}
	return values
}

// testRoots returns the existing candidate roots of the detected runners, without duplicates.
func testRoots(directory string, detected []Runner) []string {
	seen := map[string]bool{}
	var roots []string
	for _, runner := range detected {
		for _, root := range runner.TestRoots {
			if !seen[root] && exists(directory, root) {
				seen[root] = true
				roots = append(roots, root)
			}
		}
	}
	return roots
}

// AgentsSection renders the skill as a section of an AGENTS.md: no front
// matter, headings one level down, between markers that let init update it.
func AgentsSection() string {
	body := string(plugin.Skill)
	if strings.HasPrefix(body, "---\n") {
		if end := strings.Index(body[4:], "\n---\n"); end >= 0 {
			body = body[4+end+len("\n---\n"):]
		}
	}
	lines := strings.Split(strings.TrimSpace(body), "\n")
	inFence := false
	for index, line := range lines {
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
		}
		if !inFence && strings.HasPrefix(line, "#") {
			lines[index] = "#" + line
		}
	}
	return sectionBegin + "\n" + strings.Join(lines, "\n") + "\n" + sectionEnd + "\n"
}

// agentsFile keeps the shallnot section of AGENTS.md current and leaves the rest of the file alone.
func agentsFile() provisioner {
	return provisioner{path: agentsPath, desired: func(current []byte) ([]byte, error) {
		section := AgentsSection()
		text := string(current)
		begin, end := strings.Index(text, sectionBegin), strings.Index(text, sectionEnd)
		switch {
		case begin >= 0 && end > begin:
			after := strings.TrimPrefix(text[end+len(sectionEnd):], "\n")
			return []byte(text[:begin] + section + after), nil
		case begin >= 0 || end >= 0:
			return nil, fmt.Errorf("the %s and %s markers are unbalanced", sectionBegin, sectionEnd)
		case strings.TrimSpace(text) == "":
			return []byte(section), nil
		default:
			return []byte(strings.TrimRight(text, "\n") + "\n\n" + section), nil
		}
	}}
}

// claudeMemory makes Claude Code read AGENTS.md through an import in CLAUDE.md.
func claudeMemory() provisioner {
	return provisioner{path: claudePath, desired: func(current []byte) ([]byte, error) {
		for _, line := range strings.Split(string(current), "\n") {
			if strings.TrimSpace(line) == agentsImport {
				return current, nil
			}
		}
		if len(bytes.TrimSpace(current)) == 0 {
			return []byte(agentsImport + "\n"), nil
		}
		return []byte(strings.TrimRight(string(current), "\n") + "\n\n" + agentsImport + "\n"), nil
	}}
}

// claudeSkill installs the skill where Claude Code discovers project skills.
func claudeSkill() provisioner {
	return provisioner{path: skillPath, desired: func([]byte) ([]byte, error) { return plugin.Skill, nil }}
}

const conftestMarker = `item.iter_markers(name="verifies")`

const conftestHook = `
def pytest_configure(config):
    config.addinivalue_line("markers", "verifies(*refs): requirements the test verifies, as ID~REVISION")


def pytest_collection_modifyitems(items):
    for item in items:
        for marker in item.iter_markers(name="verifies"):
            item.user_properties.append(("verifies", ", ".join(marker.args)))
`

// pytestConftest adds the hook that copies `verifies` markers into pytest's JUnit XML.
func pytestConftest() provisioner {
	return provisioner{path: conftestPath, desired: func(current []byte) ([]byte, error) {
		text := string(current)
		switch {
		case strings.Contains(text, conftestMarker):
			return current, nil
		case strings.Contains(text, "def pytest_collection_modifyitems") || strings.Contains(text, "def pytest_configure"):
			return nil, fmt.Errorf("it already defines pytest_configure or pytest_collection_modifyitems: merge the hook of docs/binding.md by hand")
		case strings.TrimSpace(text) == "":
			return []byte(strings.TrimLeft(conftestHook, "\n")), nil
		default:
			return []byte(strings.TrimRight(text, "\n") + "\n\n" + conftestHook), nil
		}
	}}
}
