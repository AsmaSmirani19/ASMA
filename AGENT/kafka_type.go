package agent
import(
	"log"
)

type AgentGroupTest struct {
	TestID     int      `json:"test_id"`
	SenderIP   string   `json:"sender_ip"`
	SenderPort int      `json:"sender_port"`
	Targets    []Target `json:"targets"`
	Duration   int      `json:"duration"`
	TestOption string   `json:"test_option"`
	Profile    Profile  `json:"profile"`
}

type Target struct {
    IP   string `json:"ip"`
    Port int    `json:"port"`
}

func agentGroupTestToTestConfig(agt AgentGroupTest) TestConfig {
    if len(agt.Targets) == 0 {
        log.Fatalf("❌ ERREUR: Aucun agent cible dans AgentGroupTest (TestID=%d). Targets est vide.", agt.TestID)
    }

    var targetIPs []string
    for _, t := range agt.Targets {
        targetIPs = append(targetIPs, t.IP)
    }

    return TestConfig{
        TestID:     agt.TestID,
        SourceIP:   agt.SenderIP,
        SourcePort: agt.SenderPort,

        TargetIPs:  targetIPs,
        Targets:    agt.Targets,

        TargetIP:   agt.Targets[0].IP,
        TargetPort: agt.Targets[0].Port,


        Duration:   int64(agt.Duration),
        TestOption: agt.TestOption,
        Profile:    &agt.Profile,
        IntervalMs: int(agt.Profile.SendingInterval / 1e6),
        PacketSize: agt.Profile.PacketSize,
    }
}



type TestConfig struct {
	TestID          int         `json:"test_id"`
    Name            string      `json:"name"`
    Duration        int64       `json:"duration"`
    NumberOfAgents  int         `json:"number_of_agents"`
    SourceID        int         `json:"source_id"`
    SourceIP        string      `json:"source_ip"`
    SourcePort      int         `json:"source_port"`
    TargetID        int         `json:"target_id"`
    TargetIP        string      `json:"target_ip"`
    TargetPort      int         `json:"target_port"`
    ProfileID       int         `json:"profile_id"`
    Profile         *Profile    `json:"profile"`
	TargetIPs       []string    `json:"target_ips,omitempty"`
	Targets         []Target `json:"targets,omitempty"`
    TargetAgentIDs  []int       `json:"target_agent_ids,omitempty"` // pour agent-to-group
    TestOption      string      `json:"test_option"`
	IntervalMs int    `json:"interval_ms"`  // ajouter ce champ
    PacketSize int    `json:"packet_size"`
	// Agents         []AgentRole  `json:"Agents"`
}


type Profile struct {
    ID              int     `json:"ID"`
    SendingInterval int64   `json:"SendingInterval"` // nanosecondes
    PacketSize      int     `json:"PacketSize"`
    PacketRate      int     `json:"PacketRate"`
}

