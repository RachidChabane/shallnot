package proofgate

import (
	"fmt"
	"strings"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

const (
	digestPrefixLength = 12
	listSeparator      = ", "
	scopeCode          = "proofs.code globs"
	scopeTests         = "tests roots"
)

func abbreviate(digest string) string {
	if len(digest) <= digestPrefixLength {
		return digest
	}
	return digest[:digestPrefixLength]
}

func staleMessage(ref string, kind proof.Kind, scope string, reasons []string, config proof.Config, manifestPresent bool) string {
	detail := "none was produced"
	if !manifestPresent {
		detail = "no verified proof set: " + strings.TrimSuffix(config.VerifiedDir, "/") + "/manifest.json is absent"
	} else if len(reasons) > 0 {
		detail = strings.Join(reasons, "; ")
	}
	return fmt.Sprintf("requirement %s has no valid %s proof%s (%s): a missing or stale proof never passes. Run `%s`, then publish with `%s`, or run the %s skill",
		ref, kind, scope, detail, kind.ProveCommand(), config.PublishCommand(), proof.UmbrellaSkill)
}

const (
	pendingTemplate      = "requirement %s: %s has requests waiting for the host harness (%s). Run the %s skill; never answer the requests yourself"
	unresponsiveTemplate = "requirement %s: the host harness did not answer the requests of %s. Report this to the user; do not loop"
)

func pendingMessage(reason proof.PendingReason, ref, module string) string {
	if reason == proof.PendingHostUnresponsive {
		return fmt.Sprintf(unresponsiveTemplate, ref, module)
	}
	return fmt.Sprintf(pendingTemplate, ref, module, reason, proof.FulfilSkill(module))
}

func forTest(test string) string { return fmt.Sprintf(" for test %q", test) }
