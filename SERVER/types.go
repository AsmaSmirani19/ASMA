package server

import (
	"time"
	"github.com/lib/pq"
	
)


type HealthUpdate struct {
	IP     string `json:"ip"`
	Status string `json:"status"` // OK or FAIL
}

type TestStatus struct {
	TestID int    `json:"test_id"`
	Status string `json:"status"` // "pending", "running", "finished", etc.
}

type QoSMetrics struct {
	PacketLossPercent float64 `json:"packet_loss_percent"`
	AvgLatencyMs      int64   `json:"avg_latency_ms"`
	AvgJitterMs       int64   `json:"avg_jitter_ms"`
	AvgThroughputKbps float64 `json:"avg_throughput_kbps"`
	TotalJitter       int64   `json:"total_jitter"`
}

type AttemptResult struct {
    TestID         int64   `json:"test_id"`        
    LatencyMs      float64 `json:"latency_ms"`
    JitterMs       float64 `json:"jitter_ms"`
    ThroughputKbps float64 `json:"throughput_kbps"`
}

type TestConfig struct {
	TestID         int    `json:"test_id"`
	Name           string `json:"name"`
	Duration       string   `json:"duration"`
	NumberOfAgents int    `json:"number_of_agents"`
	SourceID       int    `json:"source_id"`
	TargetID       int    `json:"target_id"`
	ProfileID      int    `json:"profile_id"`
	ThresholdID    int    `json:"threshold_id"`
}

type Agent struct {
	ID           int                 `json:"id"          db:"id"`
	Port         int  
	Name         string              `json:"name"        db:"Name"`
	Address      string              `json:"address"     db:"Address"`
	TestHealth   bool                `json:"testhealth"  db:"Test_health"`
}

type AgentHealthCheck struct {
    ID        int       `db:"id"`         
    AgentID   int       `db:"agent_id"`
    Timestamp time.Time `db:"timestamp"`
    Status    string    `db:"status"`
}

type PlannedTest struct {
	ID             int       `json:"id"`
	TestName       string    `json:"test_name"`
	TestDuration   string    `json:"test_duration"`
	NumberOfAgents int       `json:"number_of_agents"`
	CreationDate   time.Time `json:"creation_date"`
	TestType       string    `json:"test_type"`
	SourceID       int       `json:"source_id"`
	TargetID       int       `json:"target_id"`
	ProfileID      int       `json:"profile_id"`
	ThresholdID    int       `json:"threshold_id"`
	InProgress     bool      `json:"inProgress"` 
	Failed         bool      `json:"failed"`
	Completed      bool      `json:"completed"`
	Error          bool      `json:"error"`
}


type agentGroup struct {
	ID             int               `json:"id"`
	GroupName     string             `json:"group_name"`
	NumberOfAgents  int             `json:"number_of_agents"`
	CreationDate   time.Time         `json:"creation_date"` 
	AgentIDs       pq.Int64Array     `json:"agent_ids"`
}

type CreateGroupPayload struct {
	GroupName    string   `json:"group_name"`
	CreationDate string   `json:"creation_date"`
	AgentIDs     []int    `json:"agent_ids"`
}

type AgentLinkPayload struct {
	GroupID  int   `json:"group_id"`
	AgentIDs []int `json:"agent_ids"`
}

type testProfile struct {
    ID                  int       `json:"id"`
    ProfileName         string    `json:"profile_name"`
    CreationDate        time.Time `json:"creation_date"`
    PacketSize          int       `json:"packet_size"`
    TimeBetweenAttempts int       `json:"time_between_attempts"`
}

type Threshold struct {
	ID                uint     `json:"id"`
	Name              string   `json:"name"`
	CreationDate      string   `json:"creation_date"`
	Avg               float64  `json:"avg"`
	Min               float64  `json:"min"`
	Max               float64  `json:"max"`
	AvgStatus         bool     `json:"avg_status"`
	MinStatus         bool     `json:"min_status"`
	MaxStatus         bool     `json:"max_status"`
	AvgOpr            string   `json:"avg_opr"`
	MinOpr            string   `json:"min_opr"`
	MaxOpr            string   `json:"max_opr"`
	SelectedMetric     string   `json:"selected_metric"`
	
	ActiveThresholds  []string `json:"active_thresholds"`
	DisabledThresholds []string `json:"disabled_thresholds"`
}

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

type DisplayedTest struct {
	TestID         int       `json:"test_id"`
	TestName      string    `json:"test_name"`
	TestType      string    `json:"test_type"`
	CreationDate  time.Time `json:"creation_date"`
	TestDuration  string    `json:"test_duration"`
	SourceAgent   string    `json:"source_agent"`
	TargetAgent   string    `json:"target_agent"`
	 
	Min           float64   `json:"min"`
	Max           float64   `json:"max"`
	Avg           float64   `json:"avg"`
	ThresholdName string    `json:"threshold_name"`
	ThresholdValue float64  `json:"threshold_value"`

	Status        string    `json:"status"`
	InProgress    bool      `json:"in_progress"`
	Completed     bool      `json:"completed"`
	Failed        bool      `json:"failed"`
	Error         bool      `json:"error"`
}

type TestDetails struct {
    TestID         int     `json:"test_id"`
    TestName       string  `json:"testName"`
    Status         string  `json:"status"`
    CreationDate   string  `json:"creationDate"`
    TestDuration   string  `json:"testDuration"`
    SourceAgent    string  `json:"sourceAgent"`
    TargetAgent    string  `json:"targetAgent"`
    ThresholdName  string  `json:"thresholdName"`
    ThresholdValue float64 `json:"thresholdValue"`
}