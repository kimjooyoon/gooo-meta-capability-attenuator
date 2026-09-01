#!/usr/bin/env bash
set -euo pipefail

receipt=${1:?usage: patch-measurement.sh RECEIPT PHASE WALL_MS PEAK_RSS_KIB}
phase=${2:?measurement phase is required}
wall_ms=${3:?wall_ms is required}
peak_rss_kib=${4:?peak_rss_kib is required}
case "$phase" in
  compile|build|test|conformance|integration) ;;
  *) echo "unsupported measurement phase: $phase" >&2; exit 64 ;;
esac
case "$wall_ms" in ''|*[!0-9]*) echo 'wall_ms must be an integer' >&2; exit 64 ;; esac
case "$peak_rss_kib" in ''|*[!0-9]*) echo 'peak_rss_kib must be an integer' >&2; exit 64 ;; esac

temporary="${receipt}.tmp.$$"
if [ "$phase" = conformance ]; then
  jq --argjson wall_ms "$wall_ms" --argjson peak_rss_kib "$peak_rss_kib" \
    '.metrics.measurements.conformance_wall_ms = $wall_ms |
     .metrics.measurements.conformance_peak_rss_kib = $peak_rss_kib |
     .metrics.ci.wall_ms = $wall_ms |
     .metrics.ci.peak_rss_kib = $peak_rss_kib' "$receipt" > "$temporary"
else
  jq --arg phase "$phase" --argjson wall_ms "$wall_ms" --argjson peak_rss_kib "$peak_rss_kib" \
    '.metrics.measurements[($phase + "_wall_ms")] = $wall_ms |
     .metrics.measurements[($phase + "_peak_rss_kib")] = $peak_rss_kib' "$receipt" > "$temporary"
fi
mv "$temporary" "$receipt"
