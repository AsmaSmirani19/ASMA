package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

// AgentGroupTest est la structure envoyée pour lancer un test de groupe
type AgentGroupTest struct {
	TestID     int      `json:"test_id"`
	SenderIP   string   `json:"sender_ip"`
	SenderPort int      `json:"sender_port"`
	Targets    []Target `json:"targets"`
	Duration   int      `json:"duration"`    // en secondes, à part
	Profile    Profile  `json:"profile"`     // ici sans duration
	TestOption string   `json:"test_option"`
}


func ConvertToAgentGroupTest(config *FullTestConfiguration, db *sql.DB) (*AgentGroupTest, error) {
	if config == nil {
		return nil, fmt.Errorf("la configuration est vide")
	}
	if config.Profile == nil {
		return nil, fmt.Errorf("le profil est manquant")
	}

	var targets []Target
	for _, id := range config.TargetAgentIDs {
		var ip string
		var port int
		err := db.QueryRow(`SELECT "Address", "Port" FROM "Agent_List" WHERE id = $1`, id).Scan(&ip, &port)
		if err != nil {
			return nil, fmt.Errorf("erreur récupération IP et port pour agent ID %d : %w", id, err)
		}
		targets = append(targets, Target{IP: ip, Port: port})
	}

	agt := &AgentGroupTest{
		TestID:     config.TestID,
		SenderIP:   config.SourceIP,
		SenderPort: config.SourcePort,
		Targets:    targets,
		Duration:   int(config.Duration.Seconds()), // Duration en dehors de Profile
		Profile:    *config.Profile,                // Profile complet, sans duration dedans
		TestOption: "agent-to-group",
	}

	return agt, nil
}

// TriggerAgentToGroupTest charge la config, transforme, encode et envoie sur Kafka
func TriggerAgentToGroupTest(db *sql.DB, brokers []string, topic string, testID int) error {
	// Charger la configuration complète
	config, err := LoadFullTGroupTest(db, testID)
	if err != nil {
		return fmt.Errorf("échec chargement configuration : %w", err)
	}

	// Convertir en message test
	testMsg, err := ConvertToAgentGroupTest(config, db)
	if err != nil {
		return fmt.Errorf("erreur transformation en AgentGroupTest : %w", err)
	}

	// Encoder en JSON
	data, err := json.Marshal(testMsg)
	if err != nil {
		return fmt.Errorf("erreur encodage JSON : %w", err)
	}

	// Définir clé Kafka
	key := fmt.Sprintf("test-%d", testMsg.TestID)

	// Envoyer sur Kafka
	err = SendMessageToKafka(brokers, topic, key, string(data))
	if err != nil {
		return fmt.Errorf("erreur envoi Kafka : %w", err)
	}

	log.Printf("📤 Test ID=%d envoyé à Kafka avec succès.", testID)
	return nil
}
