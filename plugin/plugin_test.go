package plugin_test

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/plugin"
)

// Agent Plugins 1.0.0 closes the manifest: these are its only top-level fields.
var agentPluginsFields = map[string]bool{
	"$schema": true, "name": true, "version": true, "description": true, "author": true,
	"homepage": true, "repository": true, "license": true, "keywords": true, "extensions": true,
}

var agentPluginsName = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$`)

func manifest(t *testing.T, path string) map[string]any {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(content, &fields); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return fields
}

func TestPackage(t *testing.T) {
	t.Run("the Agent Plugins manifest has a valid name and only the fields the specification defines [verifies SN-78~2]", func(t *testing.T) {
		fields := manifest(t, "plugin.json")
		name, _ := fields["name"].(string)
		if !agentPluginsName.MatchString(name) || strings.Contains(name, "--") || strings.Contains(name, "..") || len(name) > 64 {
			t.Errorf("invalid name %q", name)
		}
		for field := range fields {
			if !agentPluginsFields[field] {
				t.Errorf("field %q is not part of the manifest", field)
			}
		}
	})

	t.Run("both manifests describe the same plugin version [verifies SN-78~2]", func(t *testing.T) {
		portable, claude := manifest(t, "plugin.json"), manifest(t, ".claude-plugin/plugin.json")
		for _, field := range []string{"name", "version", "license", "repository"} {
			if portable[field] != claude[field] {
				t.Errorf("%s: %v in plugin.json, %v in .claude-plugin/plugin.json", field, portable[field], claude[field])
			}
		}
	})

	t.Run("each skill carries the name and description a client needs to offer it unprompted [verifies SN-78~2]", func(t *testing.T) {
		for _, skill := range plugin.Skills() {
			text := string(skill.Content)
			if !strings.HasPrefix(text, "---\nname: "+skill.Name+"\ndescription: ") {
				t.Fatalf("%s front matter:\n%.200s", skill.Name, text)
			}
			description := strings.SplitN(strings.SplitN(text, "description: ", 2)[1], "\n", 2)[0]
			if len(description) > 1024 || !strings.Contains(description, "shallnot.yaml") {
				t.Errorf("%s: description of %d characters: %s", skill.Name, len(description), description)
			}
		}
	})

	t.Run("the Claude Code hooks call the binary and nothing else [verifies SN-78~2]", func(t *testing.T) {
		stop, err := os.ReadFile("hooks/stop.sh")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(stop), "exec shallnot hook claude-stop") {
			t.Errorf("stop.sh:\n%s", stop)
		}
		hooks := manifest(t, "hooks/hooks.json")["hooks"].(map[string]any)
		if _, ok := hooks["Stop"]; !ok {
			t.Error("no Stop hook")
		}
	})
}
