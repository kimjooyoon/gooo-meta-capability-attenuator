package attenuator

import "testing"

func TestCapabilityLatticeRefutesAmplification(t *testing.T) {
	contract := Contract{Capabilities: []string{"read:source", "write:repository"}}
	scenario := Scenario{
		ID:             "amplification",
		Declared:       []string{"read:source"},
		Observed:       []string{"read:source", "write:repository"},
		ExpectedState:  "REFUTED",
		ExpectedReason: "UNDECLARED_REPOSITORY_WRITE",
		Operations: []Operation{{
			ID:         "op",
			Capability: "write:repository",
			Action:     "repository.write",
			Origin:     "direct",
			OriginProof: &OriginProof{
				Kind: "direct-declaration",
			},
		}},
	}
	result, err := evaluateScenario(contract, StageEdge{ID: "edge", From: "a", To: "b"}, scenario, Identity{})
	if err != nil {
		t.Fatalf("evaluate scenario: %v", err)
	}
	if result.State != "REFUTED" || result.Counts.Amplified != 1 {
		t.Fatalf("got state=%s amplified=%d", result.State, result.Counts.Amplified)
	}
}

func TestMissingOriginStaysUnknown(t *testing.T) {
	contract := Contract{Capabilities: []string{"write:caller-owned-output"}}
	scenario := Scenario{
		ID:             "missing-origin",
		Declared:       []string{"write:caller-owned-output"},
		Observed:       []string{"write:caller-owned-output"},
		ExpectedState:  "UNKNOWN",
		ExpectedReason: "DIRECT_MISSING",
		Operations: []Operation{{
			ID:         "op",
			Capability: "write:caller-owned-output",
			Action:     "emit.caller_output",
			Origin:     "missing",
		}},
	}
	result, err := evaluateScenario(contract, StageEdge{ID: "edge", From: "a", To: "b"}, scenario, Identity{})
	if err != nil {
		t.Fatalf("evaluate scenario: %v", err)
	}
	if result.State != "UNKNOWN" || result.Unknown == nil || len(result.Unknown.BlockedBy) != 2 {
		t.Fatalf("missing required unknown detail: %#v", result.Unknown)
	}
}

func TestImprovementNeedsMatchingIdentity(t *testing.T) {
	scenario := Scenario{Improvement: &ImprovementEvidence{
		Before:         &MetricVector{DeclaredCount: 1},
		After:          &MetricVector{DeclaredCount: 1},
		BeforeIdentity: &Identity{Scenario: "same", Source: "a", Contract: "b", Toolchain: "go1.27.0", Runner: "runner-a"},
		AfterIdentity:  &Identity{Scenario: "same", Source: "a", Contract: "b", Toolchain: "go1.27.0", Runner: "runner-b"},
	}}
	result := evaluateImprovement(scenario, "CLOSED", Identity{})
	if result.State != "UNKNOWN" || result.Unknown == nil || result.Unknown.UnknownClass != "IDENTITY_MISMATCH" {
		t.Fatalf("got improvement result: %#v", result)
	}
}
