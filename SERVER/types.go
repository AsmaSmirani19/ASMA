// types.go
package main

import (
	"time"
)


// Déclaration du type Agent
type Agent struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Address      string `json:"address"`
	TestHealth   bool   `json:"test_health"`
	Availability int    `json:"availability"`
}

// Déclaration du type QoSMetrics
type QoSMetrics struct {
	PacketLossPercent float64 `json:"packet_loss_percent"`
	AvgLatencyMs      int64   `json:"avg_latency_ms"`
	AvgJitterMs       int64   `json:"avg_jitter_ms"`
	AvgThroughputKbps float64 `json:"avg_throughput_kbps"`
	TotalJitter       int64   `json:"total_jitter"`
}

type plannedtest struct {
	ID             int       `json:"id"`
	TestName       string    `json:"test_name"`
	TestDuration   string    `json:"test_duration"`   
	NumberOfAgents int       `json:"number_of_agents"`
	CreationDate   time.Time `json:"creation_date"`  

	TestType     string       `json:"test_type"`          
	SourceID     int          `json:"source_id"`      
	TargetID     int          `json:"target_id"`      
	ProfileID    int          `json:"profile_id"`     
	ThresholdID  int          `json:"threshold_id"`   
	Waiting      bool         `json:"waiting"`        
	Failed       bool         `json:"failed"`         
	Completed    bool          `json:"completed"`	
}	


type agentGroup struct {
	ID             int       `json:"id"`
	GroupName     string    `json:"group_name"`
	NumberOfAgents  int       `json:"number_of_agents"`
	CreationDate   time.Time `json:"creation_date"` 
}

type testProfile struct {
	ID           int              `json:"id"`
	ProfileName  string           `json:"profile_name"`
	CreationDate time.Time        `json:"creation_date"` 
	PacketSize   int		      `json:"packet_size"`
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











