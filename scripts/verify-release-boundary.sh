#!/usr/bin/env bash
set -euo pipefail

mode=${1:?usage: verify-release-boundary.sh tag TAG COMMIT | release TAG COMMIT}
repo=${GITHUB_REPOSITORY:-kimjooyoon/gooo-meta-capability-attenuator}
api_version='X-GitHub-Api-Version: 2026-03-10'

case "$mode" in
  tag)
    tag=${2:?tag mode requires TAG}
    expected_commit=${3:?tag mode requires COMMIT}
    ref=$(gh api --header "$api_version" "repos/$repo/git/ref/tags/$tag")
    test "$(jq -er '.object.type' <<<"$ref")" = tag
    tag_object=$(jq -er '.object.sha' <<<"$ref")
    tag_data=$(gh api --header "$api_version" "repos/$repo/git/tags/$tag_object")
    test "$(jq -er '.object.sha' <<<"$tag_data")" = "$expected_commit"
    test "$(jq -er '.tag' <<<"$tag_data")" = "$tag"
    ;;
  release)
    tag=${2:?release mode requires TAG}
    expected_commit=${3:?release mode requires COMMIT}
    response=$(gh api --header "$api_version" "repos/$repo/releases/tags/$tag")
    test "$(jq -er '.draft' <<<"$response")" = false
    test "$(jq -er '.immutable' <<<"$response")" = true
    test "$(jq -er '.assets | length' <<<"$response")" -gt 0
    jq -e 'all(.assets[]; (.digest | type == "string" and startswith("sha256:")))' <<<"$response" >/dev/null
    ref=$(gh api --header "$api_version" "repos/$repo/git/ref/tags/$tag")
    test "$(jq -er '.object.type' <<<"$ref")" = tag
    tag_object=$(jq -er '.object.sha' <<<"$ref")
    tag_data=$(gh api --header "$api_version" "repos/$repo/git/tags/$tag_object")
    test "$(jq -er '.object.sha' <<<"$tag_data")" = "$expected_commit"
    ;;
  *)
    echo "unknown verification mode: $mode" >&2
    exit 2
    ;;
esac

printf 'release boundary: %s closed\n' "$mode"
