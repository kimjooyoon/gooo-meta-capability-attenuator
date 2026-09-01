#!/usr/bin/env bash
set -euo pipefail

output=${1:?usage: measure-step.sh METRICS_FILE COMMAND [ARGS...]}
shift
if [ "$#" -eq 0 ]; then
  echo 'a command is required' >&2
  exit 64
fi

start=$(date +%s%N)
/usr/bin/time -f '%M' -o "${output}.rss" "$@"
end=$(date +%s%N)
wall_ms=$(( (end - start) / 1000000 ))
peak_rss_kib=$(tr -d '[:space:]' < "${output}.rss")
if [ "$wall_ms" -lt 1 ]; then wall_ms=1; fi
if [ "$peak_rss_kib" -lt 1 ]; then peak_rss_kib=1; fi
{
  echo "wall_ms=$wall_ms"
  echo "peak_rss_kib=$peak_rss_kib"
} > "$output"
