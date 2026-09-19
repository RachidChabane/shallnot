package scaffold

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Change says what init does to one file.
type Change uint8

const (
	ChangeNone Change = iota
	ChangeCreate
	ChangeUpdate
)

var changeNames = []string{"unchanged", "create", "update"}

func (c Change) String() string { return changeNames[c] }

// Action is one file init wants in a given state.
type Action struct {
	Path    string
	Change  Change
	Content []byte
}

// Plan is everything init wants for a project.
type Plan struct {
	Directory string
	Actions   []Action
	NextSteps []string
}

// Pending reports whether applying the plan would change a file.
func (p Plan) Pending() bool {
	for _, action := range p.Actions {
		if action.Change != ChangeNone {
			return true
		}
	}
	return false
}

// Apply writes the files that differ from the plan.
func (p Plan) Apply() error {
	for _, action := range p.Actions {
		if action.Change == ChangeNone {
			continue
		}
		path := filepath.Join(p.Directory, action.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, action.Content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Options select what init installs.
type Options struct {
	Directory string
	// Hooks are the harnesses whose end-of-turn hook is installed.
	Hooks []string
}

// provisioner is the strategy for one file: from its current content (nil when
// absent), it returns the content init wants.
type provisioner struct {
	path    string
	desired func(current []byte) ([]byte, error)
}

// Build computes the plan for a project without touching it.
func Build(options Options) (Plan, error) {
	detected := DetectRunners(options.Directory)
	plan := Plan{Directory: options.Directory}
	provisioners := append([]provisioner{configFile(options.Directory, detected), agentsFile(), claudeMemory()}, claudeSkills()...)
	for _, runner := range detected {
		if runner.Name == "pytest" {
			provisioners = append(provisioners, pytestConftest())
		}
		if runner.NextStep != "" {
			plan.NextSteps = append(plan.NextSteps, runner.NextStep)
		}
	}
	for _, harness := range options.Hooks {
		hook, known := hookProvisioners[harness]
		if !known {
			return Plan{}, fmt.Errorf("unknown hook harness %q (expected %v)", harness, HookHarnesses())
		}
		provisioners = append(provisioners, hook)
	}
	if len(detected) == 0 {
		plan.NextSteps = append(plan.NextSteps, "No known test runner was found: set `tests`, `results` and `test_commands` in shallnot.yaml; see docs/configuration.md.")
	}
	if !exists(options.Directory, specsDirectory) {
		plan.NextSteps = append(plan.NextSteps, "Write the requirements in "+specsDirectory+"/ (Markdown or YAML); see docs/spec-format.md. The gate gives no verdict without a spec.")
	}

	for _, provisioner := range provisioners {
		action, err := provisioner.action(options.Directory)
		if err != nil {
			return Plan{}, err
		}
		plan.Actions = append(plan.Actions, action)
	}
	return plan, nil
}

func (p provisioner) action(directory string) (Action, error) {
	current, err := os.ReadFile(filepath.Join(directory, p.path))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Action{}, err
	}
	desired, err := p.desired(current)
	if err != nil {
		return Action{}, fmt.Errorf("%s: %w", p.path, err)
	}
	action := Action{Path: p.path, Content: desired}
	switch {
	case current == nil:
		action.Change = ChangeCreate
	case !bytes.Equal(current, desired):
		action.Change = ChangeUpdate
	}
	return action, nil
}
