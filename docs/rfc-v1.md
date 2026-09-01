# Capability attenuation RFC v1

## Scope

The attenuator is a staged semantic IR evaluator for Gooo metaprograms. It is
not a generic security scanner and it does not execute generated code. A stage
edge is safe only when the generated-stage capability set is a subset of the
declared set and every observed capability has an origin proof.

For an edge `A -> B`, the evaluator computes exact integer counts:

- declared capabilities: `|D|`
- observed capabilities: `|O|`
- preserved capabilities: `|D ∩ O|`
- amplified capabilities: `|O - D|`
- attenuated capabilities: `|D - O|`

No score or percentage is derived from these counts. The capability lattice is
the fixed vocabulary in `contracts/attenuation-v1.json`.

## Decisions

The decision order is `REFUTED > UNKNOWN > CLOSED`. Any non-empty amplification
is REFUTED. A missing or ambiguous origin proof is UNKNOWN unless amplification
also exists, in which case REFUTED wins. CLOSED means the lattice comparison
and all required origin proofs are complete.

UNKNOWN must carry all six fields: `stage`, `step`, `reason`, `unknown_class`,
`next_operation`, and `blocked_by`. These fields are part of the durable
receipt and the NDJSON violation stream.

Improvement evidence has an independent exact identity tuple of scenario,
source digest, contract digest, toolchain, and runner. It is CLOSED only when
both before and after integer vectors are present, the identities match, and
the scenario itself is CLOSED. Missing pairs or identity mismatches remain
UNKNOWN.

## Authority boundary

`.gooo` owns stage declaration, origin binding, generated projection, lattice
comparison, counterexample preservation, and receipt emission. Go reads the
source activities and evaluates the supplied semantic IR. It has zero
repository-write, source-mutation, commit, push, merge, release, and local-test
authority. Its sole write target is the caller-owned output directory.
