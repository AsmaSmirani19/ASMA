package agent


import (
	"time"
	"context"
	"sync"

	"mon-projet-go/testpb"
)


type TestStatus struct {
    TestID int    `json:"test_id"`
    Status string `json:"status"`
}

type WebSocketMessage struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

type PacketStats struct {
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

type Threshold struct {
	ID                  int
	LatencyThreshold    float64
	JitterThreshold     float64
	PacketLossThreshold float64
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

type twampAgent struct {
	testpb.UnimplementedTestServiceServer
	mu                sync.Mutex
	currentTestCancel context.CancelFunc
}

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

type KafkaConfig struct {
	Brokers          []string
	TestRequestTopic string
	GroupID          string
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
	Waiting           bool        `json:"waiting"`
	Failed            bool        `json:"failed"`
	Completed         bool        `json:"completed"`
}
