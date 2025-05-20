package main

import (
	"context"
	"encoding/json"
	"database/sql"  
	"log"

	"mon-projet-go/server" 

	"github.com/segmentio/kafka-go"
)

// Structure des résultats de test à envoyer au backend
type TestResult struct {
	AgentID            string   `json:"agent_id"`
	Target             string   `json:"target"`
	Port               int      `json:"port"`
	PacketLossPercent  float64  `json:"packet_loss_percent"`
	AvgLatencyMs       float64  `json:"avg_latency_ms"`
	AvgJitterMs        float64  `json:"avg_jitter_ms"`
	AvgThroughputKbps  float64  `json:"avg_throughput_Kbps"`
}

//func listenToTestRequestsFromKafka(db *sql.DB) {
	func listenToTestRequestsFromKafka(db *sql.DB) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: AppConfig.Kafka.Brokers,
		Topic:   AppConfig.Kafka.TestRequestTopic,
		GroupID: AppConfig.Kafka.GroupID,
	})
	defer reader.Close()

	for {
		message, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("❌ Erreur de lecture Kafka : %v", err)
			continue
		}

		log.Printf("📨 Message Kafka reçu : %s", message.Value)

		var testReq server.TestConfig // ✅ ici
		if err := json.Unmarshal(message.Value, &testReq); err != nil {
			log.Printf("❌ Erreur JSON : %v", err)
			continue
		}

		log.Printf("🔄 Déclenchement test avec ID : %d", testReq.TestID)

		testDetails, err := getTestDetailsByID(db, testReq.TestID)
		if err != nil {
			log.Printf("❌ Erreur récupération test : %v", err)
			continue
		}

		go server.Client(testDetails)
	}
}

// ✅ Signature corrigée ici aussi
func getTestDetailsByID(db *sql.DB, testID int) (server.TestConfig, error) {
	var config server.TestConfig
	query := `
		SELECT id, name, duration, number_of_agents, source_id, target_id, profile_id, threshold_id
		FROM test_configs
		WHERE id = $1`
	err := db.QueryRow(query, testID).Scan(
		&config.TestID,
		&config.Name,
		&config.Duration,
		&config.NumberOfAgents,
		&config.SourceID,
		&config.TargetID,
		&config.ProfileID,
		&config.ThresholdID,
	)
	return config, err
}
