package server


import (

	"time" 
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
