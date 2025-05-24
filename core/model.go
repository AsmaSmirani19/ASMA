package core 
import(
	"time"
)

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

// Structures

type FullTestConfiguration struct {
	TestID         int
	Name           string
	RawDuration    string
	Duration       time.Duration
	NumberOfAgents int
	SourceID       int
	SourceIP       string
	SourcePort     int
	TargetID       int
	TargetIP       string
	TargetPort     int
	ProfileID      int
	ThresholdID    int
	Profile        *Profile
	Threshold      *Threshold

	Waiting   bool
	Failed    bool
	Completed bool

}


type Profile struct {
	ID              int
	SendingInterval time.Duration
	PacketSize      int
	PacketRate      int // Remarque : dans ta table tu as "time_between_attempts" mais pas packet_rate, à voir si tu l'utilises
}

type Threshold struct {
	ID             int
	Name           string
	Avg            float64
	Min            float64
	Max            float64
	AvgStatus      string
	MinStatus      string
	AvgOpr         string
	MinOpr         string
	MaxOpr         string
	SelectedMetric string
}

type AttemptResult struct {
    TestID         int64   `json:"test_id"`        
    LatencyMs      float64 `json:"latency_ms"`
    JitterMs       float64 `json:"jitter_ms"`
    ThroughputKbps float64 `json:"throughput_kbps"`
}

type PlannedTest struct {
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

