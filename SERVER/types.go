package server

import (
	"time"	
)


type TestStatus struct {
	TestID int    `json:"test_id"`
	Status string `json:"status"` 
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

type DisplayedTest struct {
	TestID         int       `json:"test_id"`
	TestName      string    `json:"test_name"`
	TestType      string    `json:"test_type"`
	CreationDate  time.Time `json:"creation_date"`
	TestDuration  string    `json:"test_duration"`
	SourceAgent   string    `json:"source_agent"`
	TargetAgent   string    `json:"target_agent"`
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


/////////////////////
type TestKafkaMessage struct {
	TestID     int           `json:"test_id"`
	TestType   string        `json:"test_type"`    
	Sender     string        `json:"sender"`      
	Reflectors []string      `json:"reflectors"`  
	Profile    *Profile      `json:"profile"`     
}
////////////////////////////


type Profile struct {
	ID              int           `json:"id"`
	SendingInterval time.Duration `json:"sending_interval"`
	PacketSize      int           `json:"packet_size"`
	PacketRate      int           `json:"packet_rate"`
}

type FullTestConfiguration struct {
	TestID          int           `json:"test_id"`
	Name            string        `json:"name"`
	TestType        string        `json:"test_type"` // "quick", "planned", etc.
	RawDuration     string        `json:"raw_duration"`
	Duration        time.Duration `json:"duration"`
	NumberOfAgents  int           `json:"number_of_agents"`
	ProfileID       int           `json:"profile_id"`
	Profile         *Profile      `json:"profile"`
	TargetAgentIDs  []int         `json:"target_ids,omitempty"`
	SourceID        int           `json:"source_id"`
	SourceIP        string        `json:"source_ip"`
	SourcePort      int           `json:"source_port"`
	TargetID        int           `json:"target_id"`
	TargetIP        string        `json:"target_ip"`
	TargetPort      int           `json:"target_port"`
	TargetAgents    []AgentInfo
}

type Target struct {
    IP   string `json:"ip"`
    Port int    `json:"port"`
}

type AgentInfo struct {
    ID   int
    IP   string
    Port int
}

