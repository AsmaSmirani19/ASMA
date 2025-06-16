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
		log.Printf("❌ Profil manquant dans FullTestConfiguration pour TestID=%d", config.TestID)
		return nil, fmt.Errorf("le profil est manquant")
	}

	log.Printf("🔍 DEBUG Profil brut avant copie pour TestID=%d : %+v", config.TestID, config.Profile)

	var targets []Target
	for _, id := range config.TargetAgentIDs {
		var ip string
		var port int
		err := db.QueryRow(`SELECT "Address", "Port" FROM "Agent_List" WHERE id = $1`, id).Scan(&ip, &port)
		if err != nil {
			log.Printf("❌ Erreur récupération IP/Port agent cible ID=%d : %v", id, err)
			return nil, fmt.Errorf("erreur récupération IP et port pour agent ID %d : %w", id, err)
		}
		log.Printf("📡 Agent cible ID=%d : IP=%s, Port=%d", id, ip, port)

		target := Target{
			ID:   id,   // ✅ Ajout de l’ID ici
			IP:   ip,
			Port: port,
		}
		targets = append(targets, target)
	}

	agt := &AgentGroupTest{
		TestID:     config.TestID,
		SenderIP:   config.SourceIP,
		SenderPort: config.SourcePort,
		Targets:    targets,
		Duration:   int(config.Duration.Seconds()), // Duration en dehors de Profile
		Profile:    *config.Profile,                // ⚠️ Copie par valeur
		TestOption: "agent-to-group",
	}

	log.Printf("✅ AgentGroupTest construit : TestID=%d, Targets=%d, Duration=%ds", agt.TestID, len(agt.Targets), agt.Duration)
	log.Printf("🧪 Profil copié dans AgentGroupTest : %+v", agt.Profile)

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

	log.Printf("DEBUG testMsg.Targets = %+v", testMsg.Targets)
	log.Printf("🔍 DEBUG Profil transmis dans testMsg (TestID=%d) : %+v", testMsg.TestID, testMsg.Profile)

	// (Optionnel) Vérification explicite de champs critiques
	if testMsg.Profile.SendingInterval == 0 || testMsg.Profile.PacketSize == 0 {
		log.Printf("⚠️ WARNING: Le profil semble incomplet ou invalide ! %+v", testMsg.Profile)
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
