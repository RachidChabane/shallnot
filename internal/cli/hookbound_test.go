package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"pgregory.net/rapid"

	"github.com/RachidChabane/shallnot/internal/adapters/agenthook"
)

type gateState uint8

const (
	gatePasses gateState = iota
	gateBlocksOnEvidence
	gateBlocksOnPending
)

const maxSequence = 60

// TestHostWaitBound drives the decision of the hook with any sequence of gate results, for a harness that leaves the
// count of blocks to the hook and for one that counts them itself.
func TestHostWaitBound(t *testing.T) {
	verdicts := map[gateState]hookVerdict{
		gatePasses:           {},
		gateBlocksOnEvidence: {blocked: true},
		gateBlocksOnPending:  {blocked: true, pendingOnly: true},
	}
	longest := agenthook.MaxAttempts + agenthook.MaxHostWaits

	for name, countsBlocks := range map[string]bool{"the hook counts": false, "the harness counts": true} {
		t.Run("any interleaving of waits and other blocks ends within MaxAttempts + MaxHostWaits blocks when "+name+" [verifies SN-127~1]", func(t *testing.T) {
			base, iteration := t.TempDir(), 0
			rapid.Check(t, func(t *rapid.T) {
				iteration++
				directory := filepath.Join(base, strconv.Itoa(iteration))
				if err := os.Mkdir(directory, 0o700); err != nil {
					t.Fatal(err)
				}
				attempts := agenthook.Attempts{Directory: directory}
				sequence := rapid.SliceOfN(rapid.SampledFrom([]gateState{gatePasses, gateBlocksOnEvidence, gateBlocksOnPending}), 1, maxSequence).Draw(t, "gate results")
				run, evidenceBlocks, waitBlocks := 0, 0, 0
				for step, state := range sequence {
					event := agenthook.Event{SessionID: "session", CountsBlocks: countsBlocks, PriorBlocks: run}
					decision, _ := decide(event, attempts, func() hookVerdict { return verdicts[state] })
					switch decision {
					case agenthook.DecisionBlock:
						run++
						if state == gateBlocksOnPending {
							waitBlocks++
						} else {
							evidenceBlocks++
						}
					default:
						run, evidenceBlocks, waitBlocks = 0, 0, 0
					}
					if run > longest || evidenceBlocks > agenthook.MaxAttempts || waitBlocks > agenthook.MaxHostWaits {
						t.Fatalf("step %d of %v: %d consecutive blocks (%d on evidence, %d waits)", step, sequence, run, evidenceBlocks, waitBlocks)
					}
				}
			})
		})
	}

	t.Run("a block that is not a wait never resets the count of waits, nor the other way round [verifies SN-127~1]", func(t *testing.T) {
		attempts := agenthook.Attempts{Directory: t.TempDir()}
		event := agenthook.Event{SessionID: "s"}
		for _, state := range []gateState{gateBlocksOnPending, gateBlocksOnEvidence, gateBlocksOnPending} {
			decide(event, attempts, func() hookVerdict { return verdicts[state] })
		}
		if attempts.Count("s") != 1 || attempts.CountHostWaits("s") != 2 {
			t.Fatalf("attempts %d, waits %d", attempts.Count("s"), attempts.CountHostWaits("s"))
		}
		decide(event, attempts, func() hookVerdict { return verdicts[gatePasses] })
		if attempts.Count("s") != 0 || attempts.CountHostWaits("s") != 0 {
			t.Fatal("a pass left a counter behind")
		}
	})
}
