# Repository contract

This repository evaluates staged Gooo semantic IR fixtures. The authoritative
meaning of metaprogram stages and capability origins is `.gooo/`; Go is only
the evaluator, generator, and runtime for the fixture format.

The evaluator never invokes a capability, writes the source repository, runs a
repository command, commits, pushes, merges, or releases. It writes only the
caller-owned output directory passed by `--out`. Repository, commit, merge,
release, and local-test authority are recorded as exact zeroes in the receipt.

GitHub Actions is the validation authority. Do not run local `go test`, `go
build`, `go vet`, conformance, or integration commands when producing release
evidence. The Go/toolchain declaration is 1.27.

Source inventory excludes root `README.md`, `.git`, caller-owned temp/output,
cache, vendor, and toolchain internals. Do not modify sibling repositories
under `/Users/alice/meta-go`.
