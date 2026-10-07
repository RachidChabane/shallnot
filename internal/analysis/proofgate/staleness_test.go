package proofgate_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/RachidChabane/shallnot/internal/analysis/proofgate"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

const stalenessGoldenPath = "testdata/staleness.golden.json"

var treeFiles = []string{
	"src/a.py", "src/b.py", "src/pkg/c.py", "src/pkg/d.py", "src/e.py",
	"tests/a_test.py", "tests/b_test.py", "tests/pkg/c_test.py", "tests/d_test.py",
}

func roleOf(file string) proof.InputRole {
	if strings.HasPrefix(file, "src/") {
		return proof.RoleCodeTree
	}
	return proof.RoleTestSource
}

// treeEvidence is a verified set holding one proof that lists exactly the given files, over a tree that holds all of treeFiles.
func treeEvidence(listed []string, edited map[string]bool) (proof.Evidence, proofgate.Input) {
	digests := map[string]string{}
	for _, file := range treeFiles {
		digests[file] = prooftest.Digest("content of " + file)
	}
	inputs := []proof.Input{{Role: proof.RoleSpec, Name: proof.SubjectName(requirementID, revision), Digest: proof.Digest{SHA256: statementDigest}}}
	testFiles := []string{}
	for _, file := range listed {
		inputs = append(inputs, proof.Input{Role: roleOf(file), Name: file, Digest: proof.Digest{SHA256: digests[file]}})
		if roleOf(file) == proof.RoleTestSource {
			testFiles = append(testFiles, file)
		}
	}
	statement := prooftest.Seal(prooftest.DiscriminatingPower(requirementID, revision, statementDigest, inputs...))
	for file := range edited {
		digests[file] = prooftest.Digest("edited " + file)
	}
	ev := evidence(statement)
	ev.Derived = proof.DerivedInputs{SpecFiles: map[string]string{}, Files: digests}
	bound := criterion()
	bound.TestFiles = testFiles
	return ev, proofgate.Input{Config: config(), Known: known(), Criteria: []proofgate.Criterion{bound}, Scope: scope(), Evidence: ev}
}

func stateOf(input proofgate.Input) proof.State {
	return proofgate.Evaluate(input).Report.Consumed[0].State
}

func TestPerFileStalenessProperty(t *testing.T) {
	drawListed := func(t *rapid.T) []string {
		return rapid.SliceOfNDistinct(rapid.SampledFrom(treeFiles), 1, len(treeFiles), func(file string) string { return file }).Draw(t, "listed")
	}

	t.Run("editing a file outside every descriptor never makes a proof stale [verifies SN-132~1]", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			listed := drawListed(t)
			isListed := map[string]bool{}
			for _, file := range listed {
				isListed[file] = true
			}
			edited := map[string]bool{}
			for _, file := range treeFiles {
				if !isListed[file] && rapid.Bool().Draw(t, "edit "+file) {
					edited[file] = true
				}
			}
			_, input := treeEvidence(listed, edited)
			if got := stateOf(input); got != proof.StateValid {
				t.Fatalf("editing %v outside %v made the proof %s", edited, listed, got)
			}
		})
	})

	t.Run("editing a listed file always makes a proof stale [verifies SN-132~1]", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			listed := drawListed(t)
			target := rapid.SampledFrom(listed).Draw(t, "edited listed file")
			_, input := treeEvidence(listed, map[string]bool{target: true})
			if got := stateOf(input); got != proof.StateStale {
				t.Fatalf("editing %s of %v left the proof %s", target, listed, got)
			}
		})
	})
}

type stalenessGolden struct {
	Case   string `json:"case"`
	State  string `json:"state"`
	Reason string `json:"reason"`
}

func stalenessCases() []struct {
	name  string
	build func() proofgate.Input
} {
	listed := []string{"src/a.py", "tests/a_test.py"}
	with := func(change func(*proofgate.Input)) func() proofgate.Input {
		return func() proofgate.Input {
			_, input := treeEvidence(listed, nil)
			change(&input)
			return input
		}
	}
	return []struct {
		name  string
		build func() proofgate.Input
	}{
		{"current", with(func(*proofgate.Input) {})},
		{"listed code file differs", with(func(in *proofgate.Input) { in.Evidence.Derived.Files["src/a.py"] = prooftest.Digest("edited") })},
		{"listed test file absent", with(func(in *proofgate.Input) { delete(in.Evidence.Derived.Files, "tests/a_test.py") })},
		{"code file outside proofs.code", with(func(in *proofgate.Input) { in.Scope.Code = []string{"lib"} })},
		{"test file outside the tests roots", with(func(in *proofgate.Input) { in.Scope.Tests = []string{"spec"} })},
		{"bound test file not listed", with(func(in *proofgate.Input) { in.Criteria[0].TestFiles = []string{"tests/a_test.py", "tests/b_test.py"} })},
		{"alignment lists only the file of its own test", func() proofgate.Input {
			return alignmentInput([]string{"tests/a_test.py"}, map[string]string{testName: "tests/a_test.py", "test_two": "tests/b_test.py"})
		}},
		{"alignment omits the file of its own test", func() proofgate.Input {
			return alignmentInput([]string{"tests/b_test.py"}, map[string]string{testName: "tests/a_test.py", "test_two": "tests/b_test.py"})
		}},
		{"statement digest differs", with(func(in *proofgate.Input) {
			in.Known = map[domain.RequirementID]proofgate.KnownRequirement{requirementID: {Revision: revision, Digest: prooftest.Digest("new statement")}}
		})},
	}
}

// alignmentInput is a verified set holding one alignment proof of testName that lists the given test files, for a
// criterion whose tests live in the files given per test name.
func alignmentInput(listedTests []string, files map[string]string) proofgate.Input {
	inputs := []proof.Input{{Role: proof.RoleSpec, Name: proof.SubjectName(requirementID, revision), Digest: proof.Digest{SHA256: statementDigest}}}
	for _, file := range listedTests {
		inputs = append(inputs, proof.Input{Role: proof.RoleTestSource, Name: file, Digest: proof.Digest{SHA256: prooftest.Digest("content of " + file)}})
	}
	statement := prooftest.Alignment(requirementID, revision, statementDigest, testName, inputs...)
	statement.Alignment.Verdict = verdict(proof.VerdictAligned, 0.9)
	ev := evidence(prooftest.Seal(statement))
	ev.Derived.Files = map[string]string{}
	bound := criterion()
	bound.PassingTests = nil
	for name, file := range files {
		bound.PassingTests = append(bound.PassingTests, proofgate.TestName{Name: name, File: file})
		bound.TestFiles = append(bound.TestFiles, file)
		ev.Derived.Files[file] = prooftest.Digest("content of " + file)
	}
	sort.Strings(bound.TestFiles)
	return proofgate.Input{Config: config(), Known: known(), Criteria: []proofgate.Criterion{bound}, Scope: scope(), Evidence: ev}
}

func TestAlignmentTestFile(t *testing.T) {
	files := map[string]string{testName: "tests/a_test.py", "test_two": "tests/b_test.py"}

	t.Run("requires only the file that holds the proof's own test [verifies SN-134~2]", func(t *testing.T) {
		input := alignmentInput([]string{"tests/a_test.py"}, files)
		if got := proofgate.Evaluate(input).Report.Consumed[0].State; got != proof.StateValid {
			t.Fatalf("state %s", got)
		}
	})

	t.Run("classifies an alignment proof stale when the file of its own test is not listed [verifies SN-134~2]", func(t *testing.T) {
		input := alignmentInput([]string{"tests/b_test.py"}, files)
		consumed := proofgate.Evaluate(input).Report.Consumed[0]
		if consumed.State != proof.StateStale || !strings.Contains(consumed.Reason, "tests/a_test.py") {
			t.Fatalf("consumed %+v", consumed)
		}
	})

	t.Run("a listed file edit or an unlisted file edit keeps the property for alignment proofs [verifies SN-132~1]", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			input := alignmentInput([]string{"tests/a_test.py"}, files)
			if rapid.Bool().Draw(t, "edit listed") {
				input.Evidence.Derived.Files["tests/a_test.py"] = prooftest.Digest("edited")
				if got := stateOf(input); got != proof.StateStale {
					t.Fatalf("listed edit left %s", got)
				}
				return
			}
			input.Evidence.Derived.Files["tests/b_test.py"] = prooftest.Digest("edited")
			if got := stateOf(input); got != proof.StateValid {
				t.Fatalf("unlisted edit made it %s", got)
			}
		})
	})
}

func TestStalenessGoldens(t *testing.T) {
	var computed []stalenessGolden
	for _, c := range stalenessCases() {
		consumed := proofgate.Evaluate(c.build()).Report.Consumed[0]
		computed = append(computed, stalenessGolden{Case: c.name, State: string(consumed.State), Reason: consumed.Reason})
	}
	encoded, err := json.MarshalIndent(computed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, '\n')
	if *update {
		if err := os.MkdirAll(filepath.Dir(stalenessGoldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(stalenessGoldenPath, encoded, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("gives each staleness check the committed state and reason [verifies SN-132~1, SN-133~2, SN-134~2, SN-135~2]", func(t *testing.T) {
		committed, err := os.ReadFile(stalenessGoldenPath)
		if err != nil {
			t.Fatalf("%v (run go test ./internal/analysis/proofgate -run TestStalenessGoldens -update)", err)
		}
		if !bytes.Equal(committed, encoded) {
			t.Fatalf("the staleness outcomes differ from %s (run with -update and review the diff):\n%s", stalenessGoldenPath, encoded)
		}
	})
}
