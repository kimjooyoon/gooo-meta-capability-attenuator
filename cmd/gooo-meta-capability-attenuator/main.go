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
	if err := flags.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}

	err := attenuator.Evaluate(attenuator.EvaluateOptions{
		Source:       *source,
		Contract:     *contract,
		Fixtures:     *fixtures,
		Output:       *output,
		SourceRoot:   *sourceRoot,
		Toolchain:    *toolchain,
		Runner:       *runner,
		CIWallMS:     *ciWallMS,
		CIPeakRSSKiB: *ciPeakRSSKiB,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
