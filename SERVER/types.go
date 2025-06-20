package server

import (
	"time"	
	"database/sql"
)


type TestStatus struct {
	TestID int    `json:"test_id"`
	Status string `json:"status"` 
}

type QoSMetrics struct {
	TestID            int     `json:"test_id"`
	TargetID          int     `json:"target_id"`
	AvgLatencyMs      float64 `json:"avg_latency_ms"`
	AvgJitterMs       float64 `json:"avg_jitter_ms"`
	AvgThroughputKbps float64 `json:"avg_throughput_kbps"`
	PacketLossPercent float64 `json:"packet_loss_percent"`
}



type AttemptResult struct {
	TestID         int64   `json:"test_id"`
	TargetID       int64   `json:"target_id"`
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
	TargetIDs      []int
	ThresholdName string    `json:"threshold_name"`
	ThresholdValue float64  `json:"threshold_value"` 
	Status        string    `json:"status"`
	InProgress    bool      `json:"in_progress"`
	Completed     bool      `json:"completed"`
	Failed        bool      `json:"failed"`
	Error         bool      `json:"error"`
}
		
type TestDetails struct {
    TestID            int            `json:"test_id"`
    TestName          string         `json:"testName"`
    Status            string         `json:"status"`
    CreationDate      string         `json:"creationDate"`
    TestDuration      string         `json:"testDuration"`
    SourceAgent       string         `json:"sourceAgent"`
    TargetAgent       string         `json:"targetAgent"`
    ThresholdName     string         `json:"thresholdName"`
    ThresholdValue    sql.NullFloat64 `json:"thresholdValue"`
    SelectedMetric    string         `json:"selectedMetric"`
    ThresholdType     sql.NullString  `json:"thresholdType"`
    ThresholdOperator sql.NullString  `json:"thresholdOperator"`
}

type TestDetailsDTO struct {
    TestID            int     `json:"test_id"`
    TestName          string  `json:"testName"`
    Status            string  `json:"status"`
    CreationDate      string  `json:"creationDate"`
    TestDuration      string  `json:"testDuration"`
    SourceAgent       string  `json:"sourceAgent"`
    TargetAgent       string  `json:"targetAgent"`
    ThresholdName     string  `json:"thresholdName"`
    ThresholdValue    float64 `json:"thresholdValue"`
    SelectedMetric    string  `json:"selectedMetric"`
    ThresholdType     string  `json:"thresholdType"`
    ThresholdOperator string  `json:"thresholdOperator"`
}

func ConvertToTestDetailsDTO(t TestDetails) TestDetailsDTO {
    dto := TestDetailsDTO{
        TestID:         t.TestID,
        TestName:       t.TestName,
        Status:         t.Status,
        CreationDate:   t.CreationDate,
        TestDuration:   t.TestDuration,
        SourceAgent:    t.SourceAgent,
        TargetAgent:    t.TargetAgent,
        ThresholdName:  t.ThresholdName,
        SelectedMetric: t.SelectedMetric,
    }

    // Vérifie les valeurs valides
    if t.ThresholdValue.Valid {
        dto.ThresholdValue = t.ThresholdValue.Float64
    }

    if t.ThresholdType.Valid {
        dto.ThresholdType = t.ThresholdType.String
    }

    if t.ThresholdOperator.Valid {
        dto.ThresholdOperator = t.ThresholdOperator.String
    }

    return dto
}

/////////////////////
type TestKafkaMessage struct {
	TestID     int           `json:"test_id"`
	TestType   string        `json:"test_type"`    
	Sender     string        `json:"sender"`      
	Reflectors []string      `json:"reflectors"`
	Targets    []Target `json:"targets"`   
	Profile    *Profile      `json:"profile"`  
	Duration   time.Duration `json:"duration"`   
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
	ID   int    `json:"id"` 
    IP   string `json:"ip"`
    Port int    `json:"port"`
}

type AgentInfo struct {
    ID   int
    IP   string
    Port int
}

