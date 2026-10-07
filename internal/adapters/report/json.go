package report

import (
	"encoding/json"
	"io"

	"github.com/RachidChabane/shallnot/internal/app"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/version"
)

// SchemaVersion is the version of the JSON report format. The major number
// changes only when a consumer of an earlier report would break.
const SchemaVersion = "1.2"

// Document is the JSON report. Its shape is specified by schemas/report.schema.json.
type Document struct {
	SchemaVersion string            `json:"schema_version"`
	Tool          ToolInfo          `json:"tool"`
	Run           RunInfo           `json:"run"`
	Verdict       domain.Verdict    `json:"verdict"`
	ExitCode      app.ExitCode      `json:"exit_code"`
	Summary       Summary           `json:"summary"`
	Requirements  []RequirementInfo `json:"requirements"`
	OutOfScope    []OutOfScopeInfo  `json:"out_of_scope"`
	Findings      []FindingInfo     `json:"findings"`
	Extensions    Extensions        `json:"extensions"`
}

// Extensions is reserved for data contributed by analyses outside the core.
type Extensions map[string]json.RawMessage

type ToolInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type RunInfo struct {
	Advisory  bool            `json:"advisory"`
	FailOn    domain.Severity `json:"fail_on"`
	IDPattern string          `json:"id_pattern"`
	// StatementCanonicalization names the canonical form the statement digests are computed over.
	StatementCanonicalization string    `json:"statement_canonicalization"`
	Focus                     FocusInfo `json:"focus"`
	SpecFiles                 []string  `json:"spec_files"`
	TestRoots                 []string  `json:"test_roots"`
	ResultsFiles              []string  `json:"results_files"`
	// Commit is absent when the caller named none.
	Commit string `json:"commit,omitempty"`
}

type FocusInfo struct {
	Everything bool     `json:"everything"`
	Files      []string `json:"files"`
	IDs        []string `json:"ids"`
}

type Summary struct {
	Requirements RequirementSummary `json:"requirements"`
	Tests        TestSummary        `json:"tests"`
	Findings     FindingSummary     `json:"findings"`
}

type RequirementSummary struct {
	Known       int `json:"known"`
	InFocus     int `json:"in_focus"`
	Covered     int `json:"covered"`
	Failed      int `json:"failed"`
	Skipped     int `json:"skipped"`
	NotRun      int `json:"not_run"`
	Uncovered   int `json:"uncovered"`
	NonTestable int `json:"non_testable"`
}

type TestSummary struct {
	Total    int `json:"total"`
	Bound    int `json:"bound"`
	Untagged int `json:"untagged"`
}

type FindingSummary struct {
	Error    int `json:"error"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
	Blocking int `json:"blocking"`
}

type LocationInfo struct {
	File string `json:"file"`
	Line int    `json:"line,omitempty"`
}

type RequirementInfo struct {
	ID              domain.RequirementID `json:"id"`
	Revision        domain.Revision      `json:"revision"`
	Ref             string               `json:"ref"`
	Title           string               `json:"title,omitempty"`
	Statement       string               `json:"statement"`
	StatementDigest string               `json:"statement_digest"`
	Status          domain.Status        `json:"status"`
	Justification   string               `json:"justification,omitempty"`
	Location        LocationInfo         `json:"location"`
	InFocus         bool                 `json:"in_focus"`
	Examples        []ExampleInfo        `json:"examples"`
	Coverage        domain.Coverage      `json:"coverage"`
	Tests           []TestInfo           `json:"tests"`
	Extensions      Extensions           `json:"extensions"`
}

// OutOfScopeInfo is a spec's declaration that a dimension is not covered.
type OutOfScopeInfo struct {
	SpecFile  string `json:"spec_file"`
	Line      int    `json:"line"`
	Dimension string `json:"dimension"`
	Reason    string `json:"reason"`
}

// LiteralInfo is a typed value of an example.
type LiteralInfo struct {
	Type  domain.LiteralType `json:"type"`
	Value string             `json:"value"`
}

// OutcomeInfo is what an example expects.
type OutcomeInfo struct {
	Kind  domain.OutcomeKind `json:"kind"`
	Type  domain.LiteralType `json:"type"`
	Value string             `json:"value"`
}

// ExampleInfo is one inline example of a statement.
type ExampleInfo struct {
	Index     int                    `json:"index"`
	Operation string                 `json:"operation"`
	Args      map[string]LiteralInfo `json:"args"`
	Expect    OutcomeInfo            `json:"expect"`
}

type TestInfo struct {
	Name        string         `json:"name"`
	ClassName   string         `json:"classname,omitempty"`
	Suite       string         `json:"suite,omitempty"`
	Outcome     domain.Outcome `json:"outcome"`
	ResultsFile string         `json:"results_file,omitempty"`
	Source      *LocationInfo  `json:"source,omitempty"`
}

type FindingInfo struct {
	Category      domain.Category      `json:"category"`
	Severity      domain.Severity      `json:"severity"`
	Blocking      bool                 `json:"blocking"`
	Message       string               `json:"message"`
	RequirementID domain.RequirementID `json:"requirement_id,omitempty"`
	Module        string               `json:"module,omitempty"`
	Location      LocationInfo         `json:"location"`
	Test          *TestRefInfo         `json:"test,omitempty"`
}

type TestRefInfo struct {
	Name        string `json:"name"`
	ClassName   string `json:"classname,omitempty"`
	ResultsFile string `json:"results_file"`
}

// JSONReporter writes the machine-readable report.
type JSONReporter struct{}

// Write encodes the report with a stable key order and indentation.
func (JSONReporter) Write(writer io.Writer, outcome app.Outcome) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(BuildDocument(outcome))
}

// BuildDocument maps a run to the JSON report.
func BuildDocument(outcome app.Outcome) Document {
	document := Document{
		SchemaVersion: SchemaVersion,
		Tool:          ToolInfo{Name: version.Name, Version: version.Version},
		Run: RunInfo{
			Advisory:                  outcome.Policy.Advisory,
			FailOn:                    outcome.Policy.FailOn,
			IDPattern:                 outcome.Inputs.IDPattern,
			StatementCanonicalization: domain.StatementCanonicalization,
			Focus: FocusInfo{
				Everything: outcome.Inputs.FocusIsEverything(),
				Files:      orEmpty(outcome.Inputs.FocusFiles),
				IDs:        orEmpty(outcome.Inputs.FocusIDs),
			},
			SpecFiles:    orEmpty(outcome.Inputs.SpecFiles),
			TestRoots:    orEmpty(outcome.Inputs.TestRoots),
			ResultsFiles: orEmpty(outcome.Inputs.ResultsFiles),
			Commit:       outcome.Inputs.Commit,
		},
		Verdict:      outcome.Analysis.Verdict,
		ExitCode:     outcome.ExitCode(),
		Summary:      Summarize(outcome.Analysis),
		Requirements: []RequirementInfo{},
		OutOfScope:   []OutOfScopeInfo{},
		Findings:     []FindingInfo{},
		Extensions:   Extensions{},
	}
	for _, row := range outcome.Analysis.Rows {
		document.Requirements = append(document.Requirements, requirementInfo(row))
	}
	for _, declaration := range outcome.Analysis.OutOfScope {
		document.OutOfScope = append(document.OutOfScope, OutOfScopeInfo{
			SpecFile:  declaration.Location.File,
			Line:      declaration.Location.Line,
			Dimension: declaration.Dimension,
			Reason:    declaration.Reason,
		})
	}
	for _, finding := range outcome.Analysis.Findings {
		document.Findings = append(document.Findings, findingInfo(finding))
	}
	if outcome.Analysis.Proofs != nil {
		document.Extensions[ProofsExtension] = proofsExtension(*outcome.Analysis.Proofs)
	}
	return document
}

// ProofsExtension is the key of the extension that records the consumed proofs.
const ProofsExtension = "proofs"

func proofsExtension(proofs proof.Report) json.RawMessage {
	encoded, err := json.Marshal(proofs)
	if err != nil {
		panic(err)
	}
	return encoded
}

// Summarize counts requirements in focus by coverage, tests, and findings by severity.
func Summarize(analysis domain.Analysis) Summary {
	summary := Summary{Tests: TestSummary(analysis.Tests)}
	counters := map[domain.Coverage]*int{
		domain.CoverageCovered:     &summary.Requirements.Covered,
		domain.CoverageFailed:      &summary.Requirements.Failed,
		domain.CoverageSkipped:     &summary.Requirements.Skipped,
		domain.CoverageNotRun:      &summary.Requirements.NotRun,
		domain.CoverageUncovered:   &summary.Requirements.Uncovered,
		domain.CoverageNonTestable: &summary.Requirements.NonTestable,
	}
	for _, row := range analysis.Rows {
		summary.Requirements.Known++
		if row.InFocus {
			summary.Requirements.InFocus++
			*counters[row.Coverage]++
		}
	}
	for _, finding := range analysis.Findings {
		switch finding.Severity {
		case domain.SeverityError:
			summary.Findings.Error++
		case domain.SeverityWarning:
			summary.Findings.Warning++
		case domain.SeverityInfo:
			summary.Findings.Info++
		}
		if finding.Blocking {
			summary.Findings.Blocking++
		}
	}
	return summary
}

func requirementInfo(row domain.Row) RequirementInfo {
	requirement := row.Requirement
	info := RequirementInfo{
		ID:              requirement.ID,
		Revision:        requirement.Revision,
		Ref:             requirement.Ref().String(),
		Title:           requirement.Title,
		Statement:       requirement.Statement,
		StatementDigest: requirement.StatementDigest(),
		Status:          requirement.Status,
		Justification:   requirement.Justification,
		Location:        LocationInfo(requirement.Location),
		Examples:        exampleInfos(row.Examples),
		InFocus:         row.InFocus,
		Coverage:        row.Coverage,
		Tests:           []TestInfo{},
		Extensions:      Extensions{},
	}
	for _, test := range row.Tests {
		info.Tests = append(info.Tests, TestInfo{
			Name:        test.Name,
			ClassName:   test.ClassName,
			Suite:       test.Suite,
			Outcome:     test.Outcome,
			ResultsFile: test.ResultsFile,
			Source:      locationInfo(test.Source),
		})
	}
	return info
}

func exampleInfos(examples []domain.Example) []ExampleInfo {
	infos := []ExampleInfo{}
	for _, example := range examples {
		args := make(map[string]LiteralInfo, len(example.Args))
		for _, argument := range example.Args {
			args[argument.Name] = LiteralInfo(argument.Value)
		}
		infos = append(infos, ExampleInfo{
			Index:     example.Index,
			Operation: example.Operation,
			Args:      args,
			Expect:    OutcomeInfo{Kind: example.Expect.Kind, Type: example.Expect.Value.Type, Value: example.Expect.Value.Value},
		})
	}
	return infos
}

func findingInfo(finding domain.Finding) FindingInfo {
	info := FindingInfo{
		Category:      finding.Category,
		Severity:      finding.Severity,
		Blocking:      finding.Blocking,
		Message:       finding.Message,
		RequirementID: finding.RequirementID,
		Module:        finding.Module,
		Location:      LocationInfo(finding.Location),
	}
	if finding.Test != nil {
		info.Test = &TestRefInfo{Name: finding.Test.Name, ClassName: finding.Test.ClassName, ResultsFile: finding.Test.ResultsFile}
	}
	return info
}

func locationInfo(location *domain.Location) *LocationInfo {
	if location == nil {
		return nil
	}
	info := LocationInfo(*location)
	return &info
}

func orEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
