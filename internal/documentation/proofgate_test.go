package documentation_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/cli"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

const (
	reportDocument        = "docs/report.md"
	configurationDocument = "docs/configuration.md"
	pipelineDocument      = "docs/pipeline-gate.md"
	categoriesHeading     = "### Finding categories"
	routingHeading        = "### Routing table: finding category to owner and action"
	checkFlagsHeading     = "## Flags of `shallnot check`"
	dimensionsHeading     = "## Out-of-scope declarations"
)

var (
	helpFlag     = regexp.MustCompile(`(?m)^\s+-([a-z][a-z-]*)`)
	documentFlag = regexp.MustCompile("`--([a-z][a-z-]*)")
)

// codeSpansOfFirstColumn returns the code spans of the first cell of each row of a table.
func codeSpansOfFirstColumn(t *testing.T, rows [][]string) []string {
	t.Helper()
	var names []string
	for _, row := range rows {
		for _, span := range codeSpan.FindAllStringSubmatch(row[0], -1) {
			names = append(names, span[1])
		}
	}
	sort.Strings(names)
	return names
}

func sorted(names []string) []string {
	copied := append([]string(nil), names...)
	sort.Strings(copied)
	return copied
}

func TestProofGateDocumentation(t *testing.T) {
	t.Run("docs/report.md documents each finding category the code defines and no other [verifies SN-100~1]", func(t *testing.T) {
		documented := codeSpansOfFirstColumn(t, tableRows(t, section(t, read(t, reportDocument), categoriesHeading)))
		if want := sorted(domain.CategoryNames()); !reflect.DeepEqual(documented, want) {
			t.Fatalf("documented %v; the code defines %v", documented, want)
		}
	})

	t.Run("the routing table of docs/pipeline-gate.md routes each finding category and no other [verifies SN-100~1]", func(t *testing.T) {
		routed := codeSpansOfFirstColumn(t, tableRows(t, section(t, read(t, pipelineDocument), routingHeading)))
		categories := map[string]bool{}
		for _, name := range domain.CategoryNames() {
			categories[name] = true
		}
		seen := map[string]bool{}
		for _, name := range routed {
			if !categories[name] {
				t.Errorf("the routing table names %q, which is no category", name)
			}
			seen[name] = true
		}
		for name := range categories {
			if !seen[name] {
				t.Errorf("the routing table does not route %s", name)
			}
		}
	})

	t.Run("docs/spec-format.md lists the out-of-scope dimensions the code accepts and no other [verifies SN-100~1]", func(t *testing.T) {
		documented := codeSpansOfFirstColumn(t, tableRows(t, section(t, read(t, specFormat), dimensionsHeading)))
		if want := sorted(domain.Dimensions()); !reflect.DeepEqual(documented, want) {
			t.Fatalf("documented %v; the code accepts %v", documented, want)
		}
	})

	t.Run("docs/configuration.md documents each flag of shallnot check and no other [verifies SN-100~1]", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		cli.Main([]string{"check", "--help"}, strings.NewReader(""), &stdout, &stderr)
		implemented := map[string]bool{}
		for _, flag := range helpFlag.FindAllStringSubmatch(stderr.String(), -1) {
			implemented[flag[1]] = true
		}
		documented := map[string]bool{}
		for _, row := range tableRows(t, section(t, read(t, configurationDocument), checkFlagsHeading)) {
			if flag := documentFlag.FindStringSubmatch(row[0]); flag != nil {
				documented[flag[1]] = true
			}
		}
		if got, want := sortedKeys(documented), sortedKeys(implemented); !reflect.DeepEqual(got, want) {
			t.Fatalf("documented %v; shallnot check accepts %v", got, want)
		}
	})
}

const (
	proofsDocument    = "docs/proofs.md"
	proofsKeysHeading = "## The `proofs` block"
	extensionHeading  = "## Proof extension"
	manifestHeading   = "### Manifest"
	reasonsHeading    = "### Pending reasons"
	trustHeading      = "## Trust levels"
	basesHeading      = "### Pass bases"
	isolationHeading  = "### Isolation levels"
	configSchemaPath  = "schemas/config.schema.json"
	reportSchemaPath  = "schemas/report.schema.json"
)

// schemaProperties lists the property names of the object at a path of a JSON Schema file.
func schemaProperties(t *testing.T, file string, path ...string) []string {
	t.Helper()
	var node any
	if err := json.Unmarshal([]byte(read(t, file)), &node); err != nil {
		t.Fatal(err)
	}
	for _, key := range path {
		node = node.(map[string]any)[key]
	}
	var names []string
	for name := range node.(map[string]any)["properties"].(map[string]any) {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func jsonKeys(value any) []string {
	var names []string
	kind := reflect.TypeOf(value)
	for index := range kind.NumField() {
		name, _, _ := strings.Cut(kind.Field(index).Tag.Get("json"), ",")
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func TestProofsDocumentation(t *testing.T) {
	check := func(name, document, heading string, want []string) {
		t.Run(name+" [verifies SN-126~1]", func(t *testing.T) {
			got := codeSpansOfFirstColumn(t, tableRows(t, section(t, read(t, document), heading)))
			if !reflect.DeepEqual(got, sorted(want)) {
				t.Fatalf("documented %v; the code defines %v", got, sorted(want))
			}
		})
	}
	var reasons []string
	for _, reason := range proof.PendingReasons() {
		reasons = append(reasons, string(reason))
	}
	check("docs/configuration.md documents each key of the proofs block and no other", configurationDocument, proofsKeysHeading, schemaProperties(t, configSchemaPath, "properties", "proofs"))
	check("docs/report.md documents each property of extensions.proofs and no other", reportDocument, extensionHeading, schemaProperties(t, reportSchemaPath, "$defs", "proofs"))
	check("docs/proofs.md documents each key of the manifest and no other", proofsDocument, manifestHeading, jsonKeys(proof.Manifest{}))
	check("docs/proofs.md documents each pending reason and no other", proofsDocument, reasonsHeading, reasons)
	check("docs/proofs.md documents each trust level and no other", proofsDocument, trustHeading, proof.TrustNames())
	check("docs/proofs.md documents each pass basis and no other", proofsDocument, basesHeading, proof.SignedPassBasisNames())
	check("docs/proofs.md documents each isolation level and no other", proofsDocument, isolationHeading, proof.IsolationNames())
}
