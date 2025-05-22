package core 

type TestConfigWithAgents struct {
	TestID         int
	Name           string
	Duration       string
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
