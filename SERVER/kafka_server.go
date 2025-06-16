package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
)

// Envoie un message à Kafka
func SendMessageToKafka(brokers []string, topic, key, value string) error {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	msg := kafka.Message{
		Key:   []byte(key),
		Value: []byte(value),
	}
	return writer.WriteMessages(context.Background(), msg)
}

func TriggerTestToKafka(db *sql.DB, testID int) error {
	// Charger la configuration complète du test depuis la BDD
	config, err := LoadFullTestConfiguration(db, testID)
	if err != nil {
		return fmt.Errorf("❌ Erreur chargement config test : %v", err)
	}
	log.Printf("🔍 DEBUG Config chargée depuis BDD : %+v", config)

	if config.Profile == nil {
		return fmt.Errorf("❌ Erreur : config.Profile est nil pour test %d", testID)
	}

	// Vérifie que TargetAgents contient des données
	if len(config.TargetAgents) == 0 {
		return fmt.Errorf("❌ Erreur : Aucun agent cible dans config.TargetAgents")
	}

	// Créer la liste des reflectors (IP:port)
	var reflectors []string
	for _, agent := range config.TargetAgents {
		reflectors = append(reflectors, fmt.Sprintf("%s:%d", agent.IP, agent.Port))
	}


	var targets []Target
	for _, agent := range config.TargetAgents {
		targets = append(targets, Target{
			ID:   agent.ID,
			IP:   agent.IP,
			Port: agent.Port,
		})
	}
	
	// Construire le message à envoyer
	msg := TestKafkaMessage{
		TestID:     config.TestID,
		TestType:   config.TestType,
		Sender:     fmt.Sprintf("%s:%d", config.SourceIP, config.SourcePort),
		Reflectors: reflectors,
		Targets:  targets,
		Profile:    config.Profile,
		Duration:   config.Duration, 
	}

	// Encoder le message
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("❌ Erreur JSON config : %v", err)
	}

	key := fmt.Sprintf("test-%d", testID)

	err = SendMessageToKafka([]string{"localhost:9092"}, "test-requests", key, string(data))
	if err != nil {
		return fmt.Errorf("❌ Erreur envoi Kafka : %v", err)
	}

	log.Printf("✅ Test %d envoyé à Kafka avec succès", testID)
	return nil
}

type TestResult1 struct {
    TestID         int     `json:"test_id"`
	TargetID       int64   `json:"target_id"` 
    LatencyMs      float64 `json:"latency_ms"`
    JitterMs       float64 `json:"jitter_ms"`
    ThroughputKbps float64 `json:"throughput_kbps"`
}

// db est ta connexion globale ou passée en paramètre à la fonction
var db *sql.DB

func ConsumeTestResults(ctx context.Context, brokers []string, topic string, groupID string, db *sql.DB) {
    // Création d'un reader Kafka (consommateur)
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  brokers,
        GroupID:  groupID,
        Topic:    topic,
        MinBytes: 10e3, // 10KB
        MaxBytes: 10e6, // 10MB
    })
    defer r.Close()

    log.Printf("👂 Démarrage de la consommation Kafka sur le topic %s", topic)

     for {
        m, err := r.ReadMessage(ctx)
        if err != nil {
            log.Printf("❌ Erreur lecture message Kafka : %v", err)
            if ctx.Err() != nil {
                // Contexte annulé, sortie propre
                break
            }
            continue
        }

        log.Printf("📩 Message reçu - Partition:%d Offset:%d Key:%s", m.Partition, m.Offset, string(m.Key))

        var result TestResult1
        if err := json.Unmarshal(m.Value, &result); err != nil {
            log.Printf("❌ Erreur désérialisation JSON : %v", err)
            continue
        }

       // Sauvegarde dans la base
			if err := SaveAttemptResult(db, int64(result.TestID), result.TargetID, result.LatencyMs, result.JitterMs, result.ThroughputKbps); err != nil {
		log.Printf("❌ Erreur sauvegarde en base : %v", err)
	} else {
		log.Printf("✅ Résultat TestID %d (target %d) sauvegardé en base", result.TestID, result.TargetID)
	}


    }
}