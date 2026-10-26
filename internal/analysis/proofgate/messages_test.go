package proofgate_test

import (
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

func TestStaleProofMessages(t *testing.T) {
	for _, setting := range []struct {
		name    string
		trust   proof.Trust
		dir     string
		publish string
	}{
		{"local default", proof.TrustLocal, ".musthold/verified", "musthold-verify --accept-local .musthold/cache --out .musthold/verified"},
		{"signed custom", proof.TrustSigned, "./proofs/published", "musthold-verify --bundles <ledger export dir> --out ./proofs/published"},
	} {
		for _, state := range []string{"absent manifest", "empty manifest", "stale proof"} {
			t.Run(setting.name+" "+state+" [verifies SN-113~2]", func(t *testing.T) {
				cfg := config()
				cfg.Trust, cfg.VerifiedDir = setting.trust, setting.dir
				ev := evidence()
				detail := "none was produced"
				switch state {
				case "absent manifest":
					ev.ManifestPresent = false
					detail = "no verified proof set: " + setting.dir + "/manifest.json is absent"
				case "stale proof":
					ev = evidence(dp())
					ev.Derived.Files[codeFile] = prooftest.Digest("changed code")
					detail = "the code_tree file src/parkfee.py changed since the proof was produced"
				}
				result := evaluate(cfg, ev)
				wantCategories(t, result, "stale_proof")
				want := "requirement PARK-2~1 has no valid discriminating-power proof (" + detail +
					"): a missing or stale proof never passes. Run `musthold-dp prove --changed`, then publish with `" +
					setting.publish + "`, or run the musthold:prove-all skill"
				if got := result.Findings[0].Message; got != want {
					t.Fatalf("message = %q, want %q", got, want)
				}
			})
		}
	}
}
