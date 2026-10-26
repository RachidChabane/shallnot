package proofgate_test

import (
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/analysis/proofgate"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

func TestComposition(t *testing.T) {
	t.Run("passes a covered requirement whose required module passes [verifies SN-113~1]", func(t *testing.T) {
		result := evaluate(config(), evidence(dp()))
		wantCategories(t, result)
		if outcome(t, result) != proofgate.OutcomePass || result.Report.Criteria[0].Modules[proof.KindDiscriminatingPower].Outcome != proof.OutcomePass {
			t.Fatalf("result %+v", result.Report.Criteria)
		}
	})

	t.Run("never passes a required module with no proof and names it [verifies SN-113~1]", func(t *testing.T) {
		for name, ev := range map[string]proof.Evidence{"no manifest": {ManifestPath: "verified/manifest.json", Derived: derived()}, "empty set": evidence()} {
			result := evaluate(config(), ev)
			wantCategories(t, result, "stale_proof")
			finding := result.Findings[0]
			if outcome(t, result) != proofgate.OutcomeBlock || finding.Effect != proofgate.EffectBlock || finding.Module != "musthold-dp" ||
				!strings.Contains(finding.Message, "discriminating-power") || !strings.Contains(finding.Message, "musthold-dp prove --changed") {
				t.Errorf("%s: %+v", name, finding)
			}
			if got := result.Report.Criteria[0]; len(got.Absent) != 1 || got.Absent[0] != proof.KindDiscriminatingPower || got.Modules[proof.KindDiscriminatingPower].Outcome != proof.OutcomeAbsent {
				t.Errorf("%s: report %+v", name, got)
			}
		}
	})

	t.Run("blocks an absent discriminating-power proof and escalates an absent alignment proof [verifies SN-113~1]", func(t *testing.T) {
		result := evaluate(config(requireBoth), evidence())
		wantCategories(t, result, "stale_proof", "stale_proof")
		effects := map[string]proofgate.Effect{}
		for _, finding := range result.Findings {
			effects[finding.Module] = finding.Effect
		}
		if effects["musthold-dp"] != proofgate.EffectBlock || effects["musthold-alignment"] != proofgate.EffectEscalate {
			t.Fatalf("effects %v", effects)
		}
	})

	t.Run("names the absent module of each criterion [verifies SN-113~1]", func(t *testing.T) {
		result := evaluate(config(requireBoth), evidence(dp()))
		wantCategories(t, result, "stale_proof")
		if got := result.Report.Criteria[0].Absent; len(got) != 1 || got[0] != proof.KindAlignment {
			t.Fatalf("absent = %v", got)
		}
		if !strings.Contains(result.Findings[0].Message, testName) {
			t.Errorf("message names no test: %s", result.Findings[0].Message)
		}
	})

	t.Run("raises nothing for an absent module that is not required [verifies SN-113~1]", func(t *testing.T) {
		wantCategories(t, evaluate(config(func(c *proof.Config) { c.Require = nil }), evidence()))
	})

	t.Run("asks nothing of a requirement that traceability did not find covered [verifies SN-113~1]", func(t *testing.T) {
		result := proofgate.Evaluate(proofgate.Input{Config: config(), Known: known(), Evidence: evidence()})
		wantCategories(t, result)
		if len(result.Report.Criteria) != 0 {
			t.Fatalf("criteria %+v", result.Report.Criteria)
		}
	})

	t.Run("applies the findings of a proof that is present although it is not required [verifies SN-113~1]", func(t *testing.T) {
		failing := dp()
		failing.DiscriminatingPower.CorrectCodePasses = prooftest.Executed(false)
		result := evaluate(config(func(c *proof.Config) { c.Require = nil }), evidence(failing))
		wantCategories(t, result, "correct_code_fails")
	})
}

func TestExecutedFailures(t *testing.T) {
	t.Run("blocks when the tests fail on the reference code [verifies SN-114~1]", func(t *testing.T) {
		statement := dp()
		statement.DiscriminatingPower.CorrectCodePasses = prooftest.Executed(false)
		result := evaluate(config(), evidence(statement))
		wantCategories(t, result, "correct_code_fails")
		if outcome(t, result) != proofgate.OutcomeBlock {
			t.Fatal("not blocked")
		}
	})

	t.Run("blocks when an example of the criterion fails on the reference code [verifies SN-114~1]", func(t *testing.T) {
		statement := dp()
		statement.DiscriminatingPower.Examples = []proof.ExampleRun{
			{Index: 0, Result: prooftest.Executed(proof.ExamplePassed)}, {Index: 1, Result: prooftest.Executed(proof.ExampleFailed)},
		}
		result := evaluate(config(), evidence(statement))
		wantCategories(t, result, "example_failed")
		if !strings.Contains(result.Findings[0].Message, "example 1") || outcome(t, result) != proofgate.OutcomeBlock {
			t.Fatalf("got %v", messages(result))
		}
	})
}

func TestPendingProofs(t *testing.T) {
	pending := func(module string, reason proof.PendingReason) proof.Evidence {
		ev := evidence()
		ev.Manifest.Pending = []proof.Pending{{Module: module, Criterion: "PARK-2~1", Reason: reason}}
		return ev
	}

	for reason, category := range map[proof.PendingReason]string{
		proof.PendingAwaitingHost: "awaiting_host", proof.PendingAwaitingProof: "awaiting_proof",
		proof.PendingAwaitingHostJudgment: "awaiting_host_judgment", proof.PendingHostUnresponsive: "host_unresponsive",
	} {
		t.Run("reports "+category+" instead of stale_proof [verifies SN-123~1]", func(t *testing.T) {
			result := evaluate(config(), pending("musthold-dp", reason))
			wantCategories(t, result, category)
			finding := result.Findings[0]
			if finding.Module != "musthold-dp" || result.Report.Criteria[0].Modules[proof.KindDiscriminatingPower].State != proof.StatePending {
				t.Fatalf("got %+v", finding)
			}
		})
	}

	t.Run("names the module and its fulfil skill and forbids answering the requests oneself [verifies SN-123~1]", func(t *testing.T) {
		message := evaluate(config(), pending("musthold-dp", proof.PendingAwaitingHost)).Findings[0].Message
		for _, expected := range []string{"musthold-dp", "musthold-dp-fulfil", "never answer the requests yourself"} {
			if !strings.Contains(message, expected) {
				t.Errorf("no %q in %q", expected, message)
			}
		}
	})

	t.Run("falls back to the umbrella skill for a module it does not know [verifies SN-123~1]", func(t *testing.T) {
		if got := proof.FulfilSkill("another-module"); got != "musthold:prove-all" {
			t.Fatalf("skill %s", got)
		}
	})

	t.Run("tells the agent to report to the user and not to loop when the host is unresponsive [verifies SN-123~1]", func(t *testing.T) {
		message := evaluate(config(), pending("musthold-dp", proof.PendingHostUnresponsive)).Findings[0].Message
		if !strings.Contains(message, "report this to the user") && !strings.Contains(message, "Report this to the user") || !strings.Contains(message, "do not loop") {
			t.Fatalf("message %q", message)
		}
	})

	t.Run("escalates the judgment and the unresponsive host but blocks on executed evidence still awaited [verifies SN-123~1]", func(t *testing.T) {
		wantEffect := map[proof.PendingReason]proofgate.Effect{
			proof.PendingAwaitingHost: proofgate.EffectBlock, proof.PendingAwaitingProof: proofgate.EffectBlock,
			proof.PendingAwaitingHostJudgment: proofgate.EffectEscalate, proof.PendingHostUnresponsive: proofgate.EffectEscalate,
		}
		for reason, effect := range wantEffect {
			if got := evaluate(config(), pending("musthold-dp", reason)).Findings[0].Effect; got != effect {
				t.Errorf("%s: effect %v, want %v", reason, got, effect)
			}
		}
	})

	t.Run("reports awaiting_proof for a required module whose criterion has another module pending [verifies SN-123~1]", func(t *testing.T) {
		ev := pending("musthold-dp", proof.PendingAwaitingHost)
		result := evaluate(config(requireBoth), ev)
		wantCategories(t, result, "awaiting_host", "awaiting_proof")
		for _, finding := range result.Findings {
			if finding.Module != "musthold-dp" || finding.Category.String() == "stale_proof" {
				t.Errorf("finding %+v", finding)
			}
		}
	})

	t.Run("ignores a pending entry of a module that is not required [verifies SN-123~1]", func(t *testing.T) {
		ev := pending("musthold-alignment", proof.PendingAwaitingHost)
		result := evaluate(config(), ev)
		wantCategories(t, result, "stale_proof")
	})
}

func TestMalformedProofs(t *testing.T) {
	t.Run("reports an unusable manifest and counts every proof as missing [verifies SN-124~1]", func(t *testing.T) {
		ev := evidence(dp())
		ev.ManifestProblem = "trust \"maybe\" is not one of signed, local"
		result := evaluate(config(), ev)
		wantCategories(t, result, "malformed_proof", "stale_proof")
	})

	t.Run("reports a malformed statement and counts it as missing [verifies SN-124~1]", func(t *testing.T) {
		ev := evidence()
		ev.Statements = []proof.LoadedStatement{{Path: "verified/bad.json", Digest: prooftest.Digest("bad"), Problem: "unsorted at predicate.tests"}}
		result := evaluate(config(), ev)
		wantCategories(t, result, "malformed_proof", "stale_proof")
		if result.Report.Consumed[0].State != proof.StateMalformed {
			t.Fatalf("consumed %+v", result.Report.Consumed)
		}
	})

	t.Run("reports several valid proofs of one kind for one criterion and counts none [verifies SN-124~1]", func(t *testing.T) {
		second := dp()
		second.DiscriminatingPower.Module.Version = "0.2.0"
		wantCategories(t, evaluate(config(), evidence(dp(), prooftest.Seal(second))), "malformed_proof", "stale_proof")
	})

	t.Run("ignores a statement of another predicate type [verifies SN-124~1]", func(t *testing.T) {
		ev := evidence(dp())
		ev.Statements = append(ev.Statements, proof.LoadedStatement{Path: "verified/future.json", Digest: prooftest.Digest("future"), Ignored: true})
		result := evaluate(config(), ev)
		wantCategories(t, result)
		if result.Report.Consumed[1].State != proof.StateIgnored {
			t.Fatalf("consumed %+v", result.Report.Consumed)
		}
	})
}

func TestCompleteness(t *testing.T) {
	t.Run("reports each question at the spec and never blocks [verifies SN-121~1]", func(t *testing.T) {
		completeness := prooftest.Completeness(specPath, specFileDigest, completenessInputs()...)
		completeness.Completeness.Questions = []proof.Question{
			{Dimension: "boundaries", Question: "what happens at exactly 15 minutes?", RequirementIDs: []string{"PARK-2"}},
			{Dimension: "time", Question: "which time zone applies?"},
		}
		result := evaluate(config(), evidence(dp(), prooftest.Seal(completeness)))
		wantCategories(t, result, "completeness_question", "completeness_question")
		for _, finding := range result.Findings {
			if finding.Effect != proofgate.EffectInform || finding.Location.File != specPath {
				t.Errorf("finding %+v", finding)
			}
		}
		if outcome(t, result) != proofgate.OutcomePass {
			t.Fatal("a question changed the outcome")
		}
	})
}
