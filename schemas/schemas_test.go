package schemas_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"

	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/schemas"
)

func lookup(t *testing.T, schema []byte, path ...string) any {
	t.Helper()
	var node any
	if err := json.Unmarshal(schema, &node); err != nil {
		t.Fatal(err)
	}
	for _, key := range path {
		object, isObject := node.(map[string]any)
		if !isObject {
			t.Fatalf("no %v in the schema", path)
		}
		node = object[key]
	}
	return node
}

func enumAt(t *testing.T, schema []byte, path ...string) []string {
	t.Helper()
	values, isList := lookup(t, schema, append(path, "enum")...).([]any)
	if !isList {
		t.Fatalf("no enum at %v", path)
	}
	names := make([]string, len(values))
	for index, value := range values {
		names[index] = value.(string)
	}
	return names
}

// requirable lists the kinds and predicate types a criterion may be required to hold a proof of.
func requirable() []string {
	var names, uris []string
	for _, kind := range proof.Kinds() {
		if kind == proof.KindCompleteness {
			continue
		}
		names = append(names, string(kind))
		uris = append(uris, kind.PredicateType())
	}
	return append(names, uris...)
}

func without(names []string, excluded string) []string {
	var kept []string
	for _, name := range names {
		if name != excluded {
			kept = append(kept, name)
		}
	}
	return kept
}

func TestSchemasAgreeWithTheTypes(t *testing.T) {
	reported := without(domain.SeverityNames(), domain.SeverityOff.String())
	enums := []struct {
		name   string
		schema []byte
		path   []string
		want   []string
	}{
		{"report categories", schemas.Report, []string{"$defs", "finding", "properties", "category"}, domain.CategoryNames()},
		{"report severities", schemas.Report, []string{"$defs", "finding", "properties", "severity"}, reported},
		{"report fail_on", schemas.Report, []string{"properties", "run", "properties", "fail_on"}, reported},
		{"report coverage", schemas.Report, []string{"$defs", "requirement", "properties", "coverage"}, domain.CoverageNames()},
		{"report status", schemas.Report, []string{"$defs", "requirement", "properties", "status"}, domain.StatusNames()},
		{"report outcomes", schemas.Report, []string{"$defs", "bound_test", "properties", "outcome"}, domain.OutcomeNames()},
		{"report verdicts", schemas.Report, []string{"properties", "verdict"}, domain.VerdictNames()},
		{"report example value types", schemas.Report, []string{"$defs", "literal", "properties", "type"}, without(domain.LiteralTypeNames(), domain.LiteralName.String())},
		{"report outcome value types", schemas.Report, []string{"$defs", "outcome", "properties", "type"}, domain.LiteralTypeNames()},
		{"report outcome kinds", schemas.Report, []string{"$defs", "outcome", "properties", "kind"}, domain.OutcomeKindNames()},
		{"config fail_on", schemas.Config, []string{"properties", "fail_on"}, reported},
		{"config proofs require", schemas.Config, []string{"properties", "proofs", "properties", "require", "items"}, requirable()},
		{"config proofs trust", schemas.Config, []string{"properties", "proofs", "properties", "trust"}, proof.TrustNames()},
		{"config proofs escalation", schemas.Config, []string{"properties", "proofs", "properties", "escalation"}, proof.EscalationNames()},
		{"config proofs signed_pass_basis", schemas.Config, []string{"properties", "proofs", "properties", "signed_pass_basis"}, proof.SignedPassBasisNames()},
		{"config proofs min_isolation", schemas.Config, []string{"properties", "proofs", "properties", "min_isolation"}, proof.IsolationNames()},
		{"spec status", schemas.Spec, []string{"properties", "requirements", "items", "properties", "status"}, domain.StatusNames()},
	}
	for _, enum := range enums {
		t.Run("the schema lists the "+enum.name+" the types define [verifies SN-60~1]", func(t *testing.T) {
			if got := enumAt(t, enum.schema, enum.path...); !reflect.DeepEqual(got, enum.want) {
				t.Fatalf("schema %v, types %v", got, enum.want)
			}
		})
	}

	t.Run("the config schema accepts a severity for every category [verifies SN-60~1]", func(t *testing.T) {
		properties := lookup(t, schemas.Config, "properties", "severities", "properties").(map[string]any)
		var got []string
		for name := range properties {
			got = append(got, name)
		}
		want := domain.CategoryNames()
		sort.Strings(got)
		sort.Strings(want)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("schema %v, types %v", got, want)
		}
	})
}

func validateYAML(t *testing.T, schema []byte, path string) error {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var instance any
	if err := yaml.Unmarshal(text, &instance); err != nil {
		t.Fatal(err)
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", document); err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.Compile("schema.json")
	if err != nil {
		t.Fatal(err)
	}
	return compiled.Validate(instance)
}

func TestSchemasDescribeTheInputs(t *testing.T) {
	t.Run("the config schema accepts a config file the tool accepts [verifies SN-53~1]", func(t *testing.T) {
		for _, path := range []string{"../fixtures/pipeline/shallnot.yaml", "../shallnot.yaml"} {
			if err := validateYAML(t, schemas.Config, path); err != nil {
				t.Errorf("%s: %v", path, err)
			}
		}
	})

	t.Run("the spec schema accepts a YAML spec the tool accepts, and refuses one it reports [verifies SN-5~1]", func(t *testing.T) {
		if err := validateYAML(t, schemas.Spec, "../fixtures/cart/spec/cart.yaml"); err != nil {
			t.Error(err)
		}
		if err := validateYAML(t, schemas.Spec, "../internal/adapters/spec/testdata/spec.yaml"); err == nil {
			t.Error("a spec with unreadable requirements was accepted")
		}
	})

	t.Run("the spec schema accepts out-of-scope declarations and refuses unreadable ones [verifies SN-96~1]", func(t *testing.T) {
		accepted := "requirements:\n  - id: CART-1\n    statement: THE SYSTEM SHALL total.\nout_of_scope:\n  - dimension: concurrency\n    reason: one writer\n"
		refused := map[string]string{
			"no reason":    "requirements: []\nout_of_scope:\n  - dimension: concurrency\n",
			"extra key":    "requirements: []\nout_of_scope:\n  - dimension: concurrency\n    reason: x\n    more: y\n",
			"not a list":   "requirements: []\nout_of_scope: concurrency\n",
			"empty reason": "requirements: []\nout_of_scope:\n  - dimension: concurrency\n    reason: ''\n",
		}
		write := func(text string) string {
			path := filepath.Join(t.TempDir(), "spec.yaml")
			if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
				t.Fatal(err)
			}
			return path
		}
		for name, text := range map[string]string{
			"requirements and out_of_scope": accepted,
			"out_of_scope only":             "out_of_scope:\n  - dimension: concurrency\n    reason: one writer\n",
		} {
			if err := validateYAML(t, schemas.Spec, write(text)); err != nil {
				t.Errorf("%s: %v", name, err)
			}
		}
		for name, text := range refused {
			if err := validateYAML(t, schemas.Spec, write(text)); err == nil {
				t.Errorf("%s was accepted", name)
			}
		}
	})
}
