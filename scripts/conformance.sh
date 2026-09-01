#!/usr/bin/env bash
set -euo pipefail

bin=${1:?usage: conformance.sh PATH_TO_BINARY [METRICS_DIRECTORY]}
repo_root=${GITHUB_WORKSPACE:-$(pwd)}
metrics_dir=${2:-}
work=$(mktemp -d "${RUNNER_TEMP:-/tmp}/gooo-meta-capability-attenuator.XXXXXX")
trap 'rm -rf "$work"' EXIT

metric_value() {
  local phase=$1
  local field=$2
  if [ -z "$metrics_dir" ] || [ ! -f "$metrics_dir/$phase.metrics" ]; then
    echo 0
    return
  fi
  value=$(awk -F= -v key="$field" '$1 == key {print $2}' "$metrics_dir/$phase.metrics")
  echo "${value:-0}"
}

run_evaluation() {
  local out=$1
  local wall_ms=$2
  local peak_rss_kib=$3
  "$bin" evaluate \
    --source "$repo_root/.gooo/capability-attenuator.gooo" \
    --contract "$repo_root/contracts/attenuation-v1.json" \
    --fixtures "$repo_root/fixtures/semantic-ir-v1.json" \
    --source-root "$repo_root" \
    --out "$out" \
    --toolchain go1.27.0 \
    --runner github-actions-ubuntu-latest \
    --ci-wall-ms "$wall_ms" \
    --ci-peak-rss-kib "$peak_rss_kib" \
    --compile-wall-ms "$(metric_value compile wall_ms)" \
    --compile-peak-rss-kib "$(metric_value compile peak_rss_kib)" \
    --build-wall-ms "$(metric_value build wall_ms)" \
    --build-peak-rss-kib "$(metric_value build peak_rss_kib)" \
    --test-wall-ms "$(metric_value test wall_ms)" \
    --test-peak-rss-kib "$(metric_value test peak_rss_kib)" \
    --conformance-wall-ms "$wall_ms" \
    --conformance-peak-rss-kib "$peak_rss_kib" \
    --integration-wall-ms "$(metric_value integration wall_ms)" \
    --integration-peak-rss-kib "$(metric_value integration peak_rss_kib)"
}

before=$(git -C "$repo_root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
start=$(date +%s%N)
/usr/bin/time -f '%M' -o "$work/probe-rss" "$bin" evaluate \
  --source "$repo_root/.gooo/capability-attenuator.gooo" \
  --contract "$repo_root/contracts/attenuation-v1.json" \
  --fixtures "$repo_root/fixtures/semantic-ir-v1.json" \
  --source-root "$repo_root" \
  --out "$work/probe" \
  --toolchain go1.27.0 \
  --runner github-actions-ubuntu-latest \
  --ci-wall-ms 0 \
  --ci-peak-rss-kib 0
end=$(date +%s%N)
wall_ms=$(( (end - start) / 1000000 ))
peak_rss_kib=$(tr -d '[:space:]' < "$work/probe-rss")
if [ "$wall_ms" -lt 1 ]; then wall_ms=1; fi
if [ "$peak_rss_kib" -lt 1 ]; then peak_rss_kib=1; fi
rm -rf "$work/probe"

run_evaluation "$work/first" "$wall_ms" "$peak_rss_kib"
run_evaluation "$work/second" "$wall_ms" "$peak_rss_kib"
after=$(git -C "$repo_root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before" = "$after"

test "$(find "$work/first" -type f | wc -l | tr -d '[:space:]')" = 4
test "$(wc -l < "$work/first/violation.ndjson" | tr -d '[:space:]')" = 4

jq -e '
  .schema == "gooo/meta-capability-attenuator/attenuation-receipt/v1" and
  .fixed_scenario_count == 8 and
  .precedence == ["REFUTED", "UNKNOWN", "CLOSED"] and
  .summary == {total:8,closed:4,unknown:2,refuted:2} and
  .improvement == {total:8,closed:4,unknown:4} and
  .tests == {total:8,selected:8,executed:8,reused:0,failed:0,unknown:0} and
  .local_execution == {go_test:0,go_build:0,go_vet:0,conformance:0,integration:0} and
  .artifacts.count == 4 and
  .artifacts.files == ["capability-graph.json","attenuation-receipt.json","violation.ndjson","attenuation-report.md"] and
  .authority == {repository_writes:0,source_mutations:0,commit_authority:0,push_authority:0,merge_authority:0,release_mutation:0,local_test_executions:0} and
  .metrics.capability_kinds == 5 and
  .metrics.stage_edges == 8 and
  .metrics.measurements.compile_wall_ms > 0 and
  .metrics.measurements.compile_peak_rss_kib > 0 and
  .metrics.measurements.build_wall_ms > 0 and
  .metrics.measurements.build_peak_rss_kib > 0 and
  .metrics.measurements.test_wall_ms > 0 and
  .metrics.measurements.test_peak_rss_kib > 0 and
  .metrics.measurements.conformance_wall_ms > 0 and
  .metrics.measurements.conformance_peak_rss_kib > 0 and
  .metrics.measurements.integration_wall_ms >= 0 and
  .metrics.measurements.integration_peak_rss_kib >= 0 and
  .metrics.inventory.root_readme_excluded == true and
  .metrics.inventory.files > 0 and
  .metrics.inventory.directories > 0 and
  .metrics.inventory.go_files > 0 and
  .metrics.inventory.gooo_files > 0 and
  .metrics.inventory.go_lines > 0 and
  .metrics.inventory.gooo_lines > 0 and
  ([.scenarios[] | select(.id == "case-01-read-only-reflection-attenuation" and .state == "CLOSED" and .counts.amplified == 0 and .counts.attenuated == 1)] | length) == 1 and
  ([.scenarios[] | select(.id == "case-02-caller-owned-output-generation" and .state == "CLOSED" and .counts.preserved == 1)] | length) == 1 and
  ([.scenarios[] | select(.id == "case-03-capability-removal" and .state == "CLOSED" and .counts.attenuated == 1)] | length) == 1 and
  ([.scenarios[] | select(.id == "case-04-deterministic-replay" and .state == "CLOSED" and .replay_equal == true)] | length) == 1 and
  ([.scenarios[] | select(.id == "case-05-undeclared-repository-write" and .state == "REFUTED" and .reason == "UNDECLARED_REPOSITORY_WRITE" and (.amplified_capabilities | index("write:repository") != null))] | length) == 1 and
  ([.scenarios[] | select(.id == "case-06-cross-stage-network-amplification" and .state == "REFUTED" and .reason == "CROSS_STAGE_NETWORK_AMPLIFICATION" and (.amplified_capabilities | index("network:outbound") != null))] | length) == 1 and
  ([.scenarios[] | select(.id == "case-07-missing-capability-origin" and .state == "UNKNOWN" and .reason == "DIRECT_MISSING") | .unknown | .stage != "" and .step != "" and .reason != "" and .unknown_class != "" and .next_operation != "" and (.blocked_by | length) > 0] | length) == 1 and
  ([.scenarios[] | select(.id == "case-08-dynamic-call-target" and .state == "UNKNOWN" and .reason == "DYNAMIC_CALL_TARGET_AMBIGUOUS") | .unknown | .stage != "" and .step != "" and .reason != "" and .unknown_class != "" and .next_operation != "" and (.blocked_by | length) > 0] | length) == 1 and
  ([.scenarios[] | select(.improvement.state == "CLOSED" and .improvement.before != null and .improvement.after != null and .improvement.identity_match == true)] | length) == 4 and
  ([.scenarios[] | select(.improvement.state == "UNKNOWN" and .improvement.unknown != null)] | length) == 4
' "$work/first/attenuation-receipt.json"

jq -e '.schema == "gooo/meta-capability-attenuator/capability-graph/v1" and .capability_kinds == ["read:source","read:reflection","write:caller-owned-output","network:outbound","write:repository"] and (.stage_edges | length) == 8 and (.scenarios | length) == 8' "$work/first/capability-graph.json" >/dev/null
jq -e 'all(.[]; .state == "REFUTED" or .state == "UNKNOWN")' <(jq -s '.' "$work/first/violation.ndjson") >/dev/null

for artifact in capability-graph.json attenuation-receipt.json violation.ndjson attenuation-report.md; do
  cmp -s "$work/first/$artifact" "$work/second/$artifact"
done

evidence_root="${RUNNER_TEMP:-/tmp}/gooo-meta-capability-attenuator-evidence"
rm -rf "$evidence_root"
cp -R "$work/first" "$evidence_root"
printf 'conformance: fixed scenarios=%s CLOSED=%s UNKNOWN=%s REFUTED=%s wall_ms=%s peak_rss_kib=%s\n' \
  "$(jq -r '.summary.total' "$evidence_root/attenuation-receipt.json")" \
  "$(jq -r '.summary.closed' "$evidence_root/attenuation-receipt.json")" \
  "$(jq -r '.summary.unknown' "$evidence_root/attenuation-receipt.json")" \
  "$(jq -r '.summary.refuted' "$evidence_root/attenuation-receipt.json")" \
  "$wall_ms" "$peak_rss_kib"
