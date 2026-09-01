#!/usr/bin/env bash
set -euo pipefail

evidence=${1:?usage: integration.sh EVIDENCE_DIRECTORY}
receipt="$evidence/attenuation-receipt.json"
graph="$evidence/capability-graph.json"
violations="$evidence/violation.ndjson"
report="$evidence/attenuation-report.md"

test -s "$receipt"
test -s "$graph"
test -e "$violations"
test -s "$report"
test "$(find "$evidence" -type f | wc -l | tr -d '[:space:]')" = 4

subject_graph=$(jq -r '[.source_sha,.contract_sha,.toolchain,.runner] | @tsv' "$graph")
subject_receipt=$(jq -r '[.subject.source_sha,.subject.contract_sha,.subject.toolchain,.subject.runner] | @tsv' "$receipt")
test "$subject_graph" = "$subject_receipt"
jq -e '.scenarios | length == 8 and all(.[]; (.edge_id != "" and .from_stage != "" and .to_stage != "" and (.declared_capabilities | type == "array") and (.observed_capabilities | type == "array") and (.counts.declared | type == "number") and (.counts.observed | type == "number")))' "$receipt" >/dev/null
jq -e 'all(.scenarios[]; (.state == "CLOSED" or .state == "UNKNOWN" or .state == "REFUTED")) and (.precedence == ["REFUTED","UNKNOWN","CLOSED"])' "$graph" >/dev/null
jq -e 'all(.[]; .unknown == null or (.unknown as $u | ([$u.stage,$u.step,$u.reason,$u.unknown_class,$u.next_operation] | all(.[]; type == "string" and length > 0)) and ($u.blocked_by | length) > 0))' <(jq -s '.' "$violations") >/dev/null
printf 'integration: graph, receipt, violations, and report agree\n'
