package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-meta-capability-attenuator/internal/attenuator"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "evaluate" {
		fmt.Fprintln(os.Stderr, "usage: gooo-meta-capability-attenuator evaluate --source PATH --contract PATH --fixtures PATH --out PATH [flags]")
		os.Exit(2)
	}

	flags := flag.NewFlagSet("evaluate", flag.ExitOnError)
	source := flags.String("source", "", "authoritative .gooo source")
	contract := flags.String("contract", "", "semantic contract JSON")
	fixtures := flags.String("fixtures", "", "semantic IR fixture JSON")
	output := flags.String("out", "", "caller-owned output directory")
	sourceRoot := flags.String("source-root", ".", "repository root for inventory")
	toolchain := flags.String("toolchain", "go1.27.0", "declared evaluator toolchain")
	runner := flags.String("runner", "github-actions-ubuntu-latest", "declared validation runner")
	ciWallMS := flags.Int("ci-wall-ms", 0, "CI wall time in milliseconds")
	ciPeakRSSKiB := flags.Int("ci-peak-rss-kib", 0, "CI peak RSS in KiB")
	compileWallMS := flags.Int("compile-wall-ms", 0, "compile step wall time in milliseconds")
	compilePeakRSSKiB := flags.Int("compile-peak-rss-kib", 0, "compile step peak RSS in KiB")
	buildWallMS := flags.Int("build-wall-ms", 0, "build step wall time in milliseconds")
	buildPeakRSSKiB := flags.Int("build-peak-rss-kib", 0, "build step peak RSS in KiB")
	testWallMS := flags.Int("test-wall-ms", 0, "test step wall time in milliseconds")
	testPeakRSSKiB := flags.Int("test-peak-rss-kib", 0, "test step peak RSS in KiB")
	conformanceWallMS := flags.Int("conformance-wall-ms", 0, "conformance step wall time in milliseconds")
	conformancePeakRSSKiB := flags.Int("conformance-peak-rss-kib", 0, "conformance step peak RSS in KiB")
	integrationWallMS := flags.Int("integration-wall-ms", 0, "integration step wall time in milliseconds")
	integrationPeakRSSKiB := flags.Int("integration-peak-rss-kib", 0, "integration step peak RSS in KiB")
	if err := flags.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}

	err := attenuator.Evaluate(attenuator.EvaluateOptions{
		Source:                *source,
		Contract:              *contract,
		Fixtures:              *fixtures,
		Output:                *output,
		SourceRoot:            *sourceRoot,
		Toolchain:             *toolchain,
		Runner:                *runner,
		CIWallMS:              *ciWallMS,
		CIPeakRSSKiB:          *ciPeakRSSKiB,
		CompileWallMS:         *compileWallMS,
		CompilePeakRSSKiB:     *compilePeakRSSKiB,
		BuildWallMS:           *buildWallMS,
		BuildPeakRSSKiB:       *buildPeakRSSKiB,
		TestWallMS:            *testWallMS,
		TestPeakRSSKiB:        *testPeakRSSKiB,
		ConformanceWallMS:     *conformanceWallMS,
		ConformancePeakRSSKiB: *conformancePeakRSSKiB,
		IntegrationWallMS:     *integrationWallMS,
		IntegrationPeakRSSKiB: *integrationPeakRSSKiB,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
