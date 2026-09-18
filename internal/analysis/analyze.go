// Package analysis joins requirements, source tags and test results into a
// coverage matrix and classified findings. It performs no I/O.
package analysis

import (
	"fmt"
	"sort"

	"github.com/RachidChabane/shallnot/internal/domain"
)

// FocusedRequirement is a known requirement and whether this run demands its coverage.
type FocusedRequirement struct {
	Requirement domain.Requirement
	InFocus     bool
}

// Input is everything a run has collected.
type Input struct {
	Requirements []FocusedRequirement
	SpecProblems []domain.SpecProblem
	SourceTags   []domain.SourceTag
	TestCases    []domain.TestCase
	Policy       domain.Policy
}

type analyzer struct {
	input    Input
	known    map[domain.RequirementID]FocusedRequirement
	bound    map[domain.RequirementID][]domain.BoundTest
	findings []domain.Finding
}

// Analyze computes the matrix, the findings and the verdict of a run.
func Analyze(input Input) domain.Analysis {
	a := &analyzer{
		input: input,
		known: map[domain.RequirementID]FocusedRequirement{},
		bound: map[domain.RequirementID][]domain.BoundTest{},
	}
	rows := a.indexRequirements()
	a.reportSpecProblems()
	matchedSources := a.bindTestCases()
	a.bindUnmatchedSourceTags(matchedSources)
	for index := range rows {
		a.settleRow(&rows[index])
	}
	sortFindings(a.findings)
	return domain.Analysis{
		Rows:     rows,
		Findings: a.findings,
		Tests:    a.testTotals(),
		Verdict:  verdictOf(a.findings),
	}
}

func verdictOf(findings []domain.Finding) domain.Verdict {
	for _, finding := range findings {
		if finding.Blocking {
			return domain.VerdictFail
		}
	}
	return domain.VerdictPass
}

func (a *analyzer) report(finding domain.Finding) {
	finding.Severity = a.input.Policy.SeverityOf(finding.Category)
	if finding.Severity == domain.SeverityOff {
		return
	}
	finding.Blocking = a.input.Policy.Blocks(finding.Severity)
	a.findings = append(a.findings, finding)
}

// indexRequirements keeps the first declaration of each ID and reports the others.
func (a *analyzer) indexRequirements() []domain.Row {
	declared := append([]FocusedRequirement(nil), a.input.Requirements...)
	sort.SliceStable(declared, func(i, j int) bool {
		return locationLess(declared[i].Requirement.Location, declared[j].Requirement.Location)
	})
	var rows []domain.Row
	for _, candidate := range declared {
		requirement := candidate.Requirement
		if first, duplicate := a.known[requirement.ID]; duplicate {
			a.report(domain.Finding{
				Category:      domain.CategoryDuplicateID,
				RequirementID: requirement.ID,
				Location:      requirement.Location,
				Message: fmt.Sprintf("requirement %s is already declared at %s",
					requirement.ID, formatLocation(first.Requirement.Location)),
			})
			continue
		}
		a.known[requirement.ID] = candidate
		rows = append(rows, domain.Row{Requirement: requirement, InFocus: candidate.InFocus})
	}
	return rows
}

func (a *analyzer) reportSpecProblems() {
	for _, problem := range a.input.SpecProblems {
		a.report(domain.Finding{
			Category: domain.CategoryMalformedRequirement,
			Location: problem.Location,
			Message:  problem.Message,
		})
	}
}

// bindTestCases binds every results test case to the requirements it validly
// cites and returns the indexes of the source tags that were seen running.
func (a *analyzer) bindTestCases() map[int]bool {
	matchedSources := map[int]bool{}
	reportedAtSource := a.reportSourceTagDefects()
	for _, testCase := range a.input.TestCases {
		identity := normalizeIdentity(testCase.Identity())
		for _, ref := range testCase.Refs {
			source, sourceIndexes := a.sourcesOf(ref, identity)
			for _, sourceIndex := range sourceIndexes {
				matchedSources[sourceIndex] = true
			}
			if !a.resolves(ref) {
				if !reportedAtSource[ref] {
					a.reportUnresolved(ref, domain.Location{File: testCase.ResultsFile}, identityOf(testCase))
				}
				continue
			}
			a.bound[ref.ID] = append(a.bound[ref.ID], domain.BoundTest{
				ResultsFile: testCase.ResultsFile,
				Suite:       testCase.Suite,
				ClassName:   testCase.ClassName,
				Name:        testCase.Name,
				Outcome:     testCase.Outcome,
				Source:      source,
			})
		}
		a.reportResultsOnlyMalformed(testCase)
		a.reportUntagged(testCase)
	}
	return matchedSources
}

// reportSourceTagDefects reports orphan, mismatched and malformed tags at
// their source line and returns the refs it reported.
func (a *analyzer) reportSourceTagDefects() map[domain.Ref]bool {
	reported := map[domain.Ref]bool{}
	for _, tag := range a.input.SourceTags {
		for _, malformed := range tag.Malformed {
			a.reportMalformed(malformed, tag.Location, nil)
		}
		for _, ref := range tag.Refs {
			if a.resolves(ref) {
				continue
			}
			reported[ref] = true
			a.reportUnresolved(ref, tag.Location, nil)
		}
	}
	return reported
}

func (a *analyzer) resolves(ref domain.Ref) bool {
	known, exists := a.known[ref.ID]
	return exists && known.Requirement.Revision == ref.Revision
}

func (a *analyzer) reportUnresolved(ref domain.Ref, location domain.Location, test *domain.TestIdentity) {
	known, exists := a.known[ref.ID]
	if !exists {
		a.report(domain.Finding{
			Category:      domain.CategoryOrphanTag,
			RequirementID: ref.ID,
			Location:      location,
			Test:          test,
			Message:       fmt.Sprintf("tag cites %s, but no known spec declares %s", ref, ref.ID),
		})
		return
	}
	a.report(domain.Finding{
		Category:      domain.CategoryRevisionMismatch,
		RequirementID: ref.ID,
		Location:      location,
		Test:          test,
		Message: fmt.Sprintf("tag cites %s, but the spec declares %s: re-verify the test against the current statement, then cite %s",
			ref, known.Requirement.Ref(), known.Requirement.Ref()),
	})
}

func (a *analyzer) reportMalformed(malformed domain.MalformedTag, location domain.Location, test *domain.TestIdentity) {
	a.report(domain.Finding{
		Category: domain.CategoryMalformedTag,
		Location: location,
		Test:     test,
		Message:  fmt.Sprintf("malformed tag %s: %s", malformed.Raw, malformed.Reason),
	})
}

// reportResultsOnlyMalformed reports a malformed tag seen in results unless the source scan already located it.
func (a *analyzer) reportResultsOnlyMalformed(testCase domain.TestCase) {
	for _, malformed := range testCase.Malformed {
		if a.sourceHasMalformed(malformed) {
			continue
		}
		a.reportMalformed(malformed, domain.Location{File: testCase.ResultsFile}, identityOf(testCase))
	}
}

func (a *analyzer) sourceHasMalformed(malformed domain.MalformedTag) bool {
	for _, tag := range a.input.SourceTags {
		for _, candidate := range tag.Malformed {
			if candidate.Reason == malformed.Reason {
				return true
			}
		}
	}
	return false
}

func (a *analyzer) reportUntagged(testCase domain.TestCase) {
	if len(testCase.Refs) > 0 || len(testCase.Malformed) > 0 {
		return
	}
	a.report(domain.Finding{
		Category: domain.CategoryUntaggedTest,
		Location: domain.Location{File: testCase.ResultsFile},
		Test:     identityOf(testCase),
		Message:  fmt.Sprintf("test %q verifies no stated requirement", displayName(testCase.ClassName, testCase.Name)),
	})
}

// sourcesOf finds the source tags a results citation comes from: those citing
// the same ref whose label occurs in the test's identity. A leaf title and an
// enclosing suite title can both apply. The most specific one locates the test.
func (a *analyzer) sourcesOf(ref domain.Ref, identity string) (*domain.Location, []int) {
	var matched []int
	best, bestLength := -1, -1
	for index, tag := range a.input.SourceTags {
		if !citesRef(tag, ref) {
			continue
		}
		label := normalizeIdentity(tag.Label)
		if !containsLabel(identity, label) {
			continue
		}
		matched = append(matched, index)
		if len(label) > bestLength {
			best, bestLength = index, len(label)
		}
	}
	if best < 0 {
		return nil, nil
	}
	location := a.input.SourceTags[best].Location
	return &location, matched
}

func citesRef(tag domain.SourceTag, ref domain.Ref) bool {
	for _, candidate := range tag.Refs {
		if candidate == ref {
			return true
		}
	}
	return false
}

// bindUnmatchedSourceTags records tagged tests that no results file mentions.
func (a *analyzer) bindUnmatchedSourceTags(matched map[int]bool) {
	for index, tag := range a.input.SourceTags {
		if matched[index] {
			continue
		}
		for _, ref := range tag.Refs {
			if !a.resolves(ref) {
				continue
			}
			location := tag.Location
			a.report(domain.Finding{
				Category:      domain.CategoryTagNotInResults,
				RequirementID: ref.ID,
				Location:      location,
				Message: fmt.Sprintf("tag citing %s appears in no results file: the test did not run, or the tag sits where the runner does not report it",
					ref),
			})
			a.bound[ref.ID] = append(a.bound[ref.ID], domain.BoundTest{
				Name:    tag.Label,
				Outcome: domain.OutcomeNotRun,
				Source:  &location,
			})
		}
	}
}

func (a *analyzer) settleRow(row *domain.Row) {
	requirement := row.Requirement
	row.Tests = a.bound[requirement.ID]
	sortBoundTests(row.Tests)
	row.Coverage = coverageOf(requirement, row.Tests)
	if requirement.Status == domain.StatusNonTestable {
		a.settleNonTestable(*row)
		return
	}
	if row.InFocus {
		a.reportCoverageGap(*row)
	}
}

func (a *analyzer) settleNonTestable(row domain.Row) {
	requirement := row.Requirement
	if requirement.Justification == "" {
		a.report(domain.Finding{
			Category:      domain.CategoryUnjustifiedNonTestable,
			RequirementID: requirement.ID,
			Location:      requirement.Location,
			Message:       fmt.Sprintf("requirement %s is marked non-testable without a justification", requirement.ID),
		})
	}
	if len(row.Tests) > 0 {
		a.report(domain.Finding{
			Category:      domain.CategoryBoundNonTestable,
			RequirementID: requirement.ID,
			Location:      requirement.Location,
			Message:       fmt.Sprintf("requirement %s is marked non-testable, yet %d test(s) cite it", requirement.ID, len(row.Tests)),
		})
	}
}

func coverageOf(requirement domain.Requirement, tests []domain.BoundTest) domain.Coverage {
	if requirement.Status == domain.StatusNonTestable {
		return domain.CoverageNonTestable
	}
	seen := map[domain.Outcome]bool{}
	for _, test := range tests {
		seen[test.Outcome] = true
	}
	switch {
	case seen[domain.OutcomePassed]:
		return domain.CoverageCovered
	case seen[domain.OutcomeFailed] || seen[domain.OutcomeErrored]:
		return domain.CoverageFailed
	case seen[domain.OutcomeSkipped]:
		return domain.CoverageSkipped
	case seen[domain.OutcomeNotRun]:
		return domain.CoverageNotRun
	default:
		return domain.CoverageUncovered
	}
}

var gapCategories = map[domain.Coverage]domain.Category{
	domain.CoverageFailed:    domain.CategoryFailedRequirement,
	domain.CoverageSkipped:   domain.CategorySkippedRequirement,
	domain.CoverageNotRun:    domain.CategoryNotRunRequirement,
	domain.CoverageUncovered: domain.CategoryUncoveredRequirement,
}

var gapMessages = map[domain.Coverage]string{
	domain.CoverageFailed:    "requirement %s is not verified: no bound test passed and at least one failed",
	domain.CoverageSkipped:   "requirement %s is not verified: its bound tests were skipped",
	domain.CoverageNotRun:    "requirement %s is not verified: its tagged tests appear in no results file (they did not run, or the tag sits where the runner does not report it)",
	domain.CoverageUncovered: "requirement %s has no bound test",
}

func (a *analyzer) reportCoverageGap(row domain.Row) {
	requirement := row.Requirement
	if category, gap := gapCategories[row.Coverage]; gap {
		a.report(domain.Finding{
			Category:      category,
			RequirementID: requirement.ID,
			Location:      requirement.Location,
			Message:       fmt.Sprintf(gapMessages[row.Coverage], requirement.Ref()),
		})
		return
	}
	for _, test := range row.Tests {
		if test.Outcome != domain.OutcomeFailed && test.Outcome != domain.OutcomeErrored {
			continue
		}
		a.report(domain.Finding{
			Category:      domain.CategoryFailingBoundTest,
			RequirementID: requirement.ID,
			Location:      locationOfTest(test),
			Test:          &domain.TestIdentity{ResultsFile: test.ResultsFile, ClassName: test.ClassName, Name: test.Name},
			Message: fmt.Sprintf("test %q bound to %s %s, although another bound test passed",
				displayName(test.ClassName, test.Name), requirement.Ref(), test.Outcome),
		})
	}
}

func locationOfTest(test domain.BoundTest) domain.Location {
	if test.Source != nil {
		return *test.Source
	}
	return domain.Location{File: test.ResultsFile}
}

func (a *analyzer) testTotals() domain.TestTotals {
	totals := domain.TestTotals{Total: len(a.input.TestCases)}
	for _, testCase := range a.input.TestCases {
		switch {
		case len(testCase.Refs) > 0:
			totals.Bound++
		case len(testCase.Malformed) == 0:
			totals.Untagged++
		}
	}
	return totals
}

func identityOf(testCase domain.TestCase) *domain.TestIdentity {
	return &domain.TestIdentity{ResultsFile: testCase.ResultsFile, ClassName: testCase.ClassName, Name: testCase.Name}
}

func displayName(className, name string) string {
	if className == "" {
		return name
	}
	return className + " › " + name
}

func formatLocation(location domain.Location) string {
	if location.Line == 0 {
		return location.File
	}
	return fmt.Sprintf("%s:%d", location.File, location.Line)
}
