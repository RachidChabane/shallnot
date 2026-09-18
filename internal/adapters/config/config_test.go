package config_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/adapters/config"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/schemas"
)

func TestConfigFile(t *testing.T) {
	t.Run("the config type has exactly the keys its schema documents [verifies SN-53~1]", func(t *testing.T) {
		var schema struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(schemas.Config, &schema); err != nil {
			t.Fatal(err)
		}
		var documented, declared []string
		for key := range schema.Properties {
			documented = append(documented, key)
		}
		fileType := reflect.TypeOf(config.File{})
		for index := range fileType.NumField() {
			declared = append(declared, fileType.Field(index).Tag.Get("yaml"))
		}
		sort.Strings(documented)
		sort.Strings(declared)
		if !reflect.DeepEqual(documented, declared) {
			t.Fatalf("schema keys %v, type keys %v", documented, declared)
		}
	})

	t.Run("resolves relative paths against the config file's directory and keeps absolute ones [verifies SN-53~1]", func(t *testing.T) {
		absolute := filepath.ToSlash(t.TempDir())
		text := "version: 1\nspecs: [plans, '" + absolute + "']\nresults: ['out/*.xml']\nfocus_ids: ['ABC-1.*']\n"
		settings, err := config.Parse(strings.NewReader(text), "repo")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(settings.Specs, []string{filepath.Join("repo", "plans"), absolute}) || settings.Results[0] != filepath.Join("repo", "out", "*.xml") || settings.FocusIDs[0] != "ABC-1.*" {
			t.Fatalf("got %+v", settings)
		}
	})

	t.Run("an explicit severity wins over strict mode [verifies SN-50~1]", func(t *testing.T) {
		text := "version: 1\nstrict: true\nfail_on: warning\nadvisory: true\ndefault_excludes: false\nseverities: {untagged_test: warning, orphan_tag: 'off'}\n"
		settings, err := config.Parse(strings.NewReader(text), ".")
		if err != nil {
			t.Fatal(err)
		}
		policy := settings.Policy
		if policy.SeverityOf(domain.CategoryUntaggedTest) != domain.SeverityWarning || policy.SeverityOf(domain.CategoryOrphanTag) != domain.SeverityOff ||
			policy.FailOn != domain.SeverityWarning || !policy.Advisory || settings.UseDefaultExcludes {
			t.Fatalf("got %+v", settings)
		}
	})
}
