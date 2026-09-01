# Gooo Meta Capability Attenuator

This repository verifies that a self-improving Gooo metaprogram cannot amplify
the capabilities of generated code. It compares the declared capability set
of every staged semantic IR edge with the generated-stage observed set, then
requires a proof of capability origin. The evaluator handles inert fixtures;
it never invokes reflection, network, repository, or output capabilities.

The semantic authority is
[`capability-attenuator.gooo`](.gooo/capability-attenuator.gooo). Go is only
the evaluator, generator, and runtime for the staged IR. The fixed contract is
[`attenuation-v1.json`](contracts/attenuation-v1.json), and the eight fixed
fixtures are in [`semantic-ir-v1.json`](fixtures/semantic-ir-v1.json).

| scenario | expected state | exact purpose |
|---|---|---|
| read-only reflection attenuation | `CLOSED` | generated code attenuates reflection read |
| caller-owned output generation | `CLOSED` | output remains caller-owned |
| capability removal | `CLOSED` | a declared capability is removed |
| deterministic replay | `CLOSED` | the same IR produces the same receipt |
| undeclared repository write | `REFUTED` | repository write is amplification |
| cross-stage network amplification | `REFUTED` | network origin crosses the stage edge |
| missing capability origin | `UNKNOWN` | direct origin proof is missing |
| dynamic call target | `UNKNOWN` | origin target is ambiguous |

Resolution precedence is exactly `REFUTED > UNKNOWN > CLOSED`. An UNKNOWN
record always retains `stage`, `step`, `reason`, `unknown_class`,
`next_operation`, and `blocked_by`; it is never promoted to CLOSED. Improvement
is CLOSED only for the same scenario, source, contract, Go/toolchain, and
runner when both before and after exact integer vectors are present.

The evaluator writes only these caller-owned artifacts:

- `capability-graph.json`
- `attenuation-receipt.json`
- `violation.ndjson`
- `attenuation-report.md`

CI is the validation authority and runs Go 1.27 formatting, vet, tests, build,
semantic conformance, integration, and semantic audit. Each compile, build,
test, conformance, and integration step records its own wall time and peak RSS
as ten integer fields in the receipt. Fixed-scenario test accounting is kept
separate from semantic decisions: `tests` is always total=8, selected=8,
executed=8, reused=0, failed=0, unknown=0, while the scenario summary retains
four CLOSED, two UNKNOWN, and two REFUTED decisions. Local execution fields
for Go test/build/vet/conformance/integration are explicit zeroes.

The release workflow
creates a draft with the standard `GITHUB_TOKEN`, uploads the CI evidence,
publishes once, and verifies the public release API reports `immutable=true`
and a `sha256:` digest for every asset. Existing tags and releases are never
modified or deleted.
