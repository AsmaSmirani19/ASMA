package agent

import (

)

// TestConfig représente la configuration complète du test envoyée par le serveur central
type TestConfig struct {
    TestID         int       `json:"TestID"`
    Name           string    `json:"Name"`          
   Duration        int64     `json:"Duration"`      // durée en nanosecondes
    NumberOfAgents int       `json:"NumberOfAgents"`
    SourceID       int       `json:"SourceID"`
    SourceIP       string    `json:"SourceIP"`          
    SourcePort     int       `json:"SourcePort"`        
    TargetID       int       `json:"TargetID"`
    TargetIP       string    `json:"TargetIP"`          
    TargetPort     int       `json:"TargetPort"`        
    ProfileID      int       `json:"ProfileID"`
    ThresholdID    int       `json:"ThresholdID"`
    InProgress     bool      `json:"InProgress"`        
    Failed         bool      `json:"Failed"`
    Completed      bool      `json:"Completed"`
    Error          bool      `json:"Error"`

    Profile        *Profile  `json:"Profile"`           // structure imbriquée Profile
    Threshold      *Threshold `json:"Threshold"`         // structure imbriquée Threshold
}

type Profile struct {
    ID              int     `json:"ID"`
    SendingInterval int64   `json:"SendingInterval"` // nanosecondes
    PacketSize      int     `json:"PacketSize"`
    PacketRate      int     `json:"PacketRate"`
}

type Threshold struct {
    ID              int     `json:"id"`
    Name            string  `json:"name"`
    Avg             float64 `json:"avg"`
    Min             float64 `json:"min"`
    Max             float64 `json:"max"`
    AvgStatus       bool    `json:"avg_status"`
    MinStatus       bool    `json:"min_status"`
    MaxStatus       bool    `json:"max_status"`
    AvgOpr          string  `json:"avg_opr"`
    MinOpr          string  `json:"min_opr"`
    MaxOpr          string  `json:"max_opr"`
    SelectedMetric  string  `json:"selected_metric"`
    ActiveThresholds interface{} `json:"active_thresholds"`
    DisabledThresholds interface{} `json:"disabled_thresholds"`
}