package attenuator

type Contract struct {
	Schema                 string    `json:"schema"`
	SourceSchema           string    `json:"source_schema"`
	GoToolchain            string    `json:"go_toolchain"`
	Precedence             []string  `json:"precedence"`
	Capabilities           []string  `json:"capabilities"`
	FixedScenarios         []string  `json:"fixed_scenarios"`
	RequiredGoooActivities []string  `json:"required_gooo_activities"`
	InventoryExclusions    []string  `json:"inventory_exclusions"`
	Authority              Authority `json:"authority"`
	Artifacts              []string  `json:"artifacts"`
}

type FixtureDocument struct {
	Schema     string      `json:"schema"`
	StageEdges []StageEdge `json:"stage_edges"`
	Scenarios  []Scenario  `json:"scenarios"`
}

type StageEdge struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

type Scenario struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	Purpose        string               `json:"purpose"`
	Edge           string               `json:"edge"`
	Declared       []string             `json:"declared"`
	Observed       []string             `json:"observed"`
	Operations     []Operation          `json:"operations"`
	Replay         bool                 `json:"replay"`
	ExpectedState  string               `json:"expected_state"`
	ExpectedReason string               `json:"expected_reason"`
	Improvement    *ImprovementEvidence `json:"improvement"`
}

type Operation struct {
	ID          string       `json:"id"`
	Capability  string       `json:"capability"`
	Action      string       `json:"action"`
	Origin      string       `json:"origin"`
	OriginStage string       `json:"origin_stage"`
	CallTarget  string       `json:"call_target"`
	CrossStage  bool         `json:"cross_stage"`
	OriginProof *OriginProof `json:"origin_proof"`
}

type OriginProof struct {
	Kind           string `json:"kind"`
	DeclaredStage  string `json:"declared_stage"`
	SourceActivity string `json:"source_activity"`
	Evidence       string `json:"evidence"`
}

type ImprovementEvidence struct {
	Before         *MetricVector `json:"before"`
	After          *MetricVector `json:"after"`
	BeforeIdentity *Identity     `json:"before_identity"`
	AfterIdentity  *Identity     `json:"after_identity"`
}

type MetricVector struct {
	DeclaredCount   int `json:"declared_count"`
	ObservedCount   int `json:"observed_count"`
	AmplifiedCount  int `json:"amplified_count"`
	AttenuatedCount int `json:"attenuated_count"`
}

type Identity struct {
	Scenario  string `json:"scenario"`
	Source    string `json:"source"`
	Contract  string `json:"contract"`
	Toolchain string `json:"toolchain"`
	Runner    string `json:"runner"`
}

type UnknownDetail struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type CountVector struct {
	Declared   int `json:"declared"`
	Observed   int `json:"observed"`
	Preserved  int `json:"preserved"`
	Amplified  int `json:"amplified"`
	Attenuated int `json:"attenuated"`
}

type OriginRecord struct {
	OperationID string `json:"operation_id"`
	Capability  string `json:"capability"`
	Action      string `json:"action"`
	Origin      string `json:"origin"`
	OriginStage string `json:"origin_stage"`
	CallTarget  string `json:"call_target"`
	ProofKind   string `json:"proof_kind"`
	ProofStatus string `json:"proof_status"`
}

type ImprovementResult struct {
	State          string         `json:"state"`
	Before         *MetricVector  `json:"before"`
	After          *MetricVector  `json:"after"`
	BeforeIdentity *Identity      `json:"before_identity"`
	AfterIdentity  *Identity      `json:"after_identity"`
	IdentityMatch  bool           `json:"identity_match"`
	Unknown        *UnknownDetail `json:"unknown"`
}

type ScenarioResult struct {
	ID                     string            `json:"id"`
	Name                   string            `json:"name"`
	Purpose                string            `json:"purpose"`
	State                  string            `json:"state"`
	Reason                 string            `json:"reason"`
	EdgeID                 string            `json:"edge_id"`
	FromStage              string            `json:"from_stage"`
	ToStage                string            `json:"to_stage"`
	DeclaredCapabilities   []string          `json:"declared_capabilities"`
	ObservedCapabilities   []string          `json:"observed_capabilities"`
	PreservedCapabilities  []string          `json:"preserved_capabilities"`
	AmplifiedCapabilities  []string          `json:"amplified_capabilities"`
	AttenuatedCapabilities []string          `json:"attenuated_capabilities"`
	Counts                 CountVector       `json:"counts"`
	OriginProof            []OriginRecord    `json:"origin_proof"`
	ReplayEqual            *bool             `json:"replay_equal"`
	Unknown                *UnknownDetail    `json:"unknown"`
	Improvement            ImprovementResult `json:"improvement"`
}

type GraphEdge struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

type CapabilityGraph struct {
	Schema       string           `json:"schema"`
	SourceSHA    string           `json:"source_sha"`
	ContractSHA  string           `json:"contract_sha"`
	Toolchain    string           `json:"toolchain"`
	Runner       string           `json:"runner"`
	Capabilities []string         `json:"capability_kinds"`
	StageEdges   []GraphEdge      `json:"stage_edges"`
	Precedence   []string         `json:"precedence"`
	Scenarios    []ScenarioResult `json:"scenarios"`
}

type Subject struct {
	SourceSHA   string `json:"source_sha"`
	ContractSHA string `json:"contract_sha"`
	Toolchain   string `json:"toolchain"`
	Runner      string `json:"runner"`
}

type Summary struct {
	Total   int `json:"total"`
	Closed  int `json:"closed"`
	Unknown int `json:"unknown"`
	Refuted int `json:"refuted"`
}

type ImprovementSummary struct {
	Total   int `json:"total"`
	Closed  int `json:"closed"`
	Unknown int `json:"unknown"`
}

type Inventory struct {
	RootReadmeExcluded bool `json:"root_readme_excluded"`
	Files              int  `json:"files"`
	Directories        int  `json:"directories"`
	Bytes              int  `json:"bytes"`
	GoFiles            int  `json:"go_files"`
	GoLines            int  `json:"go_lines"`
	GoooFiles          int  `json:"gooo_files"`
	GoooLines          int  `json:"gooo_lines"`
}

type CIMetrics struct {
	WallMS     int `json:"wall_ms"`
	PeakRSSKiB int `json:"peak_rss_kib"`
}

type Metrics struct {
	CapabilityKinds        int       `json:"capability_kinds"`
	StageEdges             int       `json:"stage_edges"`
	DeclaredCapabilities   int       `json:"declared_capabilities"`
	ObservedCapabilities   int       `json:"observed_capabilities"`
	PreservedCapabilities  int       `json:"preserved_capabilities"`
	AmplifiedCapabilities  int       `json:"amplified_capabilities"`
	AttenuatedCapabilities int       `json:"attenuated_capabilities"`
	Inventory              Inventory `json:"inventory"`
	CI                     CIMetrics `json:"ci"`
}

type Authority struct {
	RepositoryWrites    int `json:"repository_writes"`
	SourceMutations     int `json:"source_mutations"`
	CommitAuthority     int `json:"commit_authority"`
	PushAuthority       int `json:"push_authority"`
	MergeAuthority      int `json:"merge_authority"`
	ReleaseMutation     int `json:"release_mutation"`
	LocalTestExecutions int `json:"local_test_executions"`
}

type ArtifactManifest struct {
	Files []string `json:"files"`
	Count int      `json:"count"`
}

type AttenuationReceipt struct {
	Schema             string             `json:"schema"`
	Subject            Subject            `json:"subject"`
	SourceActivities   []string           `json:"source_activities"`
	Precedence         []string           `json:"precedence"`
	FixedScenarioCount int                `json:"fixed_scenario_count"`
	Summary            Summary            `json:"summary"`
	Scenarios          []ScenarioResult   `json:"scenarios"`
	Improvement        ImprovementSummary `json:"improvement"`
	Metrics            Metrics            `json:"metrics"`
	Authority          Authority          `json:"authority"`
	Artifacts          ArtifactManifest   `json:"artifacts"`
}
