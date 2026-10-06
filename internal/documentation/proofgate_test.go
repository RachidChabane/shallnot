package documentation_test

import (
	"bytes"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/cli"
	"github.com/RachidChabane/shallnot/internal/domain"
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
