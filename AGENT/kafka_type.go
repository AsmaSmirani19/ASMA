package agent
import(
	"log"
    "strings"
    "time"
)



func agentGroupTestToTestConfig(agt AgentGroupTest) TestConfig {
    if len(agt.Targets) == 0 {
        log.Fatalf("❌ ERREUR: Aucun agent cible dans AgentGroupTest (TestID=%d). Targets est vide.", agt.TestID)
    }

    var targetIPs []string
    for i, t := range agt.Targets {
        // Nettoyer le champ IP s’il contient un port (ex: "127.0.0.1:8081")
        cleanIP := strings.Split(t.IP, ":")[0]
        agt.Targets[i].IP = cleanIP           // Met à jour l'IP nettoyée dans la structure
        targetIPs = append(targetIPs, cleanIP)
    }

    log.Printf("DEBUG agt.Targets[0].IP = '%s', Port = %d", agt.Targets[0].IP, agt.Targets[0].Port)

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
        IntervalMs: int(agt.Profile.SendingInterval / int64(time.Millisecond)),
        PacketSize: agt.Profile.PacketSize,
    }
}




type AgentGroupTest struct {
	TestID         int          `json:"test_id"`
	SenderIP      string        `json:"sender_ip"`
	SenderPort     int          `json:"sender_port"`
	Targets      []Target       `json:"targets"`
	Duration       int          `json:"duration"`
	TestOption   string         `json:"test_option"`
	Profile      Profile        `json:"profile"`
}

type Target struct {
    IP          string         `json:"ip"`
    Port        int            `json:"port"`
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
	Targets         []Target    `json:"targets,omitempty"`
    TargetAgentIDs  []int       `json:"target_agent_ids,omitempty"` 
    TestOption      string      `json:"test_option"`
	IntervalMs int              `json:"interval_ms"`  
    PacketSize int              `json:"packet_size"`
}


type Profile struct {
    ID              int           `json:"ID"`
    SendingInterval int64         `json:"SendingInterval"` // nanosecondes
    PacketSize      int           `json:"PacketSize"`
    PacketRate      int           `json:"PacketRate"`
}
