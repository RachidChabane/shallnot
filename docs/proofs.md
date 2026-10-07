# Proofs

Traceability proves that a test cited by a requirement ran and passed. It does
not prove that the test checks the requirement, or that it would fail if the
behaviour were wrong. Independent modules produce proofs of those two facts
(alignment and discriminating power) and a third, advisory one (completeness).
`shallnot` consumes the proofs in the final gate: it decodes them, checks that
they are current, applies one policy and composes the result with traceability.
It runs no module, calls no model and opens no network connection.

The gate reads proofs only when `shallnot.yaml` has a [`proofs` block](configuration.md#the-proofs-block).
Without the block nothing changes: no proof is read and `extensions` stays empty.

## The verified set

`proofs.verified_dir` (default `.spec-proof/verified`) holds one `manifest.json`
and the statement files it lists. The signature verifier writes them; `shallnot`
reads only this directory. Every statement is an in-toto Statement v1 whose
predicate is one of three types:

| Kind | Predicate type | Evidence | Subject |
|---|---|---|---|
| `alignment` | `https://rachid-chabane.com/attestations/alignment/v1` | judged | the criterion `ID~REV` and its statement digest, for one test |
| `discriminating-power` | `https://rachid-chabane.com/attestations/discriminating-power/v1` | executed kills, judged validity | the criterion `ID~REV` and its statement digest |
| `completeness` | `https://rachid-chabane.com/attestations/completeness/v1` | advisory | a spec file path and the SHA-256 of the file |

A statement of another predicate type is ignored. A member the format does not
define is ignored. A statement that breaks an invariant of the format is
`malformed` and counts as missing.

### Manifest

| Key | Meaning |
|---|---|
| `verifierVersion` | Version of the verifier that wrote the set. |
| `trust` | `signed` when the verifier checked signatures against its trust policy; `local` when it accepted unsigned local statements. |
| `trustedRootDigest` | SHA-256 of the trusted root the signatures were checked against. Required when `trust` is `signed`. |
| `policyDigest` | SHA-256 of the trust policy. Required when `trust` is `signed`. |
| `statements` | The statement files: `path`, relative to the directory with `/` separators, and `digest`, the SHA-256 of the file's bytes. Strictly ascending by `path`. |
| `pending` | The proofs a module is still producing: `module`, `criterion` as `ID~REV` and `reason`. Strictly ascending by module, criterion and reason. |

A manifest that is missing, unreadable or breaks an invariant makes every proof
missing; an unreadable or invalid one is also reported as `malformed_proof`. A
statement whose bytes differ from the digest the manifest lists is malformed.

### Pending reasons

| Reason | Meaning |
|---|---|
| `awaiting_host` | The host harness has requests of the module to answer. |
| `awaiting_proof` | Another proof the module consumes is not there yet. The gate also reports it for an alignment proof that is absent while the discriminating-power proof of the same criterion is pending. |
| `awaiting_host_judgment` | CI produced a survivor that needs a judgment of the host. |
| `host_unresponsive` | The host did not answer after every emission. |

## Classification

Each statement is `valid`, `stale`, `malformed` or `ignored`; a required kind
with no valid statement for a criterion is `missing` (or `pending`). A statement
is stale when:

- its subject digest or revision differs from the requirement's current statement, or no spec declares the requirement; for a completeness proof, the spec file changed;
- a `code_tree` or `test_source` input names a file that is absent, or whose bytes no longer have the SHA-256 the descriptor records;
- a `code_tree` input names a file outside the `proofs.code` entries, or a `test_source` input a file outside the `tests` roots, or either names an excluded path;
- a test file that holds a test bound to the criterion by a source tag is not among the `test_source` inputs of a discriminating-power proof, or the file that holds the proof's own test is not among those of an alignment proof;
- the `spec` input of a proof about a criterion is not named `ID~REV` of that criterion, or its digest is not the current statement digest of the criterion;
- an input of the role `proof` names a proof, by kind and by the SHA-256 of its file, that is not valid in the verified set;
- `proofs.trust` is `signed` and the statement was produced locally, or the manifest's `trust` is not `signed`.

### Input descriptors

`code_tree` and `test_source` inputs are one descriptor per file: `name` is the
path relative to the configuration file's directory with `/` separators, and
`digest` is the SHA-256 of the file's exact bytes. The module that wrote the proof
lists the files the criterion depends on, so `shallnot` reads exactly those files
and no other: a change to a file that no descriptor names leaves the proof valid.
`proofs.code` entries and `tests` roots are files, directories or globs; `exclude`
and, by default, dependency and build directories are left out. A `spec` input
of a completeness proof is the spec file itself and is covered by its subject
digest.

## Trust levels

| Level | Admits |
|---|---|
| `signed` | Statements produced by CI, in a set whose manifest the signature verifier wrote. |
| `local` | Also unsigned local statements. They are feedback for the agent, never proof: the report labels them `local`. |

## Policy

A criterion passes only if traceability and every kind in `proofs.require`
pass. A required kind with no valid proof is a `stale_proof` finding that names
the module and the command that regenerates it, never a pass. It blocks for
`discriminating-power`, whose evidence is executed, and escalates for
`alignment`, whose evidence is judged. A criterion that traceability did not find
covered is not asked for proofs.

An escalating finding fails the verdict when `proofs.escalation` is `error`
(the default) and is only reported when it is `warning`. A severity set in
`severities` for the category wins. A judged signal never blocks alone.

### Claim power

τ is `proofs.thresholds.validity`. The effects below apply to the mutants of a
discriminating-power proof; "corroboration" is the executed evidence a judged
claim lists in `corroboratedBy`.

| Claim | Effect | Executed corroboration required | Otherwise |
|---|---|---|---|
| tests fail on the reference code, or an example of the criterion fails on it | block (`correct_code_fails`, `example_failed`) | the claim itself | none |
| VALID survivor that fails an example of the criterion (`example_failed`) | block (`surviving_valid_mutant`) | the example | none |
| VALID survivor, confidence at least τ | block | `f4_witness`, `builds`, `other_criteria_pass` | escalate (`escalated_mutant`) |
| VALID killed | pass credit | `example_failed`; or, by basis, see below | no credit; `no_killed_valid_mutant` |
| EQUIVALENT survivor | excluded | `no_diff_found` | escalate |
| NOT_VIOLATING, off-target or unattributed survivor | excluded | `other_example_failed`; or a judgment alone under the `judged` basis | escalate |
| UNDETERMINED survivor | escalate | none | none |
| no mutant judged VALID | escalate (`no_valid_mutant`) | none | none |
| test that killed a sentinel | escalate (`mutation_sensitive_test`), no kill credit, alignment capped at UNDETERMINED | none | none |
| alignment ALIGNED | clears the test | confidence at least `thresholds.alignment`, isolation at the floor, `specific_matrix` | escalate (`misaligned_test`) |
| alignment MISALIGNED or UNDETERMINED | escalate (`misaligned_test`) | none | none |
| completeness question | reported (`completeness_question`) | none | none |

### Pass bases

`proofs.signed_pass_basis` says what a signed pass may rest on. Under
`trust: local` the basis is `judged`.

| Basis | Pass credit for a killed VALID mutant, and exclusion of a NOT_VIOLATING survivor |
|---|---|
| `executed-only` | Only on executed evidence: an example of the criterion that fails on the mutant, or of another criterion. A judged validity alone gives none. |
| `corroborated` | Also to a judged claim at confidence τ that lists `specific_kill`, `builds`, `other_criteria_pass` and `f4_witness` for a kill; a survivor judged NOT_VIOLATING without an example escalates. |
| `judged` | Also to a judged claim at confidence τ, with no further corroboration. At most `max_host_exclusions` survivors are excluded on a judgment alone. |

### Isolation levels

A judged claim answered below `proofs.min_isolation` earns no pass credit and no
exclusion; it keeps its block power. The levels, strongest first:

| Level | Meaning |
|---|---|
| `restricted-attested` | A restricted, blind executor whose transcript and capability were attested. |
| `restricted` | A restricted, blind executor. The default floor. |
| `restricted-unverified` | A restricted executor whose restriction was not verified. |
| `unrestricted` | An executor that holds the tools of the main agent. |
| `inline` | The main agent answered. |

A floor weaker than `restricted` is flagged in the report
(`isolationFloorLowered`).

## What an agent does

| Finding | What to do |
|---|---|
| `stale_proof` | Run the command the message names, or the `spec-proof:prove-all` skill. Never edit a proof file. |
| `awaiting_host`, `awaiting_proof`, `awaiting_host_judgment` | Run the fulfil skill the message names (`<module>-fulfil`, or `spec-proof:prove-all`). Never answer the module's requests yourself. |
| `host_unresponsive` | Report it to the user. Do not loop. |

An end-of-turn hook counts blocks whose only blocking findings are these
pending reasons apart from the other blocks: at most `MaxAttempts + MaxHostWaits`
consecutive blocks (3 + 3) reach the agent before the hook lets the turn end.
