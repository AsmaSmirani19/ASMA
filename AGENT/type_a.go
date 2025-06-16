package agent


import (
	"time"	
)

type WsQoSMetrics struct { 
	Type              string  `json:"type"` // "qos_metrics"
	TestID            int     `json:"test_id"`
	TargetID          int     `json:"target_id"`
	AvgLatencyMs      float64 `json:"avg_latency_ms"`
	AvgJitterMs       float64 `json:"avg_jitter_ms"`
	AvgThroughputKbps float64 `json:"avg_throughput_kbps"`
	PacketLossPercent float64 `json:"packet_loss_percent"`
}


type WebSocketMessage struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

type PacketStats struct {
	TargetID        int
	TestID             int 
	SentPackets        int
	ReceivedPackets    int
	TotalBytesSent     int64
	TotalBytesReceived int64
	LastLatency        int64
	StartTime          time.Time
	LatencySamples     []int64
	TargetAddress      string
	TargetPort         int
}


type TwampTestPacket struct {
	SequenceNumber        uint32
	Timestamp             uint64
	ErrorEstimation       uint16
	MBZ                   uint16
	ReceptionTimestamp    uint64
	SenderSequenceNumber  uint64
	SenderTimestamp       uint64
	SenderErrorEstimation uint16
	SenderTTL             uint8
	Padding               []byte
}


 
type KafkaConfig struct {
	Brokers          []string
	TestRequestTopic string
	GroupID          string
}

// Struct pour les paramètres parsés
type TestParams struct {
	TargetIP       string
	TargetPort     int
	Duration       time.Duration
	PacketInterval time.Duration
}

type QoSMetrics struct {
	PacketLossPercent float64
	AvgLatencyMs      float64
	AvgJitterMs       float64
	AvgThroughputKbps float64
	TotalJitter       int64
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



type TestRequest struct {
	TestID int `json:"test_id"`
}

type PlannedTest struct {
	ID             int            `json:"id"`
	TestName       string         `json:"test_name"`
	TestDuration   string         `json:"test_duration"`         
	NumberOfAgents  int           `json:"number_of_agents"`
	CreationDate   time.Time      `json:"creation_date"`
	TestType        string        `json:"test_type"`
	SourceID          int         `json:"source_id"`
	TargetID          int         `json:"target_id"`
	ProfileID         int         `json:"profile_id"`
	ThresholdID       int         `json:"threshold_id"`
	InProgress     bool      `json:"inProgress"` 
	Failed         bool      `json:"failed"`
	Completed      bool      `json:"completed"`
	Error          bool      `json:"error"`
}