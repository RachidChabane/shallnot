# Relation to other tools

Requirement tracing is an established field. `shallnot` exists because of one
combination that the existing tools do not offer: a tag bound to a specific
test and joined with that test's actual result, in a dependency-free binary
with a JSON report designed for programs.

| | Spec format | How coverage is declared | Uses test results | Runtime | Licence |
|---|---|---|---|---|---|
| **shallnot** | Markdown or YAML, IDs of any shape, revisions | Tag surfaced by the test runner into JUnit XML | Yes: a requirement is covered only by a test that ran and passed | None (static binary) | Apache-2.0 |
| [OpenFastTrace](https://github.com/itsallcode/openfasttrace) | Markdown/RST items `type~name~revision` with `Needs`/`Covers` | Comment tags `[utest->req~login~1]` found by a language-agnostic source scan | No: a tag in a file counts whether or not any test runs or passes | JVM | GPL-3.0 |
| [Doorstop](https://github.com/doorstop-dev/doorstop) | One YAML file per item, in a VCS tree | Explicit links between items | No | Python | LGPL-3.0 |
| [StrictDoc](https://github.com/strictdoc-project/strictdoc) | Its own `.sdoc` document format | `@relation` markers in source files | Partly: JUnit XML import into its documentation model | Python | Apache-2.0 |
| [sphinx-needs](https://github.com/useblocks/sphinx-needs) | Sphinx directives | Links between need objects | Through the separate sphinx-test-reports extension, as linkable objects in a Sphinx build | Python + Sphinx | MIT |

**Why this is not a contribution to OpenFastTrace.** OpenFastTrace is the
closest relative, and `shallnot` borrows two of its ideas deliberately: the
revision carried in every reference, so that a change of meaning invalidates
the coverage that cites the old revision; and tag discovery by a
language-agnostic scan, which costs no code per framework. The differences are
structural rather than incremental:

- OpenFastTrace traces *static* links across artifact types (requirement →
  design → implementation → test). Whether a test runs, passes, or is skipped
  is outside its model. `shallnot`'s model is the join between a tag and a
  test outcome; without that join it has nothing to say.
- OpenFastTrace requires a JVM and integrates through Maven and Gradle.
  `shallnot` is one static binary, usable alike in a Python, JavaScript or JVM
  project, and in a container that holds nothing else.
- `shallnot`'s primary output is a versioned JSON report consumed by programs
  and agents, with exit codes that separate a verdict from a tool failure.
- `shallnot` has an explicit, justified non-testable status, and accepts
  requirement IDs in whatever shape the project already uses rather than
  `type~name~revision`.

OpenFastTrace remains the better tool for multi-level tracing across
artifact types, which `shallnot` does not attempt. StrictDoc and sphinx-needs
are documentation systems: adopting them means authoring requirements in
their formats and building their documents, which suits a documentation-led
process and not a gate that reads the Markdown plan an agent wrote an hour
ago. [ReqToCode](https://arxiv.org/abs/2603.13999) embeds requirements as
language-native code elements validated at build time; it gives stronger
structural guarantees inside one language, where `shallnot` stays outside the
language and works from what every test runner can already emit.

## Why the citation sits on the test

Requiring an agent to cite a requirement ID on every line of generated code
makes invented requirements mechanically detectable, but measurably reduces
the consistency of the agent's output
([arXiv 2606.30689](https://arxiv.org/abs/2606.30689)). `shallnot` moves the
citation from the code line to the test, where its placement is unambiguous.
A cited ID that no spec declares is an `orphan_tag` finding.
