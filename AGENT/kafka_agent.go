package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Structures

type TestRequest struct {
	TestID int `json:"test_id"`
}

type PlannedTest struct {
	ID             int            `json:"id"`
	TestName       string         `json:"test_name"`
	TestDuration   string         `json:"test_duration"`         
	NumberOfAgents  int           `json:"number_of_agents"`
	CreationDate   time.Time      `json:"creation_date"`
	TestType        string        `json:"test_type"`
	SourceID          int         `json:"source_id"`
	TargetID          int         `json:"target_id"`
	ProfileID         int         `json:"profile_id"`
	ThresholdID       int         `json:"threshold_id"`
	Waiting           bool        `json:"waiting"`
	Failed            bool        `json:"failed"`
	Completed         bool        `json:"completed"`
}


type KafkaConfig struct {
	Brokers          []string
	TestRequestTopic string
	GroupID          string
}

type TestHandler func(config TestConfig)
func ListenToTestRequestsFromKafka(db *sql.DB, handler func(TestConfig)) {
	// Configuration Kafka
	kafkaConfig := KafkaConfig{
		Brokers:          []string{"localhost:9092"},
		TestRequestTopic: "test-requests",
		GroupID:          "agent-group",
	}

	// Initialisation du lecteur Kafka
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: kafkaConfig.Brokers,
		Topic:   kafkaConfig.TestRequestTopic,
		GroupID: kafkaConfig.GroupID,
	})
	defer reader.Close()

	for {
		// Lire un message Kafka
		message, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Erreur Kafka : %v", err)
			continue
		}

		// Décoder le message JSON → `TestRequest` (qui contient juste l’ID du test)
		var testReq TestRequest
		if err := json.Unmarshal(message.Value, &testReq); err != nil {
			log.Printf("Erreur JSON : %v", err)
			continue
		}

		// Aller chercher les détails du test dans la base via l’ID
		testDetails, err := getPlannedTestByID(db, testReq.TestID)
		if err != nil {
			log.Printf("Erreur DB : %v", err)
			continue
		}

		layout := "15:04:05"
		parsedDuration, err := time.Parse(layout, testDetails.TestDuration)
				if err != nil {
					log.Printf("Erreur conversion durée : %v", err)
					continue
				}
				durationSeconds := parsedDuration.Hour()*3600 + parsedDuration.Minute()*60 + parsedDuration.Second()
				duration := time.Duration(durationSeconds) * time.Second

				config := TestConfig{
					TestID:         testDetails.ID,
					Name:           testDetails.TestName,
					Duration:       duration.String(), // ex: "4s"
					NumberOfAgents: testDetails.NumberOfAgents,
					SourceID:       testDetails.SourceID,
					TargetID:       testDetails.TargetID,
					ProfileID:      testDetails.ProfileID,
					ThresholdID:    testDetails.ThresholdID,
                }
		// Appeler le handler (fourni depuis `main.go`)
		handler(config)
	}
}


func getPlannedTestByID(db *sql.DB, testID int) (PlannedTest, error) {
	var t PlannedTest
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