package proofgate_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/RachidChabane/shallnot/internal/analysis/proofgate"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

var update = flag.Bool("update", false, "rewrite the gate-outcome goldens")

const goldenPath = "testdata/outcomes.golden.json"

// goldenOutcome is what the gate gave one row of the claim-power table under one trust level and basis.
type goldenOutcome struct {
	Row        string   `json:"row"`
	Trust      string   `json:"trust"`
	Basis      string   `json:"basis"`
	Outcome    string   `json:"outcome"`
	Categories []string `json:"categories"`
}

type claimPowerRow struct {
	name      string
	statement func() proof.Statement
	// alignment is the verdict of the alignment proof that accompanies the row, if any.
	alignment *proof.Claim[proof.Verdict]
}

func claimPowerRows() []claimPowerRow {
	pool := func(mutants ...proof.Mutant) func() proof.Statement {
		return func() proof.Statement { return withCredit(mutants...) }
	}
	alone := func(mutants ...proof.Mutant) func() proof.Statement {
		return func() proof.Statement { return dp(mutants...) }
	}
	aligned := func(value proof.Verdict, kinds ...proof.CorroborationKind) *proof.Claim[proof.Verdict] {
		claim := prooftest.Judged(value, highConfidence, kinds...)
		return &claim
	}
	return []claimPowerRow{
		{name: "tests fail on the reference code", statement: func() proof.Statement {
			statement := withCredit()
			statement.DiscriminatingPower.CorrectCodePasses = prooftest.Executed(false)
			return statement
		}},
		{name: "an example fails on the reference code", statement: func() proof.Statement {
			statement := withCredit()
			statement.DiscriminatingPower.Examples = []proof.ExampleRun{{Index: 0, Result: prooftest.Executed(proof.ExampleFailed)}}
			return statement
		}},
		{name: "VALID survivor failing an example of the criterion", statement: pool(survivor("s", valid(lowConfidence, proof.CorroborationExampleFailed)))},
		{name: "VALID survivor with a full witness", statement: pool(survivor("s", valid(highConfidence, witness...)))},
		{name: "VALID survivor without a witness", statement: pool(survivor("s", valid(highConfidence, proof.CorroborationBuilds)))},
		{name: "VALID killed with the evidence of every row", statement: alone(killed("k", valid(highConfidence, creditKinds...)))},
		{name: "VALID killed with a judgment alone", statement: alone(killed("k", valid(highConfidence, proof.CorroborationBuilds)))},
		{name: "VALID killed failing an example of the criterion", statement: alone(killed("k", valid(lowConfidence, proof.CorroborationExampleFailed)))},
		{name: "EQUIVALENT survivor with no difference found", statement: pool(survivor("s", judged(proof.ValidityEquivalent, highConfidence, proof.CorroborationNoDiffFound)))},
		{name: "EQUIVALENT survivor on a judgment alone", statement: pool(survivor("s", judged(proof.ValidityEquivalent, highConfidence)))},
		{name: "NOT_VIOLATING survivor failing an example of another criterion", statement: pool(survivor("s", judged(proof.ValidityNotViolating, highConfidence, proof.CorroborationOtherExampleFailed)))},
		{name: "NOT_VIOLATING survivor on a judgment alone", statement: pool(survivor("s", judged(proof.ValidityNotViolating, highConfidence)))},
		{name: "UNDETERMINED survivor", statement: pool(survivor("s", judged(proof.ValidityUndetermined, highConfidence)))},
		{name: "no mutant judged VALID", statement: func() proof.Statement {
			statement := dp()
			statement.DiscriminatingPower.Mutants = nil
			return prooftest.Seal(statement)
		}},
		{name: "test that killed a sentinel", statement: func() proof.Statement {
			sentinel := prooftest.Mutant("sentinel", judged(proof.ValidityEquivalent, 1), map[string]proof.Outcome{testName: proof.OutcomeKilled})
			sentinel.Sentinel = true
			return dp(killed("k", valid(highConfidence, proof.CorroborationExampleFailed)), sentinel)
		}},
		{name: "alignment ALIGNED with matrix evidence", statement: pool(), alignment: aligned(proof.VerdictAligned, proof.CorroborationSpecificMatrix)},
		{name: "alignment ALIGNED without matrix evidence", statement: pool(), alignment: aligned(proof.VerdictAligned)},
		{name: "alignment MISALIGNED", statement: pool(), alignment: aligned(proof.VerdictMisaligned, proof.CorroborationSpecificMatrix)},
		{name: "alignment UNDETERMINED", statement: pool(), alignment: aligned(proof.VerdictUndetermined)},
	}
}

func settings() []struct {
	trust proof.Trust
	basis proof.SignedPassBasis
} {
	return []struct {
		trust proof.Trust
		basis proof.SignedPassBasis
	}{
		{proof.TrustLocal, proof.BasisCorroborated},
		{proof.TrustSigned, proof.BasisExecutedOnly},
		{proof.TrustSigned, proof.BasisCorroborated},
		{proof.TrustSigned, proof.BasisJudged},
	}
}

var outcomeNames = map[proofgate.Outcome]string{proofgate.OutcomePass: "pass", proofgate.OutcomeEscalate: "escalate", proofgate.OutcomeBlock: "block"}

func computeGoldens() []goldenOutcome {
	var goldens []goldenOutcome
	for _, row := range claimPowerRows() {
		for _, setting := range settings() {
			cfg := config(func(c *proof.Config) { c.Trust, c.SignedPassBasis = setting.trust, setting.basis })
			statements := []proof.Statement{row.statement()}
			if row.alignment != nil {
				cfg.Require = []proof.Kind{proof.KindAlignment, proof.KindDiscriminatingPower}
				statements = append(statements, prooftestAlignment(*row.alignment))
			}
			result := evaluate(cfg, localize(cfg, statements...))
			goldens = append(goldens, goldenOutcome{
				Row: row.name, Trust: string(setting.trust), Basis: string(setting.basis),
				Outcome: outcomeNames[result.Outcomes[criterion().Ref]], Categories: categories(result),
			})
		}
	}
	return goldens
}

func prooftestAlignment(claim proof.Claim[proof.Verdict]) proof.Statement {
	return aligned(testName, claim)
}

func TestGateOutcomeGoldens(t *testing.T) {
	encoded, err := json.MarshalIndent(computeGoldens(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, '\n')
	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, encoded, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("gives each row of the claim-power table the committed outcome under each trust level and basis [verifies SN-131~1]", func(t *testing.T) {
		committed, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("%v (run go test ./internal/analysis/proofgate -run TestGateOutcomeGoldens -update)", err)
		}
		if !bytes.Equal(committed, encoded) {
			t.Fatalf("the gate outcomes differ from %s (run with -update and review the diff):\n%s", goldenPath, encoded)
		}
	})

	t.Run("covers every row of the table under every setting [verifies SN-131~1]", func(t *testing.T) {
		var committed []goldenOutcome
		raw, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &committed); err != nil {
			t.Fatal(err)
		}
		if want := len(claimPowerRows()) * len(settings()); len(committed) != want {
			t.Fatalf("%d goldens, want %d", len(committed), want)
		}
	})
}
