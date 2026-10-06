package cli_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/RachidChabane/shallnot/internal/adapters/report"
	"github.com/RachidChabane/shallnot/internal/domain"
)

const (
	proofGateFixtures = "internal/cli/testdata/proofgate"
	cartCheckResults  = cartFixtures + "/pytest/results/junit.xml"
)

// cartPlanDigest is the statement digest of CART-2~2 in the specification of the canonical form.
const cartPlanDigest = "1640b763e3b155b4799bedb9a66384bfe5dc9d52030bf3c68d14aaa1b2c89478"

func proofGateReport(t *testing.T, specs ...string) (report.Document, string) {
	t.Helper()
	args := []string{"check", "--no-config", "--results", cartCheckResults, "--format", "json", "--advisory"}
	for _, spec := range specs {
		args = append(args, "--specs", spec)
	}
	result := shallnot(t, args...)
	if result.stderr != "" {
		t.Fatalf("stderr: %s", result.stderr)
	}
	return decode(t, result.stdout), result.stdout
}

func requirementOf(t *testing.T, document report.Document, id string) report.RequirementInfo {
	t.Helper()
	for _, requirement := range document.Requirements {
		if string(requirement.ID) == id {
			return requirement
		}
	}
	t.Fatalf("no requirement %s", id)
	return report.RequirementInfo{}
}

func TestProofGateReport(t *testing.T) {
	inRepository(t)

	t.Run("the report is schema version 1.2 and names the canonicalization [verifies SN-89~1]", func(t *testing.T) {
		document, _ := proofGateReport(t, cartFixtures+"/spec/cart.md")
		if document.SchemaVersion != "1.2" || document.Run.StatementCanonicalization != domain.StatementCanonicalization {
			t.Fatalf("got %q and %q", document.SchemaVersion, document.Run.StatementCanonicalization)
		}
	})

	t.Run("every requirement carries the digest of its canonical form [verifies SN-89~1]", func(t *testing.T) {
		document, _ := proofGateReport(t, cartFixtures+"/spec/cart.md")
		for _, requirement := range document.Requirements {
			recomputed := domain.Requirement{
				Title: requirement.Title, Statement: requirement.Statement,
				Status: requirement.Status, Justification: requirement.Justification,
			}.StatementDigest()
			if requirement.StatementDigest != recomputed {
				t.Errorf("%s: digest %s, recomputed %s", requirement.ID, requirement.StatementDigest, recomputed)
			}
		}
		if got := requirementOf(t, document, "CART-2").StatementDigest; got != cartPlanDigest {
			t.Fatalf("CART-2 has digest %s, the specification states %s", got, cartPlanDigest)
		}
	})

	t.Run("a requirement has the same digest in its Markdown and YAML specs [verifies SN-89~1]", func(t *testing.T) {
		fromMarkdown, _ := proofGateReport(t, cartFixtures+"/spec/cart.md")
		fromYAML, _ := proofGateReport(t, cartFixtures+"/spec/cart.yaml")
		for _, requirement := range fromMarkdown.Requirements {
			if twin := requirementOf(t, fromYAML, string(requirement.ID)); twin.StatementDigest != requirement.StatementDigest {
				t.Errorf("%s: %s in Markdown, %s in YAML", requirement.ID, requirement.StatementDigest, twin.StatementDigest)
			}
		}
	})

	t.Run("a requirement lists its inline examples with typed arguments and outcome [verifies SN-94~1]", func(t *testing.T) {
		document, _ := proofGateReport(t, proofGateFixtures+"/parking.md", proofGateFixtures+"/parking.yaml")
		examples := requirementOf(t, document, "PARK-2").Examples
		if len(examples) != 2 || examples[0].Index != 1 || examples[1].Index != 2 {
			t.Fatalf("got %+v", examples)
		}
		first := examples[0]
		wantArgs := map[string]report.LiteralInfo{
			"entry": {Type: domain.LiteralDatetime, Value: "2026-03-02T10:00"},
			"exit":  {Type: domain.LiteralDatetime, Value: "2026-03-02T10:15"},
		}
		if first.Operation != "quote" || !equalArgs(first.Args, wantArgs) ||
			first.Expect != (report.OutcomeInfo{Kind: domain.OutcomeReturns, Type: domain.LiteralDecimal, Value: "0.00"}) {
			t.Fatalf("got %+v", first)
		}
		if raised := examples[1].Expect; raised != (report.OutcomeInfo{Kind: domain.OutcomeRaises, Type: domain.LiteralName, Value: "InvalidStay"}) {
			t.Fatalf("got %+v", raised)
		}
		yamlExamples := requirementOf(t, document, "PARK-4").Examples
		if len(yamlExamples) != 1 || yamlExamples[0].Args["minutes"] != (report.LiteralInfo{Type: domain.LiteralInteger, Value: "15"}) {
			t.Fatalf("got %+v", yamlExamples)
		}
	})

	t.Run("a requirement without examples lists none, as an empty array [verifies SN-94~1]", func(t *testing.T) {
		_, text := proofGateReport(t, cartFixtures+"/spec/cart.md")
		var generic struct {
			Requirements []map[string]json.RawMessage `json:"requirements"`
		}
		if err := json.Unmarshal([]byte(text), &generic); err != nil {
			t.Fatal(err)
		}
		for _, requirement := range generic.Requirements {
			if string(requirement["examples"]) != "[]" {
				t.Errorf("examples is %s", requirement["examples"])
			}
		}
	})

	t.Run("a malformed example and a detached example are warnings at their lines [verifies SN-92~1]", func(t *testing.T) {
		document, _ := proofGateReport(t, proofGateFixtures+"/parking.md")
		var malformed, detached []report.FindingInfo
		for _, finding := range document.Findings {
			switch finding.Category {
			case domain.CategoryMalformedExample:
				malformed = append(malformed, finding)
			case domain.CategoryDetachedExample:
				detached = append(detached, finding)
			}
		}
		if len(malformed) != 1 || malformed[0].Location.Line != 3 || malformed[0].RequirementID != "PARK-2" ||
			len(detached) != 1 || detached[0].Location.Line != 13 || detached[0].Severity != domain.SeverityWarning {
			t.Fatalf("malformed %+v, detached %+v", malformed, detached)
		}
	})

	t.Run("the report lists the out-of-scope declarations of every spec, ordered by file and line [verifies SN-98~1]", func(t *testing.T) {
		document, _ := proofGateReport(t, proofGateFixtures+"/parking.yaml", proofGateFixtures+"/parking.md")
		want := []report.OutOfScopeInfo{
			{SpecFile: proofGateFixtures + "/parking.md", Line: 8, Dimension: "concurrency", Reason: "one gate per lane."},
			{SpecFile: proofGateFixtures + "/parking.md", Line: 9, Dimension: "payment", Reason: "cards only."},
			{SpecFile: proofGateFixtures + "/parking.yaml", Line: 8, Dimension: "boundaries", Reason: "stays stay under a day"},
		}
		if len(document.OutOfScope) != len(want) {
			t.Fatalf("got %+v", document.OutOfScope)
		}
		for index := range want {
			if document.OutOfScope[index] != want[index] {
				t.Errorf("declaration %d: got %+v, want %+v", index, document.OutOfScope[index], want[index])
			}
		}
	})

	t.Run("a spec without declarations leaves out_of_scope an empty array [verifies SN-98~1]", func(t *testing.T) {
		_, text := proofGateReport(t, cartFixtures+"/spec/cart.md")
		if !strings.Contains(text, "\"out_of_scope\": []") {
			t.Fatalf("no empty out_of_scope in:\n%s", text)
		}
	})

	t.Run("an unknown dimension is a warning at its line in the report [verifies SN-97~1]", func(t *testing.T) {
		document, _ := proofGateReport(t, proofGateFixtures+"/parking.md")
		found := false
		for _, finding := range document.Findings {
			if finding.Category == domain.CategoryUnknownDimension {
				found = finding.Location.Line == 9 && finding.Severity == domain.SeverityWarning && strings.Contains(finding.Message, `"payment"`)
			}
		}
		if !found {
			t.Fatalf("got %+v", document.Findings)
		}
	})

	t.Run("a report with examples, declarations and findings conforms to the published schema [verifies SN-60~1]", func(t *testing.T) {
		_, text := proofGateReport(t, proofGateFixtures+"/parking.md", proofGateFixtures+"/parking.yaml")
		instance, err := jsonschema.UnmarshalJSON(strings.NewReader(text))
		if err != nil {
			t.Fatal(err)
		}
		if err := reportSchema(t).Validate(instance); err != nil {
			t.Fatalf("%v", err)
		}
	})
}

func equalArgs(got, want map[string]report.LiteralInfo) bool {
	if len(got) != len(want) {
		return false
	}
	for name, value := range want {
		if got[name] != value {
			return false
		}
	}
	return true
}
