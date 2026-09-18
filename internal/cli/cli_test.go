package cli_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/RachidChabane/shallnot/internal/adapters/report"
	"github.com/RachidChabane/shallnot/internal/app"
	"github.com/RachidChabane/shallnot/internal/cli"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/schemas"
)

var update = flag.Bool("update", false, "rewrite the expected reports under fixtures/")

const (
	repositoryRoot = "../.."
	cartFixtures   = "fixtures/cart"
	pipelineConfig = "fixtures/pipeline/shallnot.yaml"
	currentPlan    = "fixtures/pipeline/plans/ABC-101.md"
)

var ecosystems = []string{"pytest", "jest", "vitest", "junit5-maven", "junit5-gradle", "junit5-kotlin"}

type run struct {
	exit   app.ExitCode
	stdout string
	stderr string
}

func shallnot(t *testing.T, args ...string) run {
	t.Helper()
	var stdout, stderr bytes.Buffer
	exit := cli.Main(args, &stdout, &stderr)
	return run{exit: exit, stdout: stdout.String(), stderr: stderr.String()}
}

func inRepository(t *testing.T) {
	t.Helper()
	t.Chdir(repositoryRoot)
}

func cartArgs(ecosystem, format string) []string {
	return []string{"check", "--no-config", "--specs", cartFixtures + "/spec/cart.md", "--tests", cartFixtures + "/" + ecosystem,
		"--results", cartFixtures + "/" + ecosystem + "/results/" + resultsOf(ecosystem), "--format", format}
}

// resultsOf selects the results of an ecosystem: pytest keeps one file per
// junit_family side by side, the others have one run per directory.
func resultsOf(ecosystem string) string {
	if ecosystem == "pytest" {
		return "junit.xml"
	}
	return "*.xml"
}

func decode(t *testing.T, text string) report.Document {
	t.Helper()
	var document report.Document
	if err := json.Unmarshal([]byte(text), &document); err != nil {
		t.Fatalf("report is not JSON: %v\n%s", err, text)
	}
	return document
}

func expectGolden(t *testing.T, path, got string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run the tests with -update to create it)", err)
	}
	if got != string(want) {
		t.Fatalf("%s differs from the report:\n%s", path, got)
	}
}

func reportSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemas.Report))
	if err != nil {
		t.Fatal(err)
	}
	closeObjects(document)
	compiler := jsonschema.NewCompiler()
	const location = "report.schema.json"
	if err := compiler.AddResource(location, document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(location)
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

// closeObjects forbids undocumented properties, so that validation also proves
// the report emits nothing the schema does not describe.
func closeObjects(node any) {
	switch value := node.(type) {
	case map[string]any:
		if _, documented := value["properties"]; documented {
			if _, open := value["additionalProperties"]; !open {
				value["additionalProperties"] = false
			}
		}
		for _, child := range value {
			closeObjects(child)
		}
	case []any:
		for _, child := range value {
			closeObjects(child)
		}
	}
}

func TestEcosystems(t *testing.T) {
	inRepository(t)
	schema := reportSchema(t)
	for _, ecosystem := range ecosystems {
		t.Run("traces the "+ecosystem+" project from its real results [verifies SN-22~1]", func(t *testing.T) {
			result := shallnot(t, cartArgs(ecosystem, "json")...)
			if result.exit != app.ExitBlocked {
				t.Fatalf("exit %d, stderr %s", result.exit, result.stderr)
			}
			expectGolden(t, filepath.Join(cartFixtures, "expected", ecosystem+".json"), result.stdout)
			coverage := map[domain.RequirementID]domain.Coverage{}
			for _, requirement := range decode(t, result.stdout).Requirements {
				coverage[requirement.ID] = requirement.Coverage
			}
			want := map[domain.RequirementID]domain.Coverage{
				"CART-1": domain.CoverageCovered, "CART-2": domain.CoverageCovered, "CART-3": domain.CoverageFailed,
				"CART-4": domain.CoverageSkipped, "CART-5": domain.CoverageNotRun, "CART-6": domain.CoverageUncovered,
				"CART-7": domain.CoverageNonTestable,
			}
			for id, expected := range want {
				if coverage[id] != expected {
					t.Errorf("%s is %s, want %s", id, coverage[id], expected)
				}
			}
		})
		t.Run("the "+ecosystem+" report conforms to the published schema [verifies SN-60~1]", func(t *testing.T) {
			instance, err := jsonschema.UnmarshalJSON(strings.NewReader(shallnot(t, cartArgs(ecosystem, "json")...).stdout))
			if err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(instance); err != nil {
				t.Fatalf("%v", err)
			}
		})
	}

	t.Run("a YAML spec yields the same matrix as its Markdown twin [verifies SN-5~1]", func(t *testing.T) {
		args := cartArgs("pytest", "json")
		fromMarkdown := decode(t, shallnot(t, args...).stdout)
		args[3] = cartFixtures + "/spec/cart.yaml"
		fromYAML := decode(t, shallnot(t, args...).stdout)
		if len(fromYAML.Requirements) != len(fromMarkdown.Requirements) {
			t.Fatalf("%d requirements from YAML, %d from Markdown", len(fromYAML.Requirements), len(fromMarkdown.Requirements))
		}
		coverage := map[domain.RequirementID]report.RequirementInfo{}
		for _, requirement := range fromMarkdown.Requirements {
			coverage[requirement.ID] = requirement
		}
		for _, requirement := range fromYAML.Requirements {
			twin := coverage[requirement.ID]
			if twin.Coverage != requirement.Coverage || twin.Revision != requirement.Revision || twin.Statement != requirement.Statement || twin.Justification != requirement.Justification {
				t.Errorf("%s differs:\nyaml     %+v\nmarkdown %+v", requirement.ID, requirement, twin)
			}
		}
	})
}

func TestReports(t *testing.T) {
	inRepository(t)
	golden := func(format, file string) func(t *testing.T) {
		return func(t *testing.T) {
			result := shallnot(t, cartArgs("pytest", format)...)
			if result.exit != app.ExitBlocked {
				t.Fatalf("exit %d, stderr %s", result.exit, result.stderr)
			}
			expectGolden(t, filepath.Join(cartFixtures, "expected", file), result.stdout)
		}
	}
	t.Run("renders the terminal report [verifies SN-64~1]", golden("terminal", "pytest.terminal.txt"))
	t.Run("renders the Markdown summary [verifies SN-62~1]", golden("markdown", "pytest.summary.md"))
	t.Run("renders the GitHub annotations [verifies SN-63~1]", golden("github", "pytest.annotations.txt"))

	t.Run("annotations point at the offending source line [verifies SN-63~1]", func(t *testing.T) {
		annotations := shallnot(t, cartArgs("pytest", "github")...).stdout
		want := "::error file=fixtures/cart/pytest/tests/test_misc.py,line=6,title=shallnot orphan_tag::"
		if !strings.Contains(annotations, want) {
			t.Fatalf("no %q in:\n%s", want, annotations)
		}
	})

	t.Run("writes the JSON and Markdown files beside the terminal report, with annotations on request [verifies SN-62~1]", func(t *testing.T) {
		directory := t.TempDir()
		jsonPath, markdownPath := filepath.Join(directory, "report.json"), filepath.Join(directory, "summary.md")
		args := append(cartArgs("pytest", "terminal"), "--json-out", jsonPath, "--markdown-out", markdownPath, "--github-annotations")
		result := shallnot(t, args...)
		if !strings.HasPrefix(result.stdout, "shallnot: FAIL") || !strings.Contains(result.stdout, "\n::error file=") {
			t.Fatalf("stdout:\n%s", result.stdout)
		}
		written, err := os.ReadFile(jsonPath)
		if err != nil || decode(t, string(written)).Verdict != domain.VerdictFail {
			t.Fatalf("json file: %v", err)
		}
		if summary, err := os.ReadFile(markdownPath); err != nil || !strings.HasPrefix(string(summary), "## shallnot: FAIL") {
			t.Fatalf("markdown file: %v", err)
		}
	})

	t.Run("repeated runs are byte-identical in every format [verifies SN-61~1]", func(t *testing.T) {
		for _, format := range report.FormatNames() {
			for _, ecosystem := range ecosystems {
				first := shallnot(t, cartArgs(ecosystem, format)...)
				for range 3 {
					if again := shallnot(t, cartArgs(ecosystem, format)...); again != first {
						t.Fatalf("%s/%s differs between runs", ecosystem, format)
					}
				}
			}
		}
	})

	t.Run("prints each published schema [verifies SN-65~1]", func(t *testing.T) {
		for name, schema := range map[string][]byte{"report": schemas.Report, "config": schemas.Config, "spec": schemas.Spec} {
			result := shallnot(t, "schema", name)
			if result.exit != app.ExitClean || result.stdout != string(schema) || !json.Valid([]byte(result.stdout)) {
				t.Errorf("schema %s: exit %d", name, result.exit)
			}
		}
		if result := shallnot(t, "schema", "unknown"); result.exit != app.ExitToolFailure {
			t.Errorf("unknown schema: exit %d", result.exit)
		}
	})
}

func TestScope(t *testing.T) {
	inRepository(t)

	t.Run("a focused run demands the current plan only and resolves earlier tickets' tags [verifies SN-40~1]", func(t *testing.T) {
		result := shallnot(t, "check", "--config", pipelineConfig, "--focus", currentPlan, "--format", "json")
		document := decode(t, result.stdout)
		if result.exit != app.ExitClean || document.Verdict != domain.VerdictPass || len(document.Findings) != 0 {
			t.Fatalf("exit %d, findings %+v, stderr %s", result.exit, document.Findings, result.stderr)
		}
		if document.Summary.Requirements.Known != 5 || document.Summary.Requirements.InFocus != 3 || document.Run.Focus.Everything {
			t.Fatalf("got %+v, focus %+v", document.Summary.Requirements, document.Run.Focus)
		}
	})

	t.Run("without focus every known requirement is demanded [verifies SN-40~1]", func(t *testing.T) {
		result := shallnot(t, "check", "--config", pipelineConfig, "--format", "json")
		document := decode(t, result.stdout)
		if result.exit != app.ExitBlocked || !document.Run.Focus.Everything || len(document.Findings) != 1 || document.Findings[0].RequirementID != "ABC-100.AC2" {
			t.Fatalf("exit %d, findings %+v", result.exit, document.Findings)
		}
	})

	t.Run("a tag citing a requirement outside the known specs is an orphan [verifies SN-40~1]", func(t *testing.T) {
		result := shallnot(t, "check", "--config", pipelineConfig, "--specs", currentPlan, "--format", "json")
		document := decode(t, result.stdout)
		if result.exit != app.ExitBlocked || len(document.Findings) != 1 || document.Findings[0].Category != domain.CategoryOrphanTag {
			t.Fatalf("exit %d, findings %+v", result.exit, document.Findings)
		}
	})

	t.Run("focus can be a glob on requirement IDs [verifies SN-41~1]", func(t *testing.T) {
		result := shallnot(t, "check", "--config", pipelineConfig, "--focus-id", "ABC-100.*", "--format", "json")
		document := decode(t, result.stdout)
		if result.exit != app.ExitBlocked || document.Summary.Requirements.InFocus != 2 || document.Summary.Requirements.Uncovered != 1 {
			t.Fatalf("exit %d, summary %+v", result.exit, document.Summary.Requirements)
		}
	})

	t.Run("a requirement is verified by tests and results of another repository [verifies SN-42~1]", func(t *testing.T) {
		document := decode(t, shallnot(t, "check", "--config", pipelineConfig, "--focus", currentPlan, "--format", "json").stdout)
		if len(document.Run.ResultsFiles) != 2 || len(document.Run.TestRoots) != 2 {
			t.Fatalf("got %+v", document.Run)
		}
		origins := map[domain.RequirementID]string{}
		for _, requirement := range document.Requirements {
			for _, test := range requirement.Tests {
				origins[requirement.ID] = test.ResultsFile + " " + test.Source.File
			}
		}
		if origins["ABC-101.AC1"] != "fixtures/pipeline/service/results/junit.xml fixtures/pipeline/service/tests/test_invoices.py" ||
			origins["ABC-101.AC2"] != "fixtures/pipeline/client/results/junit.xml fixtures/pipeline/client/test/export.test.ts" {
			t.Fatalf("got %+v", origins)
		}
	})

	t.Run("leaving one repository's results out turns its requirement into not_run [verifies SN-42~1]", func(t *testing.T) {
		result := shallnot(t, "check", "--config", pipelineConfig, "--focus", currentPlan, "--results", "fixtures/pipeline/service/results", "--format", "json")
		document := decode(t, result.stdout)
		if result.exit != app.ExitBlocked || document.Summary.Requirements.NotRun != 1 {
			t.Fatalf("exit %d, summary %+v", result.exit, document.Summary.Requirements)
		}
	})
}

func TestGate(t *testing.T) {
	inRepository(t)

	t.Run("advisory mode reports a failing verdict and exits 0 [verifies SN-51~1]", func(t *testing.T) {
		result := shallnot(t, append(cartArgs("pytest", "json"), "--advisory")...)
		document := decode(t, result.stdout)
		if result.exit != app.ExitClean || document.Verdict != domain.VerdictFail || !document.Run.Advisory || document.ExitCode != app.ExitClean || document.Summary.Findings.Blocking == 0 {
			t.Fatalf("exit %d, verdict %s, run %+v", result.exit, document.Verdict, document.Run)
		}
	})

	t.Run("exits 0 when clean and 1 when blocked [verifies SN-52~1]", func(t *testing.T) {
		if clean := shallnot(t, "check", "--config", pipelineConfig, "--focus", currentPlan); clean.exit != app.ExitClean {
			t.Fatalf("clean run: exit %d", clean.exit)
		}
		if blocked := shallnot(t, "check", "--config", pipelineConfig); blocked.exit != app.ExitBlocked {
			t.Fatalf("blocked run: exit %d", blocked.exit)
		}
	})

	failures := map[string][]string{
		"no command":                           {},
		"an unknown command":                   {"verify"},
		"an unknown flag":                      {"check", "--nope"},
		"a stray argument":                     {"check", "stray"},
		"no spec":                              {"check", "--no-config", "--results", cartFixtures + "/pytest/results"},
		"no results":                           {"check", "--no-config", "--specs", cartFixtures + "/spec"},
		"a spec glob matching nothing":         {"check", "--no-config", "--specs", "missing/*.md", "--results", cartFixtures + "/pytest/results"},
		"a results glob matching nothing":      {"check", "--no-config", "--specs", cartFixtures + "/spec/cart.md", "--results", "missing/*.xml"},
		"a results file that is not JUnit XML": {"check", "--no-config", "--specs", cartFixtures + "/spec/cart.md", "--results", "go.mod"},
		"a missing test root":                  {"check", "--no-config", "--specs", cartFixtures + "/spec/cart.md", "--results", cartFixtures + "/pytest/results", "--tests", "missing"},
		"a missing config file":                {"check", "--config", "missing.yaml"},
		"an invalid id pattern":                {"check", "--config", pipelineConfig, "--id-pattern", "["},
		"an unknown severity":                  {"check", "--config", pipelineConfig, "--severity", "orphan_tag=fatal"},
		"an unknown category":                  {"check", "--config", pipelineConfig, "--severity", "orphans=error"},
		"a fail-on that never blocks":          {"check", "--config", pipelineConfig, "--fail-on", "off"},
		"an unknown format":                    {"check", "--config", pipelineConfig, "--format", "xml"},
	}
	for name, args := range failures {
		t.Run("exits 2 without a verdict on "+name+" [verifies SN-52~1]", func(t *testing.T) {
			result := shallnot(t, args...)
			if result.exit != app.ExitToolFailure || result.stdout != "" || result.stderr == "" {
				t.Fatalf("exit %d\nstdout %q\nstderr %q", result.exit, result.stdout, result.stderr)
			}
		})
	}

	t.Run("a tool failure leaves no earlier report behind [verifies SN-54~1]", func(t *testing.T) {
		stale := filepath.Join(t.TempDir(), "report.json")
		if err := os.WriteFile(stale, []byte(`{"verdict":"pass"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		result := shallnot(t, "check", "--no-config", "--specs", cartFixtures+"/spec/cart.md", "--results", "missing/*.xml", "--json-out", stale)
		if _, err := os.Stat(stale); result.exit != app.ExitToolFailure || !os.IsNotExist(err) {
			t.Fatalf("exit %d, stat error %v", result.exit, err)
		}
	})
}

func TestConfiguration(t *testing.T) {
	inRepository(t)

	t.Run("reads the config file, resolving its paths against its directory [verifies SN-53~1]", func(t *testing.T) {
		document := decode(t, shallnot(t, "check", "--config", pipelineConfig, "--format", "json").stdout)
		if document.Run.IDPattern != `[A-Z]+-[0-9]+\.AC[0-9]+` || len(document.Run.SpecFiles) != 2 || document.Run.SpecFiles[0] != "fixtures/pipeline/plans/ABC-100.md" {
			t.Fatalf("got %+v", document.Run)
		}
	})

	t.Run("picks up shallnot.yaml from the working directory [verifies SN-53~1]", func(t *testing.T) {
		t.Chdir("fixtures/pipeline")
		document := decode(t, shallnot(t, "check", "--format", "json").stdout)
		if len(document.Run.SpecFiles) != 2 || document.Run.SpecFiles[0] != "plans/ABC-100.md" {
			t.Fatalf("got %+v", document.Run)
		}
	})

	t.Run("flags override the config file [verifies SN-53~1]", func(t *testing.T) {
		result := shallnot(t, "check", "--config", pipelineConfig, "--severity", "uncovered_requirement=warning", "--format", "json")
		if document := decode(t, result.stdout); result.exit != app.ExitClean || document.Summary.Findings.Warning != 1 {
			t.Fatalf("exit %d, summary %+v", result.exit, document.Summary.Findings)
		}
		if strict := shallnot(t, "check", "--config", pipelineConfig, "--fail-on", "warning", "--severity", "uncovered_requirement=warning"); strict.exit != app.ExitBlocked {
			t.Fatalf("exit %d with fail-on warning", strict.exit)
		}
	})

	t.Run("strict mode turns untagged tests into blocking findings [verifies SN-36~1]", func(t *testing.T) {
		relaxed := []string{"--severity", "uncovered_requirement=off", "--severity", "failed_requirement=off", "--severity", "skipped_requirement=off",
			"--severity", "not_run_requirement=off", "--severity", "orphan_tag=off", "--severity", "revision_mismatch=off", "--severity", "malformed_tag=off"}
		if lenient := shallnot(t, append(cartArgs("pytest", "json"), relaxed...)...); lenient.exit != app.ExitClean {
			t.Fatalf("lenient run: exit %d", lenient.exit)
		}
		if strict := shallnot(t, append(append(cartArgs("pytest", "json"), relaxed...), "--strict")...); strict.exit != app.ExitBlocked {
			t.Fatalf("strict run: exit %d", strict.exit)
		}
	})

	for name, text := range map[string]string{
		"an unknown key":      "version: 1\nspec: [plans]\n",
		"a missing version":   "specs: [plans]\n",
		"an unknown category": "version: 1\nseverities: {orphans: error}\n",
		"invalid YAML":        "version: [1\n",
	} {
		t.Run("refuses a config file with "+name+" [verifies SN-53~1]", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "shallnot.yaml")
			if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
			if result := shallnot(t, "check", "--config", path); result.exit != app.ExitToolFailure || result.stdout != "" {
				t.Fatalf("exit %d, stdout %q", result.exit, result.stdout)
			}
		})
	}
}
