#!/usr/bin/env bash
set -euo pipefail

evidence=${1:?usage: semantic-audit.sh EVIDENCE_DIRECTORY}
receipt="$evidence/attenuation-receipt.json"
graph="$evidence/capability-graph.json"

jq -e '
  .schema == "gooo/meta-capability-attenuator/attenuation-receipt/v1" and
  .precedence == ["REFUTED","UNKNOWN","CLOSED"] and
  .summary.total == 8 and
  (.scenarios | map(.id)) == [
    "case-01-read-only-reflection-attenuation",
    "case-02-caller-owned-output-generation",
    "case-03-capability-removal",
    "case-04-deterministic-replay",
    "case-05-undeclared-repository-write",
    "case-06-cross-stage-network-amplification",
    "case-07-missing-capability-origin",
    "case-08-dynamic-call-target"
  ] and
  ([.scenarios[] | select(.state == "REFUTED")] | length) == 2 and
  ([.scenarios[] | select(.state == "UNKNOWN")] | length) == 2 and
  ([.scenarios[] | select(.state == "CLOSED")] | length) == 4 and
  ([.scenarios[] | select(.state == "UNKNOWN") | .unknown as $u | ([$u.stage,$u.step,$u.reason,$u.unknown_class,$u.next_operation] | all(.[]; type == "string" and length > 0)) and ($u.blocked_by | length) > 0] | all) and
  .tests == {total:8,selected:8,executed:8,reused:0,failed:0,unknown:0} and
  .local_execution == {go_test:0,go_build:0,go_vet:0,conformance:0,integration:0} and
  ([.metrics.measurements | .[]] | all(. | type == "number" and . > 0)) and
  (.authority | to_entries | all(.[]; .value == 0)) and
  (.metrics.inventory.root_readme_excluded == true)
' "$receipt" >/dev/null

jq -e '
  .schema == "gooo/meta-capability-attenuator/capability-graph/v1" and
  .precedence == ["REFUTED","UNKNOWN","CLOSED"] and
  (.scenarios | length) == 8 and
  ([.scenarios[] | select(.state == "REFUTED" and (.amplified_capabilities | length > 0))] | length) == 2 and
  ([.scenarios[] | select(.state == "UNKNOWN" and .unknown != null)] | length) == 2
' "$graph" >/dev/null

printf 'semantic audit: precedence, lattice outcomes, origin frontier, and authority boundary verified\n'
