package scaffold

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/RachidChabane/shallnot/internal/adapters/agenthook"
)

const (
	claudeSettingsPath = ".claude/settings.json"
	cursorHooksPath    = ".cursor/hooks.json"
	cursorHooksVersion = 1
	// gateTimeoutSeconds bounds a hook's test run.
	gateTimeoutSeconds = 600
)

var hookProvisioners = map[string]provisioner{
	"claude": {path: claudeSettingsPath, desired: claudeStopHook},
	"cursor": {path: cursorHooksPath, desired: cursorStopHook},
}

// HookHarnesses lists the harnesses init can install a hook for.
func HookHarnesses() []string {
	names := make([]string, 0, len(hookProvisioners))
	for name := range hookProvisioners {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func hookCommand(harness agenthook.Harness) string { return "shallnot hook " + harness.Name() }

// claudeStopHook adds the Stop hook to .claude/settings.json, keeping every other setting.
func claudeStopHook(current []byte) ([]byte, error) {
	command := hookCommand(agenthook.ClaudeStop{})
	settings, err := decodeObject(current)
	if err != nil {
		return nil, err
	}
	if mentions(settings["hooks"], command) {
		return current, nil
	}
	hooks := object(settings, "hooks")
	hooks["Stop"] = append(list(hooks["Stop"]), map[string]any{
		"hooks": []any{map[string]any{"type": "command", "command": command, "timeout": gateTimeoutSeconds}},
	})
	return encodeObject(settings)
}

// cursorStopHook adds the stop hook to .cursor/hooks.json, keeping every other hook.
func cursorStopHook(current []byte) ([]byte, error) {
	command := hookCommand(agenthook.CursorStop{})
	settings, err := decodeObject(current)
	if err != nil {
		return nil, err
	}
	if mentions(settings["hooks"], command) {
		return current, nil
	}
	if _, versioned := settings["version"]; !versioned {
		settings["version"] = cursorHooksVersion
	}
	hooks := object(settings, "hooks")
	hooks["stop"] = append(list(hooks["stop"]), map[string]any{"command": command})
	return encodeObject(settings)
}

func decodeObject(content []byte) (map[string]any, error) {
	settings := map[string]any{}
	if len(bytes.TrimSpace(content)) == 0 {
		return settings, nil
	}
	if err := json.Unmarshal(content, &settings); err != nil {
		return nil, fmt.Errorf("not a JSON object: %w", err)
	}
	return settings, nil
}

func encodeObject(settings map[string]any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(settings); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func object(parent map[string]any, key string) map[string]any {
	child, isObject := parent[key].(map[string]any)
	if !isObject {
		child = map[string]any{}
		parent[key] = child
	}
	return child
}

func list(value any) []any {
	items, _ := value.([]any)
	return items
}

// mentions reports whether a JSON value contains the command string anywhere.
func mentions(value any, command string) bool {
	switch typed := value.(type) {
	case string:
		return typed == command
	case []any:
		for _, item := range typed {
			if mentions(item, command) {
				return true
			}
		}
	case map[string]any:
		for _, item := range typed {
			if mentions(item, command) {
				return true
			}
		}
	}
	return false
}
