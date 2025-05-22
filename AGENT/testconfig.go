package agent

type TestConfig struct {
	TestID         int    `json:"test_id"`
	Name           string `json:"name"`
	Duration       string `json:"duration"`
	NumberOfAgents int    `json:"number_of_agents"`
	SourceID       int    `json:"source_id"`
	TargetID       int    `json:"target_id"`
	ProfileID      int    `json:"profile_id"`
	ThresholdID    int    `json:"threshold_id"`
}


type TestConfigWithAgents struct {
	TestID         int
	Name           string
	Duration       int
	NumberOfAgents int
	SourceID       int
	SourceIP       string
	SourcePort     int
	TargetID       int
	TargetIP       string
	TargetPort     int
	ProfileID      int
	ThresholdID    int
}

