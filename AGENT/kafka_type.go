package agent

import (
    "log"
    "strconv"
    "strings"
)

// Nettoie les IPs et ports, et prépare les données pour le test.
func agentGroupTestToTestConfig(agt AgentGroupTest) TestConfig {
    if len(agt.Targets) == 0 {
        log.Fatalf("❌ ERREUR: Aucun agent cible dans AgentGroupTest (TestID=%d). Targets est vide.", agt.TestID)
    }

    log.Printf("✅ agentGroupTestToTestConfig reçu: %d cibles", len(agt.Targets))

    var (
        targetIPs    []string
        cleanTargets []Target
    )

    for _, t := range agt.Targets {
        originalIP := t.IP
        log.Printf("🎯 Target brut: IP=%s, Port=%d, ID=%d", originalIP, t.Port, t.ID)

        cleanIP := originalIP
        cleanPort := t.Port

        if strings.Contains(originalIP, ":") {
            parts := strings.Split(originalIP, ":")
            cleanIP = parts[0]
            if len(parts) == 2 {
                if port, err := strconv.Atoi(parts[1]); err == nil {
                    cleanPort = port
                } else {
                    log.Printf("⚠️ Port invalide dans '%s': %v", originalIP, err)
                }
            }
        }

        targetIPs = append(targetIPs, cleanIP)
        cleanTargets = append(cleanTargets, Target{
            ID:   t.ID,
            IP:   cleanIP,
            Port: cleanPort,
        })
    }

    if len(cleanTargets) == 0 {
        log.Fatalf("❌ ERREUR: Aucun agent cible valide après nettoyage.")
    }

    firstTarget := cleanTargets[0]
    log.Printf("✅ Nettoyé: cleanTargets[0].IP = '%s', Port = %d, ID=%d", firstTarget.IP, firstTarget.Port, firstTarget.ID)
    log.Printf("🕵️ Vérification finale - TargetID utilisé pour config : %d", firstTarget.ID)

    profile := Profile{
        ID:              agt.Profile.ID,
        SendingInterval: agt.Profile.SendingInterval,
        PacketSize:      agt.Profile.PacketSize,
        PacketRate:      agt.Profile.PacketRate,
    }

    intervalMs := int(profile.SendingInterval * 1000) // si en secondes

    if intervalMs <= 0 || profile.PacketSize <= 0 {
        log.Fatalf("❌ ERREUR: Intervalle d'envoi (%d ms) ou PacketSize (%d) invalide pour le test ID=%d", intervalMs, profile.PacketSize, agt.TestID)
    }

    durationMs := int64(agt.Duration * 1000) // convertit secondes → millisecondes


    // <<--- MODIFICATION ICI
    sourcePort := agt.SenderPort
    if len(agt.Targets) > 1 {
        log.Printf("ℹ️ Plusieurs cibles détectées, port source mis à 0 pour éviter conflit de port")
        sourcePort = 0
    }

    return TestConfig{
        TestID:     agt.TestID,
        SourceIP:   agt.SenderIP,
        SourcePort: sourcePort,

        TargetIPs:  targetIPs,
        Targets:    cleanTargets,
        TargetIP:   firstTarget.IP,
        TargetPort: firstTarget.Port,
        TargetID:   firstTarget.ID,

        TestOption: agt.TestOption,

        Duration:   durationMs,

        IntervalMs: intervalMs,
        PacketSize: profile.PacketSize,
        Profile:    &profile,
    }
}



// Profile contient la configuration du profil d'envoi de paquets.
type Profile struct {
    ID              int   `json:"id"`
    SendingInterval int64 `json:"sending_interval"`
    PacketSize      int   `json:"packet_size"`
    PacketRate      int   `json:"packet_rate"`
}

// AgentGroupTest représente un test planifié pour un groupe d'agents.
type AgentGroupTest struct {
    TestID     int      `json:"test_id"`
    SenderIP   string   `json:"sender_ip"`
    SenderPort int      `json:"sender_port"`
    Targets    []Target `json:"targets"`
    Duration   int      `json:"duration"`  // durée en millisecondes ?
    TestOption string   `json:"test_option"`
    Profile    Profile  `json:"profile"`
}

// Target représente une cible d'agent avec IP et port.
type Target struct {
      ID   int
    IP   string `json:"ip"`
    Port int    `json:"port"`
}

// TestConfig contient la configuration complète d'un test.
type TestConfig struct {
    TestID         int       `json:"test_id"`
    Name           string    `json:"name"`
    Duration       int64     `json:"duration"`
    NumberOfAgents int       `json:"number_of_agents"`
    SourceID       int       `json:"source_id"`
    SourceIP       string    `json:"source_ip"`
    SourcePort     int       `json:"source_port"`
    TargetID       int       `json:"target_id"`
    TargetIP       string    `json:"target_ip"`
    TargetPort     int       `json:"target_port"`
    ProfileID      int       `json:"profile_id"`
    Profile        *Profile  `json:"profile"`
    TargetIPs      []string  `json:"target_ips,omitempty"`
    Targets        []Target  `json:"targets,omitempty"`
    TargetAgentIDs []int     `json:"target_agent_ids,omitempty"`
    TestOption     string    `json:"test_option"`
    IntervalMs     int       `json:"interval_ms"`
    PacketSize     int       `json:"packet_size"`
}
