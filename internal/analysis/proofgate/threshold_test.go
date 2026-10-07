package proofgate_test

import (
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

const (
	hostVotes  proof.ConfidenceSource = "host-votes(2)"
	systemOne  proof.ConfidenceSource = "system-one:model-a@0123456789abcdef"
	middling                          = 0.75
	strictHost                        = 0.8
	strictFull                        = 0.9
	laxExactly                        = 0.5
	decidedVia                        = "claude-sonnet-5-5"
)

func decision(source proof.ConfidenceSource) *proof.Decision {
	return &proof.Decision{Basis: proof.BasisJudges, Deciders: []string{decidedVia}, ConfidenceSource: source, RawConfidence: middling, Threshold: 0.7}
}

func withSourceThresholds(validity, alignment map[proof.ConfidenceSource]float64) func(*proof.Config) {
	return func(c *proof.Config) {
		c.Thresholds.Validity.BySource = validity
		c.Thresholds.Alignment.BySource = alignment
	}
}

func alignedBy(source proof.ConfidenceSource) proof.Statement {
	statement := aligned(testName, verdict(proof.VerdictAligned, middling, proof.CorroborationSpecificMatrix))
	statement.Alignment.Extensions.Decision = decision(source)
	return statement
}

func clearsAlignment(statement proof.Statement, cfg proof.Config) bool {
	return len(evaluate(cfg, evidence(dp(), statement)).Findings) == 0
}

func TestSourceSpecificThresholds(t *testing.T) {
	alignmentThresholds := map[proof.ConfidenceSource]float64{"host-votes": strictHost, "system-one": strictFull, "host-votes(2)": laxExactly}

	t.Run("applies the default threshold to a verdict whose source has no entry [verifies SN-138~1]", func(t *testing.T) {
		cfg := config(requireBoth, withSourceThresholds(nil, map[proof.ConfidenceSource]float64{"system-one": strictFull}))
		if cleared := clearsAlignment(alignedBy(hostVotes), cfg); !cleared {
			t.Fatal("0.75 did not clear the default 0.70")
		}
	})

	t.Run("applies the entry of the kind of the source and names it in the finding [verifies SN-138~1]", func(t *testing.T) {
		cfg := config(requireBoth, withSourceThresholds(nil, map[proof.ConfidenceSource]float64{"host-votes": strictHost}))
		result := evaluate(cfg, evidence(dp(), alignedBy(hostVotes)))
		wantCategories(t, result, "misaligned_test")
		if message := result.Findings[0].Message; !strings.Contains(message, "below 0.80") {
			t.Fatalf("message %q", message)
		}
	})

	t.Run("lets the entry of the exact source win over the entry of its kind [verifies SN-138~1]", func(t *testing.T) {
		cfg := config(requireBoth, withSourceThresholds(nil, alignmentThresholds))
		if cleared := clearsAlignment(alignedBy(hostVotes), cfg); !cleared {
			t.Fatal("the exact host-votes(2) entry 0.50 did not win over the kind entry 0.80")
		}
		wantCategories(t, evaluate(cfg, evidence(dp(), alignedBy(systemOne))), "misaligned_test")
	})

	t.Run("applies the default to a verdict that records no decision [verifies SN-138~1]", func(t *testing.T) {
		cfg := config(requireBoth, withSourceThresholds(nil, map[proof.ConfidenceSource]float64{"host-votes": strictFull, "system-one": strictFull, "execution": strictFull}))
		statement := aligned(testName, verdict(proof.VerdictAligned, middling, proof.CorroborationSpecificMatrix))
		if cleared := clearsAlignment(statement, cfg); !cleared {
			t.Fatal("a verdict without a decision did not take the default")
		}
	})

	t.Run("applies the validity entry of the decision recorded on the mutant before the one of the predicate [verifies SN-138~1]", func(t *testing.T) {
		build := func(mutantSource, predicateSource proof.ConfidenceSource) proof.Statement {
			judged := survivor("s", valid(middling, witness...))
			judged.Extensions = &proof.MutantExtensions{Decision: decision(mutantSource)}
			statement := withCredit(judged)
			statement.DiscriminatingPower.Extensions.Decision = decision(predicateSource)
			return statement
		}
		cfg := config(withSourceThresholds(map[proof.ConfidenceSource]float64{"host-votes": strictHost}, nil))
		expect(t, cfg, build(systemOne, hostVotes), "surviving_valid_mutant")
		result := expect(t, cfg, build(hostVotes, systemOne), "escalated_mutant")
		if message := result.Findings[0].Message; !strings.Contains(message, "below 0.80") {
			t.Fatalf("message %q", message)
		}
	})

	t.Run("falls back to the decision of the predicate for a mutant without one [verifies SN-138~1]", func(t *testing.T) {
		statement := withCredit(survivor("s", valid(middling, witness...)))
		statement.DiscriminatingPower.Extensions.Decision = decision(hostVotes)
		cfg := config(withSourceThresholds(map[proof.ConfidenceSource]float64{"host-votes": strictHost}, nil))
		expect(t, cfg, statement, "escalated_mutant")
	})

	t.Run("applies the source threshold to a kill that would earn credit and to a judged exclusion [verifies SN-138~1]", func(t *testing.T) {
		kill := killed("k", valid(middling, creditKinds...))
		kill.Extensions = &proof.MutantExtensions{Decision: decision(hostVotes)}
		statement := dp(kill)
		lenient := config()
		wantCategories(t, evaluate(lenient, evidence(statement)))
		strict := config(withSourceThresholds(map[proof.ConfidenceSource]float64{"host-votes": strictHost}, nil))
		wantCategories(t, evaluate(strict, evidence(statement)), "no_killed_valid_mutant")

		excluded := survivor("e", judged(proof.ValidityNotViolating, middling))
		excluded.Extensions = &proof.MutantExtensions{Decision: decision(hostVotes)}
		wantCategories(t, evaluate(config(basis(proof.BasisJudged)), evidence(withCredit(excluded))))
		wantCategories(t, evaluate(config(basis(proof.BasisJudged), withSourceThresholds(map[proof.ConfidenceSource]float64{"host-votes": strictHost}, nil)), evidence(withCredit(excluded))), "escalated_mutant")
	})

	t.Run("reports the defaults and the entries by source in the policy of the report [verifies SN-138~1]", func(t *testing.T) {
		cfg := config(withSourceThresholds(map[proof.ConfidenceSource]float64{"host-votes": strictHost}, nil))
		policy := evaluate(cfg, evidence(dp())).Report.Policy
		if policy.ValidityThreshold != proof.DefaultValidityThreshold || policy.ValidityThresholdBySource["host-votes"] != strictHost || policy.AlignmentThresholdBySource != nil {
			t.Fatalf("policy %+v", policy)
		}
	})
}
