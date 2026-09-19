// Package config reads the optional shallnot.yaml configuration file.
package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v3"

	"github.com/RachidChabane/shallnot/internal/app"
	"github.com/RachidChabane/shallnot/internal/domain"
)

// DefaultFileName is the configuration file looked up in the working directory.
const DefaultFileName = "shallnot.yaml"

// SupportedVersion is the configuration format version.
const SupportedVersion = 1

// File is the configuration file. Its shape is specified by schemas/config.schema.json.
type File struct {
	Version         int               `yaml:"version"`
	IDPattern       string            `yaml:"id_pattern"`
	Specs           []string          `yaml:"specs"`
	Focus           []string          `yaml:"focus"`
	FocusIDs        []string          `yaml:"focus_ids"`
	Tests           []string          `yaml:"tests"`
	Results         []string          `yaml:"results"`
	Exclude         []string          `yaml:"exclude"`
	TestCommands    []string          `yaml:"test_commands"`
	DefaultExcludes *bool             `yaml:"default_excludes"`
	Advisory        bool              `yaml:"advisory"`
	Strict          bool              `yaml:"strict"`
	FailOn          string            `yaml:"fail_on"`
	Severities      map[string]string `yaml:"severities"`
}

// Load reads a configuration file into settings. Relative paths in the file
// are resolved against the file's directory.
func Load(path string) (app.Settings, error) {
	file, err := os.Open(path)
	if err != nil {
		return app.Settings{}, err
	}
	defer file.Close()
	settings, err := Parse(file, filepath.Dir(path))
	if err != nil {
		return app.Settings{}, fmt.Errorf("%s: %w", path, err)
	}
	return settings, nil
}

// Parse decodes a configuration document, rejecting unknown keys.
func Parse(reader io.Reader, baseDirectory string) (app.Settings, error) {
	var file File
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	if err := decoder.Decode(&file); err != nil && !errors.Is(err, io.EOF) {
		return app.Settings{}, err
	}
	return file.settings(baseDirectory)
}

func (f File) settings(baseDirectory string) (app.Settings, error) {
	settings := app.DefaultSettings()
	if f.Version != SupportedVersion {
		return settings, fmt.Errorf("`version` must be %d, got %d", SupportedVersion, f.Version)
	}
	if f.IDPattern != "" {
		settings.IDPattern = f.IDPattern
	}
	settings.Specs = rebase(baseDirectory, f.Specs)
	settings.Focus = rebase(baseDirectory, f.Focus)
	settings.Tests = rebase(baseDirectory, f.Tests)
	settings.Results = rebase(baseDirectory, f.Results)
	settings.FocusIDs = f.FocusIDs
	settings.Exclude = f.Exclude
	settings.TestCommands = f.TestCommands
	if baseDirectory != "." {
		settings.WorkDir = baseDirectory
	}
	if f.DefaultExcludes != nil {
		settings.UseDefaultExcludes = *f.DefaultExcludes
	}
	settings.Policy.Advisory = f.Advisory
	if f.FailOn != "" {
		if err := SetFailOn(&settings.Policy, f.FailOn); err != nil {
			return settings, err
		}
	}
	if f.Strict {
		ApplyStrict(&settings.Policy)
	}
	for _, category := range sortedKeys(f.Severities) {
		if err := SetSeverity(&settings.Policy, category, f.Severities[category]); err != nil {
			return settings, err
		}
	}
	return settings, nil
}

// SetFailOn sets the lowest blocking severity from its wire name.
func SetFailOn(policy *domain.Policy, text string) error {
	severity, err := domain.ParseSeverity(text)
	if err != nil {
		return fmt.Errorf("fail_on: %w", err)
	}
	if severity == domain.SeverityOff {
		return errors.New("fail_on: \"off\" is not a blocking severity; use advisory mode to never block")
	}
	policy.FailOn = severity
	return nil
}

// SetSeverity overrides the severity of one category from wire names.
func SetSeverity(policy *domain.Policy, categoryText, severityText string) error {
	category, err := domain.ParseCategory(categoryText)
	if err != nil {
		return fmt.Errorf("severities: %w", err)
	}
	severity, err := domain.ParseSeverity(severityText)
	if err != nil {
		return fmt.Errorf("severities.%s: %w", categoryText, err)
	}
	policy.Severities[category] = severity
	return nil
}

// ApplyStrict turns untagged tests into errors.
func ApplyStrict(policy *domain.Policy) {
	policy.Severities[domain.CategoryUntaggedTest] = domain.SeverityError
}

func rebase(baseDirectory string, paths []string) []string {
	rebased := make([]string, len(paths))
	for index, path := range paths {
		if filepath.IsAbs(path) || baseDirectory == "." || baseDirectory == "" {
			rebased[index] = path
			continue
		}
		rebased[index] = filepath.Join(baseDirectory, path)
	}
	return rebased
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
