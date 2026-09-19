package scaffold

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/RachidChabane/shallnot/internal/adapters/agenthook"
)

const (
	// gateTimeoutSeconds bounds a hook's test run.
	gateTimeoutSeconds = 600
	// gateTimeoutMilliseconds is the same bound for harnesses that count in milliseconds.
	gateTimeoutMilliseconds = gateTimeoutSeconds * 1000

	stopCommand        = "shallnot hook " + agenthook.StopName
	claudeStopCommand  = "shallnot hook " + agenthook.ClaudeStopName
	copilotStopCommand = "shallnot hook " + agenthook.CopilotStopName

	hookTypeCommand = "command"
	hookEntryName   = "shallnot"
)

// hookHarness is what init knows about one agent harness: how to tell that a
// project uses it, the files that install its end-of-turn hook, and what the
// files cannot do for the user.
type hookHarness struct {
	name string
	// always selects the harness even when the project shows no sign of it.
	always bool
	// markers are paths whose presence says the project uses the harness.
	markers      []string
	provisioners []provisioner
	nextStep     string
}

var hookHarnesses = []hookHarness{
	{name: "claude", always: true, provisioners: []provisioner{
		{path: ".claude/settings.json", desired: stopHookUnder("hooks", "Stop", claudeStopCommand)},
	}},
	{name: "codex", markers: []string{".codex"}, provisioners: []provisioner{
		{path: ".codex/hooks.json", desired: stopHookUnder("hooks", "Stop", stopCommand)},
	}, nextStep: "Codex runs project hooks once `hooks = true` is set under `[features]` in its config.toml and the project is trusted."},
	{name: "copilot", markers: []string{".github/copilot-instructions.md", ".github/hooks", ".github/skills", ".github/instructions"}, provisioners: []provisioner{
		{path: ".github/hooks/shallnot.json", desired: copilotStopHook},
	}},
	{name: "cursor", markers: []string{".cursor"}, provisioners: []provisioner{
		{path: ".cursor/hooks.json", desired: cursorStopHook},
	}},
	{name: "factory", markers: []string{".factory"}, provisioners: []provisioner{
		{path: ".factory/hooks.json", desired: stopHookUnder("", "Stop", stopCommand)},
	}},
	{name: "gemini", markers: []string{".gemini", "GEMINI.md"}, provisioners: []provisioner{
		{path: ".gemini/settings.json", desired: geminiSettings},
	}},
	{name: "goose", markers: []string{".goosehints", ".goose", ".agents/plugins"}, provisioners: []provisioner{
		{path: goosePluginRoot + "/plugin.json", desired: fixed(goosePluginManifest)},
		{path: goosePluginRoot + "/hooks/hooks.json", desired: stopHookUnder("hooks", "Stop", stopCommand)},
	}},
	{name: "opencode", markers: []string{".opencode", "opencode.json", "opencode.jsonc"}, provisioners: []provisioner{
		{path: ".opencode/plugins/shallnot.js", desired: fixed(opencodePlugin)},
	}},
	{name: "qwen", markers: []string{".qwen", "QWEN.md"}, provisioners: []provisioner{
		{path: ".qwen/settings.json", desired: stopHookUnder("hooks", "Stop", stopCommand)},
	}},
}

// HookHarnesses lists the harnesses init can install a hook for.
func HookHarnesses() []string {
	names := make([]string, 0, len(hookHarnesses))
	for _, harness := range hookHarnesses {
		names = append(names, harness.name)
	}
	sort.Strings(names)
	return names
}

// DetectHarnesses lists the harnesses a project shows signs of using, and those installed regardless.
func DetectHarnesses(directory string) []string {
	var names []string
	for _, harness := range hookHarnesses {
		if harness.always || anyExists(directory, harness.markers) {
			names = append(names, harness.name)
		}
	}
	return names
}

func anyExists(directory string, paths []string) bool {
	for _, path := range paths {
		if exists(directory, path) {
			return true
		}
	}
	return false
}

func lookupHarness(name string) (hookHarness, error) {
	for _, harness := range hookHarnesses {
		if harness.name == name {
			return harness, nil
		}
	}
	return hookHarness{}, fmt.Errorf("unknown hook harness %q (expected %v)", name, HookHarnesses())
}

// fixed is the content of a file init owns entirely.
func fixed(content string) func([]byte) ([]byte, error) {
	return func([]byte) ([]byte, error) { return []byte(content), nil }
}

// stopHookUnder adds a command hook to the event's list, in the layout Claude
// Code defined and other harnesses share, keeping every other setting. The
// events sit under the wrapper key, or at the top level when it is empty.
func stopHookUnder(wrapper, event, command string) func([]byte) ([]byte, error) {
	return func(current []byte) ([]byte, error) {
		settings, err := decodeObject(current)
		if err != nil {
			return nil, err
		}
		if addStopHook(settings, wrapper, event, command, gateTimeoutSeconds, nil) {
			return encodeObject(settings)
		}
		return current, nil
	}
}

// addStopHook reports whether it changed the settings. extra holds the keys a
// harness wants beside type, command and timeout.
func addStopHook(settings map[string]any, wrapper, event, command string, timeout int, extra map[string]any) bool {
	events := settings
	if wrapper != "" {
		if mentions(settings[wrapper], command) {
			return false
		}
		events = object(settings, wrapper)
	} else if mentions(settings[event], command) {
		return false
	}
	hook := map[string]any{"type": hookTypeCommand, "command": command, "timeout": timeout}
	for key, value := range extra {
		hook[key] = value
	}
	events[event] = append(list(events[event]), map[string]any{"hooks": []any{hook}})
	return true
}

const (
	geminiStopEvent      = "AfterAgent"
	geminiContextKey     = "context"
	geminiFileNameKey    = "fileName"
	geminiOwnContextFile = "GEMINI.md"
)

// geminiSettings adds the AfterAgent hook and makes Gemini CLI read AGENTS.md,
// which it does only when the context file names say so.
func geminiSettings(current []byte) ([]byte, error) {
	settings, err := decodeObject(current)
	if err != nil {
		return nil, err
	}
	hooked := addStopHook(settings, "hooks", geminiStopEvent, stopCommand, gateTimeoutMilliseconds, map[string]any{"name": hookEntryName})
	named := addContextFile(object(settings, geminiContextKey))
	if !hooked && !named {
		return current, nil
	}
	return encodeObject(settings)
}

// addContextFile reports whether it changed the context settings.
func addContextFile(context map[string]any) bool {
	var names []any
	switch typed := context[geminiFileNameKey].(type) {
	case nil:
		names = []any{geminiOwnContextFile}
	case string:
		names = []any{typed}
	case []any:
		names = typed
	}
	for _, name := range names {
		if name == agentsPath {
			return false
		}
	}
	context[geminiFileNameKey] = append([]any{agentsPath}, names...)
	return true
}

const copilotHooksVersion = 1

// copilotStopHook writes the agentStop hook in a hooks file of its own.
func copilotStopHook(current []byte) ([]byte, error) {
	settings, err := decodeObject(current)
	if err != nil {
		return nil, err
	}
	if mentions(settings["hooks"], copilotStopCommand) {
		return current, nil
	}
	if _, versioned := settings["version"]; !versioned {
		settings["version"] = copilotHooksVersion
	}
	hooks := object(settings, "hooks")
	hooks["agentStop"] = append(list(hooks["agentStop"]), map[string]any{
		"type": hookTypeCommand, "bash": copilotStopCommand, "powershell": copilotStopCommand, "timeoutSec": gateTimeoutSeconds,
	})
	return encodeObject(settings)
}

const cursorHooksVersion = 1

// cursorStopHook adds the stop hook to .cursor/hooks.json, keeping every other hook.
func cursorStopHook(current []byte) ([]byte, error) {
	command := "shallnot hook " + agenthook.CursorStop{}.Name()
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

const goosePluginRoot = ".agents/plugins/shallnot"

const goosePluginManifest = `{
  "name": "shallnot",
  "version": "1.0.0",
  "description": "Runs the shallnot gate at the end of each turn and sends a blocked verdict back to the agent."
}
`

// opencodePlugin runs the hook when a session goes idle. OpenCode has no
// blocking end-of-turn hook: the plugin prompts the session again instead.
const opencodePlugin = `const BLOCKED = 2

export const Shallnot = async ({ client, directory, $ }) => ({
  event: async ({ event }) => {
    if (event.type !== "session.idle") return
    const sessionID = event.properties.sessionID
    const session = await client.session.get({ path: { id: sessionID } })
    if (session.data?.parentID) return
    const input = JSON.stringify({ session_id: sessionID, cwd: directory })
    const hook = await $` + "`shallnot hook stop < ${new Response(input)}`" + `.cwd(directory).nothrow().quiet()
    if (hook.exitCode !== BLOCKED) return
    await client.session.prompt({
      path: { id: sessionID },
      body: { parts: [{ type: "text", text: hook.stderr.toString() }] },
    })
  },
})
`

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
