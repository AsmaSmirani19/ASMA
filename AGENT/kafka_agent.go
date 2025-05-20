package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"mon-projet-go/server"

	"github.com/segmentio/kafka-go"
)

type TestRequest struct {
	TestID int `json:"test_id"`
}

type TestResult struct {
	AgentID           string  `json:"agent_id"`
	Target            string  `json:"target"`
	Port              int     `json:"port"`
	PacketLossPercent float64 `json:"packet_loss_percent"`
	AvgLatencyMs      float64 `json:"avg_latency_ms"`
	AvgJitterMs       float64 `json:"avg_jitter_ms"`
	AvgThroughputKbps float64 `json:"avg_throughput_Kbps"`
}

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

		var testReq TestRequest
		if err := json.Unmarshal(message.Value, &testReq); err != nil {
			log.Printf("❌ Erreur JSON : %v", err)
			continue
		}

		log.Printf("🔄 Déclenchement test avec ID : %d", testReq.TestID)

		testDetails, err := getPlannedTestByID(db, testReq.TestID)
		if err != nil {
			log.Printf("❌ Erreur récupération test : %v", err)
			continue
		}

		testConfig := server.TestConfig{
		TestID:         testDetails.ID,
		Name:           testDetails.TestName,
		Duration:       testDetails.TestDuration,
		NumberOfAgents: testDetails.NumberOfAgents,
		SourceID:       testDetails.SourceID,
		TargetID:       testDetails.TargetID,
		ProfileID:      testDetails.ProfileID,
		ThresholdID:    testDetails.ThresholdID,
	}

		go server.Client(testConfig)
	}
}

func getPlannedTestByID(db *sql.DB, testID int) (server.PlannedTest, error) {
	var t server.PlannedTest
	query := `
		SELECT 
			"Id", 
			"test_name", 
			"test_duration", 
			"number_of_agents", 
			"creation_date", 
			"test_type",
			"source_id",
			"target_id",
			"profile_id",
			"threshold_id",
			"waiting",
			"failed",
			"completed"
		FROM "test"
		WHERE "Id" = $1 AND "test_type" = 'planned_test'
	`
	err := db.QueryRow(query, testID).Scan(
		&t.ID,
		&t.TestName,
		&t.TestDuration,
		&t.NumberOfAgents,
		&t.CreationDate,
		&t.TestType,
		&t.SourceID,
		&t.TargetID,
		&t.ProfileID,
		&t.ThresholdID,
		&t.Waiting,
		&t.Failed,
		&t.Completed,
	)
	return t, err
}
