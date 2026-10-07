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
	"github.com/RachidChabane/shallnot/internal/domain/proof"
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

func parseProofs(t *testing.T, block string) (*proof.Config, error) {
	t.Helper()
	settings, err := config.Parse(strings.NewReader("version: 1\nproofs:\n"+block), ".")
	return settings.Proofs, err
}

func TestProofsBlock(t *testing.T) {
	t.Run("leaves proofs unset when the file has no proofs block [verifies SN-108~2]", func(t *testing.T) {
		settings, err := config.Parse(strings.NewReader("version: 1\n"), ".")
		if err != nil || settings.Proofs != nil {
			t.Fatalf("got %+v, %v", settings.Proofs, err)
		}
	})

	t.Run("applies the defaults to an empty proofs block [verifies SN-108~2]", func(t *testing.T) {
		settings, err := config.Parse(strings.NewReader("version: 1\nproofs: {}\n"), ".")
		if err != nil || settings.Proofs == nil {
			t.Fatalf("got %v", err)
		}
		if !reflect.DeepEqual(*settings.Proofs, proof.DefaultConfig()) {
			t.Fatalf("got %+v, want %+v", *settings.Proofs, proof.DefaultConfig())
		}
		defaults := proof.DefaultConfig()
		if defaults.Trust != proof.TrustSigned || defaults.Thresholds.Validity.Default != 0.70 || defaults.Escalation != proof.EscalationError ||
			defaults.SignedPassBasis != proof.BasisCorroborated || defaults.MinIsolation != proof.IsolationRestricted {
			t.Fatalf("unexpected defaults %+v", defaults)
		}
	})

	t.Run("reads every key of the block over its default [verifies SN-108~2]", func(t *testing.T) {
		proofs, err := parseProofs(t, `  verified_dir: build/verified
  require: [alignment, 'https://rachid-chabane.com/attestations/discriminating-power/v1', alignment]
  code: ['src/**']
  trust: local
  thresholds: {validity: 0.8, alignment: 0.6}
  escalation: warning
  signed_pass_basis: executed-only
  min_isolation: unrestricted
  max_host_exclusions: 0
`)
		if err != nil {
			t.Fatal(err)
		}
		want := proof.Config{
			VerifiedDir: "build/verified", Require: []proof.Kind{proof.KindAlignment, proof.KindDiscriminatingPower}, Code: []string{"src/**"},
			Trust: proof.TrustLocal, Thresholds: proof.Thresholds{Validity: proof.Threshold{Default: 0.8}, Alignment: proof.Threshold{Default: 0.6}}, Escalation: proof.EscalationWarning,
			SignedPassBasis: proof.BasisExecutedOnly, MinIsolation: proof.IsolationUnrestricted, MaxHostExclusions: 0,
		}
		if !reflect.DeepEqual(*proofs, want) {
			t.Fatalf("got %+v, want %+v", *proofs, want)
		}
	})

	t.Run("reads a threshold as a bare number or as a mapping of default and confidence sources [verifies SN-108~2]", func(t *testing.T) {
		proofs, err := parseProofs(t, `  thresholds:
    validity: 0.8
    alignment:
      default: 0.6
      host-votes: 0.75
      system-one: 0.9
      "host-votes(2)": 0.85
      execution: 0.5
      "system-one:model-a@0123": 0.95
`)
		if err != nil {
			t.Fatal(err)
		}
		want := proof.Thresholds{
			Validity: proof.Threshold{Default: 0.8},
			Alignment: proof.Threshold{Default: 0.6, BySource: map[proof.ConfidenceSource]float64{
				"host-votes": 0.75, "system-one": 0.9, "host-votes(2)": 0.85, "execution": 0.5, "system-one:model-a@0123": 0.95,
			}},
		}
		if !reflect.DeepEqual(proofs.Thresholds, want) {
			t.Fatalf("got %+v, want %+v", proofs.Thresholds, want)
		}
	})

	t.Run("keeps the default of a mapping that names no default and of a threshold left out [verifies SN-108~2]", func(t *testing.T) {
		proofs, err := parseProofs(t, "  thresholds:\n    validity: {host-votes: 0.9}\n")
		if err != nil {
			t.Fatal(err)
		}
		validity := proofs.Thresholds.Validity
		if validity.Default != proof.DefaultValidityThreshold || validity.For("host-votes(3)") != 0.9 || validity.For("system-one:m@d") != proof.DefaultValidityThreshold {
			t.Fatalf("got %+v", validity)
		}
		if !reflect.DeepEqual(proofs.Thresholds.Alignment, proof.DefaultConfig().Thresholds.Alignment) {
			t.Fatalf("alignment = %+v", proofs.Thresholds.Alignment)
		}
	})

	t.Run("refuses a threshold that names no confidence source, is out of range or is neither a number nor a mapping [verifies SN-108~2]", func(t *testing.T) {
		for name, block := range map[string]string{
			"unknown source":       "  thresholds: {validity: {judge: 0.5}}\n",
			"host votes of zero":   "  thresholds: {validity: {host-votes(0): 0.5}}\n",
			"system one no origin": "  thresholds: {validity: {'system-one:': 0.5}}\n",
			"source out of range":  "  thresholds: {alignment: {host-votes: 1.2}}\n",
			"default out of range": "  thresholds: {alignment: {default: -0.2}}\n",
			"text":                 "  thresholds: {alignment: high}\n",
			"list":                 "  thresholds: {alignment: [0.5]}\n",
			"text in a mapping":    "  thresholds: {alignment: {host-votes: high}}\n",
		} {
			if _, err := parseProofs(t, block); err == nil {
				t.Errorf("%s was accepted", name)
			}
		}
	})

	t.Run("refuses an unknown key, a value outside its closed set and a confidence outside [0,1] [verifies SN-108~2]", func(t *testing.T) {
		for name, block := range map[string]string{
			"unknown key":        "  verifier: x\n",
			"unknown nested key": "  thresholds: {judge: 0.5}\n",
			"trust":              "  trust: trusted\n",
			"escalation":         "  escalation: info\n",
			"basis":              "  signed_pass_basis: vibes\n",
			"isolation":          "  min_isolation: moat\n",
			"require":            "  require: [mutation]\n",
			"high threshold":     "  thresholds: {validity: 1.5}\n",
			"low threshold":      "  thresholds: {alignment: -0.1}\n",
			"negative exclusion": "  max_host_exclusions: -1\n",
		} {
			if _, err := parseProofs(t, block); err == nil {
				t.Errorf("%s was accepted", name)
			}
		}
	})

	t.Run("refuses to require completeness proofs, which are advisory [verifies SN-108~2]", func(t *testing.T) {
		_, err := parseProofs(t, "  require: [completeness]\n")
		if err == nil || !strings.Contains(err.Error(), "advisory") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("the config schema describes the block the types accept [verifies SN-108~2]", func(t *testing.T) {
		var schema struct {
			Properties struct {
				Proofs struct {
					Properties map[string]json.RawMessage `json:"properties"`
				} `json:"proofs"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(schemas.Config, &schema); err != nil {
			t.Fatal(err)
		}
		var documented, declared []string
		for key := range schema.Properties.Proofs.Properties {
			documented = append(documented, key)
		}
		fileType := reflect.TypeOf(config.ProofsFile{})
		for index := range fileType.NumField() {
			declared = append(declared, fileType.Field(index).Tag.Get("yaml"))
		}
		sort.Strings(documented)
		sort.Strings(declared)
		if !reflect.DeepEqual(documented, declared) {
			t.Fatalf("schema keys %v, type keys %v", documented, declared)
		}
	})
}
