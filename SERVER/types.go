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
}

type agentGroup struct {
	ID             int       `json:"id"`
	GroupName     string    `json:"group_name"`
	NumberOfAgents  int64     `json:"number_of_agents"`
	CreationDate   time.Time `json:"creation_date"` // type Timestamp with Time Zone
}


