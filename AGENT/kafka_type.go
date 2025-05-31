package agent

import (

)

// TestConfig représente la configuration complète du test envoyée par le serveur central
type TestConfig struct {
	TestID          int         `json:"test_id"`
	Name           string       `json:"name"`
	Duration       int64        `json:"duration"`
	NumberOfAgents int          `json:"number_of_agents"`
	SourceID       int          `json:"source_id"`
	SourceIP       string       `json:"source_ip"`
	SourcePort     int          `json:"source_port"`
	TargetID       int          `json:"target_id"`
	TargetIP       string       `json:"target_ip"`
	TargetPort     int          `json:"target_port"`
	ProfileID      int          `json:"profile_id"`
	Profile        *Profile     `json:"profile"`
	// TargetAgentIDs []int        `json:"target_ids,omitempty"`
	// Agents         []AgentRole  `json:"Agents"`
}

type AgentRole struct {
	ID        int      `json:"id"`
	Role      string   `json:"role"`
	TargetIDs []int    `json:"target_ids,omitempty"`
	IP        string   `json:"ip"`   // Ajout nécessaire
	Port      int      `json:"port"` // Ajout nécessaire
}


type Profile struct {
    ID              int     `json:"ID"`
    SendingInterval int64   `json:"SendingInterval"` // nanosecondes
    PacketSize      int     `json:"PacketSize"`
    PacketRate      int     `json:"PacketRate"`
}

type Threshold struct {
	ID               int         `json:"id"`
	Name             string      `json:"name"`
	Avg              float64     `json:"avg"`
	Min              float64     `json:"min"`
	Max              float64     `json:"max"`
	AvgStatus        bool        `json:"avg_status"`
	MinStatus        bool        `json:"min_status"`
	MaxStatus        bool        `json:"max_status"`
	AvgOpr           string      `json:"avg_opr"`
	MinOpr           string      `json:"min_opr"`
	MaxOpr           string      `json:"max_opr"`
	SelectedMetric   string      `json:"selected_metric"`
	ActiveThresholds interface{} `json:"active_thresholds"`
	DisabledThresholds interface{} `json:"disabled_thresholds"`
}