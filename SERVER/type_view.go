package server

import (
	"time" 
    "github.com/lib/pq"
)

type PlannedTestInfo struct {
    TestID         int           `json:"testID"`
    TestName       string        `json:"testName"`
    TestDuration   string        `json:"testDuration"`
    CreationDate   time.Time     `json:"creationDate"`
    SourceAgent    string        `json:"sourceAgent"`
    TargetAgent    string        `json:"targetAgent"`
    ThresholdName  string        `json:"thresholdName"`
    ThresholdValue float64        `json:"thresholdValue"`
    ProfileName    string         `json:"profileName"`
}
type Agent struct {
	ID           int                 `json:"id"          db:"id"`
	Port         int  
	Name         string              `json:"name"        db:"Name"`
	Address      string              `json:"address"     db:"Address"`
	TestHealth   bool                `json:"testhealth"  db:"Test_health"`
}

type agentGroup struct {
	ID             int               `json:"id"`
	GroupName     string             `json:"group_name"`
	NumberOfAgents  int             `json:"number_of_agents"`
	CreationDate   time.Time         `json:"creation_date"` 
	AgentIDs       pq.Int64Array     `json:"agent_ids"`
}

type AgentHealthCheck struct {
    ID        int       `db:"id"`         
    AgentID   int       `db:"agent_id"`
    Timestamp time.Time `db:"timestamp"`
    Status    string    `db:"status"`
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
	InProgress  bool         `json:"waiting"`        
	Failed       bool         `json:"failed"`         
	Completed    bool          `json:"completed"`	
	Error        bool            `json:"Error"`
	TargetAgentIDs []int `json:"target_ids,omitempty"`

}