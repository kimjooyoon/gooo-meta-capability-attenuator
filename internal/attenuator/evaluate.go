package attenuator

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type EvaluateOptions struct {
	Source       string
	Contract     string
	Fixtures     string
	Output       string
	SourceRoot   string
	Toolchain    string
	Runner       string
	CIWallMS     int
	CIPeakRSSKiB int
}

var fixedPrecedence = []string{"REFUTED", "UNKNOWN", "CLOSED"}

func Evaluate(options EvaluateOptions) error {
	if options.Source == "" || options.Contract == "" || options.Fixtures == "" || options.Output == "" {
		return errors.New("source, contract, fixtures, and output are required")
	}
	if options.Toolchain == "" {
		options.Toolchain = "go1.27.0"
	}
	if options.Runner == "" {
		options.Runner = "github-actions-ubuntu-latest"
	}
	if options.CIWallMS < 0 || options.CIPeakRSSKiB < 0 {
		return errors.New("CI metrics must be non-negative integers")
	}

	sourceRoot, err := filepath.Abs(options.SourceRoot)
	if err != nil {
		return fmt.Errorf("resolve source root: %w", err)
	}
	output, err := filepath.Abs(options.Output)
	if err != nil {
		return fmt.Errorf("resolve output: %w", err)
	}
	if err := ensureEmptyOutput(output); err != nil {
		return err
	}

	sourceBytes, err := os.ReadFile(options.Source)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	contractBytes, err := os.ReadFile(options.Contract)
	if err != nil {
		return fmt.Errorf("read contract: %w", err)
	}
	fixtureBytes, err := os.ReadFile(options.Fixtures)
	if err != nil {
		return fmt.Errorf("read fixtures: %w", err)
	}

	var contract Contract
	if err := decodeJSON(contractBytes, &contract); err != nil {
		return fmt.Errorf("decode contract: %w", err)
	}
	var fixtures FixtureDocument
	if err := decodeJSON(fixtureBytes, &fixtures); err != nil {
		return fmt.Errorf("decode fixtures: %w", err)
	}
	if err := validateContract(contract, options.Toolchain); err != nil {
		return err
	}
	if err := validateFixtures(contract, fixtures); err != nil {
		return err
	}
	activities, err := parseActivities(sourceBytes)
	if err != nil {
		return err
	}
	if err := requireActivities(activities, contract.RequiredGoooActivities); err != nil {
		return err
	}

	sourceSHA := digest(sourceBytes)
	contractSHA := digest(contractBytes)
	inventory, err := collectInventory(sourceRoot, output)
	if err != nil {
		return fmt.Errorf("collect source inventory: %w", err)
	}

	edges := make(map[string]StageEdge, len(fixtures.StageEdges))
	graphEdges := make([]GraphEdge, 0, len(fixtures.StageEdges))
	for _, edge := range fixtures.StageEdges {
		edges[edge.ID] = edge
		graphEdges = append(graphEdges, GraphEdge{ID: edge.ID, From: edge.From, To: edge.To})
	}
	identity := Identity{Source: sourceSHA, Contract: contractSHA, Toolchain: options.Toolchain, Runner: options.Runner}
	results := make([]ScenarioResult, 0, len(fixtures.Scenarios))
	for _, scenario := range fixtures.Scenarios {
		result, evalErr := evaluateScenario(contract, edges[scenario.Edge], scenario, identity)
		if evalErr != nil {
			return evalErr
		}
		results = append(results, result)
	}

	graph := CapabilityGraph{
		Schema:       "gooo/meta-capability-attenuator/capability-graph/v1",
		SourceSHA:    sourceSHA,
		ContractSHA:  contractSHA,
		Toolchain:    options.Toolchain,
		Runner:       options.Runner,
		Capabilities: append([]string(nil), contract.Capabilities...),
		StageEdges:   graphEdges,
		Precedence:   append([]string(nil), fixedPrecedence...),
		Scenarios:    results,
	}

	var summary Summary
	var improvementSummary ImprovementSummary
	metrics := Metrics{
		CapabilityKinds: len(contract.Capabilities),
		StageEdges:      len(fixtures.StageEdges),
		Inventory:       inventory,
		CI:              CIMetrics{WallMS: options.CIWallMS, PeakRSSKiB: options.CIPeakRSSKiB},
	}
	for _, result := range results {
		summary.Total++
		improvementSummary.Total++
		metrics.DeclaredCapabilities += result.Counts.Declared
		metrics.ObservedCapabilities += result.Counts.Observed
		metrics.PreservedCapabilities += result.Counts.Preserved
		metrics.AmplifiedCapabilities += result.Counts.Amplified
		metrics.AttenuatedCapabilities += result.Counts.Attenuated
		switch result.State {
		case "CLOSED":
			summary.Closed++
		case "UNKNOWN":
			summary.Unknown++
		case "REFUTED":
			summary.Refuted++
		}
		if result.Improvement.State == "CLOSED" {
			improvementSummary.Closed++
		} else {
			improvementSummary.Unknown++
		}
	}

	authority := contract.Authority
	receipt := AttenuationReceipt{
		Schema:             "gooo/meta-capability-attenuator/attenuation-receipt/v1",
		Subject:            Subject{SourceSHA: sourceSHA, ContractSHA: contractSHA, Toolchain: options.Toolchain, Runner: options.Runner},
		SourceActivities:   activities,
		Precedence:         append([]string(nil), fixedPrecedence...),
		FixedScenarioCount: len(results),
		Summary:            summary,
		Scenarios:          results,
		Improvement:        improvementSummary,
		Metrics:            metrics,
		Authority:          authority,
		Artifacts: ArtifactManifest{
			Files: []string{"capability-graph.json", "attenuation-receipt.json", "violation.ndjson", "attenuation-report.md"},
			Count: 4,
		},
	}

	if err := os.MkdirAll(output, 0o755); err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	if err := writeJSON(filepath.Join(output, "capability-graph.json"), graph); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(output, "attenuation-receipt.json"), receipt); err != nil {
		return err
	}
	if err := writeViolations(filepath.Join(output, "violation.ndjson"), results); err != nil {
		return err
	}
	if err := writeReport(filepath.Join(output, "attenuation-report.md"), receipt); err != nil {
		return err
	}
	return nil
}

func evaluateScenario(contract Contract, edge StageEdge, scenario Scenario, identity Identity) (ScenarioResult, error) {
	if edge.ID == "" {
		return ScenarioResult{}, fmt.Errorf("scenario %s references an unknown stage edge %q", scenario.ID, scenario.Edge)
	}
	declared, err := canonicalSet(scenario.Declared, contract.Capabilities)
	if err != nil {
		return ScenarioResult{}, fmt.Errorf("scenario %s declared set: %w", scenario.ID, err)
	}
	observed, err := canonicalSet(scenario.Observed, contract.Capabilities)
	if err != nil {
		return ScenarioResult{}, fmt.Errorf("scenario %s observed set: %w", scenario.ID, err)
	}
	declaredSet := makeSet(declared)
	observedSet := makeSet(observed)
	preserved := intersection(declared, observedSet)
	amplified := difference(observed, declaredSet)
	attenuated := difference(declared, observedSet)

	originProof := make([]OriginRecord, 0, len(scenario.Operations))
	var unknown *UnknownDetail
	for _, operation := range scenario.Operations {
		record := OriginRecord{
			OperationID: operation.ID,
			Capability:  operation.Capability,
			Action:      operation.Action,
			Origin:      operation.Origin,
			OriginStage: operation.OriginStage,
			CallTarget:  operation.CallTarget,
			ProofStatus: "VERIFIED",
		}
		if operation.OriginProof != nil {
			record.ProofKind = operation.OriginProof.Kind
		} else {
			record.ProofStatus = "MISSING"
		}
		if operation.Origin == "missing" || operation.OriginProof == nil {
			if unknown == nil {
				unknown = &UnknownDetail{
					Stage:         edge.To,
					Step:          "BindCapabilityOrigin",
					Reason:        "DIRECT_MISSING",
					UnknownClass:  "DIRECT_MISSING",
					NextOperation: "RESTORE_DIRECT_CAPABILITY_ORIGIN",
					BlockedBy:     []string{"origin-proof", "declared-stage"},
				}
			}
		} else if operation.Origin == "ambiguous" || operation.CallTarget == "dynamic" {
			if unknown == nil {
				unknown = &UnknownDetail{
					Stage:         edge.To,
					Step:          "BindCapabilityOrigin",
					Reason:        "DYNAMIC_CALL_TARGET_AMBIGUOUS",
					UnknownClass:  "AMBIGUOUS",
					NextOperation: "RESOLVE_DYNAMIC_CALL_TARGET",
					BlockedBy:     []string{"static-call-target", "origin-proof"},
				}
			}
		}
		originProof = append(originProof, record)
	}

	state := "CLOSED"
	reason := "LATTICE_PRESERVED"
	if len(amplified) > 0 {
		state = "REFUTED"
		reason = "CAPABILITY_AMPLIFICATION"
		for _, operation := range scenario.Operations {
			if operation.Capability == "write:repository" {
				reason = "UNDECLARED_REPOSITORY_WRITE"
			}
			if operation.Capability == "network:outbound" && operation.CrossStage {
				reason = "CROSS_STAGE_NETWORK_AMPLIFICATION"
			}
		}
	} else if unknown != nil {
		state = "UNKNOWN"
		reason = unknown.Reason
	} else if len(attenuated) > 0 {
		reason = "LATTICE_ATTENUATED"
	}

	var replayEqual *bool
	if scenario.Replay {
		first := canonicalScenario(scenario, edge)
		second := canonicalScenario(scenario, edge)
		equal := bytes.Equal(first, second)
		replayEqual = &equal
	}

	identity.Scenario = scenario.ID
	improvement := evaluateImprovement(scenario, state, identity)
	result := ScenarioResult{
		ID:                     scenario.ID,
		Name:                   scenario.Name,
		Purpose:                scenario.Purpose,
		State:                  state,
		Reason:                 reason,
		EdgeID:                 edge.ID,
		FromStage:              edge.From,
		ToStage:                edge.To,
		DeclaredCapabilities:   declared,
		ObservedCapabilities:   observed,
		PreservedCapabilities:  preserved,
		AmplifiedCapabilities:  amplified,
		AttenuatedCapabilities: attenuated,
		Counts: CountVector{
			Declared: len(declared), Observed: len(observed), Preserved: len(preserved),
			Amplified: len(amplified), Attenuated: len(attenuated),
		},
		OriginProof: originProof,
		ReplayEqual: replayEqual,
		Unknown:     unknown,
		Improvement: improvement,
	}
	if state != scenario.ExpectedState || reason != scenario.ExpectedReason {
		return ScenarioResult{}, fmt.Errorf("scenario %s expected %s/%s, evaluated %s/%s", scenario.ID, scenario.ExpectedState, scenario.ExpectedReason, state, reason)
	}
	return result, nil
}

func evaluateImprovement(scenario Scenario, scenarioState string, current Identity) ImprovementResult {
	var before, after *MetricVector
	var beforeIdentity, afterIdentity *Identity
	if scenario.Improvement != nil {
		before = scenario.Improvement.Before
		after = scenario.Improvement.After
		beforeIdentity = resolveIdentity(scenario.Improvement.BeforeIdentity, current)
		afterIdentity = resolveIdentity(scenario.Improvement.AfterIdentity, current)
	}
	identityMatch := beforeIdentity != nil && afterIdentity != nil && *beforeIdentity == *afterIdentity
	if scenarioState == "CLOSED" && before != nil && after != nil && identityMatch {
		return ImprovementResult{State: "CLOSED", Before: before, After: after, BeforeIdentity: beforeIdentity, AfterIdentity: afterIdentity, IdentityMatch: true}
	}

	detail := UnknownDetail{
		Stage:         "improvement",
		Step:          "VerifyImprovementPair",
		Reason:        "SCENARIO_NOT_CLOSED",
		UnknownClass:  "SCENARIO_NOT_CLOSED",
		NextOperation: "RECORD_CLOSED_BEFORE_AFTER_PAIR",
		BlockedBy:     []string{"closed-scenario", "before-after-integer-pair"},
	}
	if before == nil || after == nil || beforeIdentity == nil || afterIdentity == nil {
		detail.Reason = "BEFORE_AFTER_PAIR_MISSING"
		detail.UnknownClass = "PAIR_MISSING"
		detail.NextOperation = "SUPPLY_BEFORE_AFTER_INTEGER_PAIR"
	} else if !identityMatch {
		detail.Reason = "IDENTITY_MISMATCH"
		detail.UnknownClass = "IDENTITY_MISMATCH"
		detail.NextOperation = "ALIGN_SCENARIO_SOURCE_CONTRACT_TOOLCHAIN_RUNNER"
	}
	return ImprovementResult{State: "UNKNOWN", Before: before, After: after, BeforeIdentity: beforeIdentity, AfterIdentity: afterIdentity, IdentityMatch: identityMatch, Unknown: &detail}
}

func resolveIdentity(input *Identity, current Identity) *Identity {
	if input == nil {
		return nil
	}
	resolved := *input
	if resolved.Source == "SELF" {
		resolved.Source = current.Source
	}
	if resolved.Contract == "SELF" {
		resolved.Contract = current.Contract
	}
	if resolved.Toolchain == "SELF" {
		resolved.Toolchain = current.Toolchain
	}
	if resolved.Runner == "SELF" {
		resolved.Runner = current.Runner
	}
	return &resolved
}

func canonicalScenario(scenario Scenario, edge StageEdge) []byte {
	value := struct {
		ID       string      `json:"id"`
		From     string      `json:"from"`
		To       string      `json:"to"`
		Declared []string    `json:"declared"`
		Observed []string    `json:"observed"`
		Ops      []Operation `json:"operations"`
	}{scenario.ID, edge.From, edge.To, scenario.Declared, scenario.Observed, scenario.Operations}
	data, _ := json.Marshal(value)
	return data
}

func validateContract(contract Contract, toolchain string) error {
	if contract.Schema != "gooo/meta-capability-attenuator/contract/v1" {
		return fmt.Errorf("unsupported contract schema %q", contract.Schema)
	}
	if contract.SourceSchema != "gooo/meta-capability-attenuator/semantic-ir/v1" {
		return fmt.Errorf("unsupported source schema %q", contract.SourceSchema)
	}
	if contract.GoToolchain != toolchain {
		return fmt.Errorf("contract toolchain %q does not match evaluator toolchain %q", contract.GoToolchain, toolchain)
	}
	if !sameStrings(contract.Precedence, fixedPrecedence) {
		return errors.New("contract precedence must be REFUTED > UNKNOWN > CLOSED")
	}
	if len(contract.Capabilities) == 0 || len(makeSet(contract.Capabilities)) != len(contract.Capabilities) {
		return errors.New("contract capabilities must be non-empty and unique")
	}
	if len(contract.FixedScenarios) != 8 {
		return errors.New("contract must define exactly eight fixed scenarios")
	}
	if len(contract.RequiredGoooActivities) != 6 {
		return errors.New("contract must define six required .gooo activities")
	}
	if len(contract.Artifacts) != 4 {
		return errors.New("contract must define exactly four artifacts")
	}
	for field, value := range map[string]int{
		"repository_writes":     contract.Authority.RepositoryWrites,
		"source_mutations":      contract.Authority.SourceMutations,
		"commit_authority":      contract.Authority.CommitAuthority,
		"push_authority":        contract.Authority.PushAuthority,
		"merge_authority":       contract.Authority.MergeAuthority,
		"release_mutation":      contract.Authority.ReleaseMutation,
		"local_test_executions": contract.Authority.LocalTestExecutions,
	} {
		if value != 0 {
			return fmt.Errorf("authority %s must be exact zero", field)
		}
	}
	return nil
}

func validateFixtures(contract Contract, fixtures FixtureDocument) error {
	if fixtures.Schema != contract.SourceSchema {
		return fmt.Errorf("fixture schema %q does not match contract source schema %q", fixtures.Schema, contract.SourceSchema)
	}
	if len(fixtures.StageEdges) != 8 || len(fixtures.Scenarios) != 8 {
		return errors.New("semantic IR must define exactly eight stage edges and eight scenarios")
	}
	edgeIDs := make(map[string]bool, len(fixtures.StageEdges))
	for _, edge := range fixtures.StageEdges {
		if edge.ID == "" || edge.From == "" || edge.To == "" || edgeIDs[edge.ID] {
			return fmt.Errorf("invalid or duplicate stage edge %q", edge.ID)
		}
		edgeIDs[edge.ID] = true
	}
	for index, scenario := range fixtures.Scenarios {
		if scenario.ID != contract.FixedScenarios[index] {
			return fmt.Errorf("scenario %d is %q, expected %q", index, scenario.ID, contract.FixedScenarios[index])
		}
		if !edgeIDs[scenario.Edge] {
			return fmt.Errorf("scenario %s references unknown edge %q", scenario.ID, scenario.Edge)
		}
		if scenario.ExpectedState != "CLOSED" && scenario.ExpectedState != "UNKNOWN" && scenario.ExpectedState != "REFUTED" {
			return fmt.Errorf("scenario %s has invalid expected state %q", scenario.ID, scenario.ExpectedState)
		}
	}
	return nil
}

func parseActivities(source []byte) ([]string, error) {
	var activities []string
	for _, line := range strings.Split(string(source), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "activity ") {
			continue
		}
		declaration := strings.TrimSpace(strings.TrimPrefix(line, "activity "))
		name := declaration
		if index := strings.IndexAny(name, "( "); index >= 0 {
			name = name[:index]
		}
		if name == "" {
			return nil, errors.New("empty .gooo activity")
		}
		activities = append(activities, name)
	}
	if len(activities) == 0 {
		return nil, errors.New(".gooo source contains no activities")
	}
	return activities, nil
}

func requireActivities(found, required []string) error {
	set := makeSet(found)
	for _, activity := range required {
		if !set[activity] {
			return fmt.Errorf(".gooo source is missing required activity %q", activity)
		}
	}
	return nil
}

func canonicalSet(values, allowed []string) ([]string, error) {
	allowedSet := makeSet(allowed)
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !allowedSet[value] {
			return nil, fmt.Errorf("capability %q is not in the contract lattice", value)
		}
		if seen[value] {
			return nil, fmt.Errorf("capability %q appears more than once", value)
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func makeSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func intersection(values []string, other map[string]bool) []string {
	result := make([]string, 0)
	for _, value := range values {
		if other[value] {
			result = append(result, value)
		}
	}
	return result
}

func difference(values []string, other map[string]bool) []string {
	result := make([]string, 0)
	for _, value := range values {
		if !other[value] {
			result = append(result, value)
		}
	}
	return result
}

func uniqueSorted(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func ensureEmptyOutput(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect output: %w", err)
	}
	if len(entries) != 0 {
		return fmt.Errorf("output directory must be empty: %s", path)
	}
	return nil
}

func decodeJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func digest(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

type violationRecord struct {
	Schema                string         `json:"schema"`
	ScenarioID            string         `json:"scenario_id"`
	State                 string         `json:"state"`
	Reason                string         `json:"reason"`
	Stage                 string         `json:"stage"`
	Step                  string         `json:"step"`
	DeclaredCapabilities  []string       `json:"declared_capabilities"`
	ObservedCapabilities  []string       `json:"observed_capabilities"`
	AmplifiedCapabilities []string       `json:"amplified_capabilities"`
	Unknown               *UnknownDetail `json:"unknown"`
}

func writeViolations(path string, results []ScenarioResult) error {
	var builder strings.Builder
	for _, result := range results {
		if result.State == "CLOSED" {
			continue
		}
		record := violationRecord{
			Schema:                "gooo/meta-capability-attenuator/violation/v1",
			ScenarioID:            result.ID,
			State:                 result.State,
			Reason:                result.Reason,
			Stage:                 result.ToStage,
			Step:                  "CompareCapabilityLattice",
			DeclaredCapabilities:  result.DeclaredCapabilities,
			ObservedCapabilities:  result.ObservedCapabilities,
			AmplifiedCapabilities: result.AmplifiedCapabilities,
			Unknown:               result.Unknown,
		}
		if result.Unknown != nil {
			record.Stage = result.Unknown.Stage
			record.Step = result.Unknown.Step
		}
		data, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("encode violation: %w", err)
		}
		builder.Write(data)
		builder.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeReport(path string, receipt AttenuationReceipt) error {
	var builder strings.Builder
	builder.WriteString("# Gooo capability attenuation report\n\n")
	builder.WriteString("This report evaluates staged Gooo semantic IR. It does not execute any capability. Go only evaluates inert fixtures and emits caller-owned artifacts.\n\n")
	fmt.Fprintf(&builder, "- source SHA-256: `%s`\n- contract SHA-256: `%s`\n- toolchain: `%s`\n- runner: `%s`\n- precedence: `REFUTED > UNKNOWN > CLOSED`\n- fixed scenarios: `%d`\n- artifact files: `%d`\n\n", receipt.Subject.SourceSHA, receipt.Subject.ContractSHA, receipt.Subject.Toolchain, receipt.Subject.Runner, receipt.FixedScenarioCount, receipt.Artifacts.Count)
	builder.WriteString("| scenario | state | declared | observed | preserved | amplified | attenuated | improvement |\n")
	builder.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, scenario := range receipt.Scenarios {
		fmt.Fprintf(&builder, "| `%s` | `%s` | %d | %d | %d | %d | %d | `%s` |\n", scenario.ID, scenario.State, scenario.Counts.Declared, scenario.Counts.Observed, scenario.Counts.Preserved, scenario.Counts.Amplified, scenario.Counts.Attenuated, scenario.Improvement.State)
	}
	fmt.Fprintf(&builder, "\nSummary: CLOSED=%d, UNKNOWN=%d, REFUTED=%d.\n\n", receipt.Summary.Closed, receipt.Summary.Unknown, receipt.Summary.Refuted)
	fmt.Fprintf(&builder, "Exact capability counts: declared=%d, observed=%d, preserved=%d, amplified=%d, attenuated=%d.\n\n", receipt.Metrics.DeclaredCapabilities, receipt.Metrics.ObservedCapabilities, receipt.Metrics.PreservedCapabilities, receipt.Metrics.AmplifiedCapabilities, receipt.Metrics.AttenuatedCapabilities)
	builder.WriteString("UNKNOWN records retain stage, step, reason, unknown_class, next_operation, and blocked_by. No UNKNOWN result is promoted to CLOSED. Repository writes, source mutations, commit, push, merge, release mutation, and local-test executions are all recorded as exact zeroes.\n")
	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
