package proof_test

import (
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

func TestPublishCommand(t *testing.T) {
	for _, tc := range []struct {
		name  string
		trust proof.Trust
		dir   string
		want  string
	}{
		{"local default", proof.TrustLocal, ".musthold/verified", "musthold-verify --accept-local .musthold/cache --out .musthold/verified"},
		{"signed default", proof.TrustSigned, ".musthold/verified", "musthold-verify --bundles <ledger export dir> --out .musthold/verified"},
		{"local custom", proof.TrustLocal, "./proofs/published/", "musthold-verify --accept-local .musthold/cache --out ./proofs/published/"},
		{"signed custom", proof.TrustSigned, "./proofs/published/", "musthold-verify --bundles <ledger export dir> --out ./proofs/published/"},
	} {
		t.Run(tc.name+" [verifies SN-113~2]", func(t *testing.T) {
			config := proof.DefaultConfig()
			config.Trust, config.VerifiedDir = tc.trust, tc.dir
			if got := config.PublishCommand(); got != tc.want {
				t.Fatalf("publish command = %q, want %q", got, tc.want)
			}
		})
	}
}
