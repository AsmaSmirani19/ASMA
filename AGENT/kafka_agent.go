package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// File d'attente pour les tests
var testQueue = make(chan TestConfig, 100)

// Worker qui lance les tests un par un, dans l’ordre
func testWorker(ctx context.Context, db *sql.DB) {
	for {
		select {
		case test := <-testQueue:
			log.Printf("▶️ Worker démarre test %d", test.TestID)
			Client(test, db) // fonction bloquante, exécute le test
			log.Printf("🏁 Worker termine test %d", test.TestID)
		case <-ctx.Done():
			log.Println("⚠️ Worker arrêté par contexte")
			return
		}
	}
}

// Fonction pour lancer un test (ajoute à la file d’attente)
func LaunchTest(test TestConfig) error {
	select {
	case testQueue <- test:
		log.Printf("📥 Test %d ajouté à la file d’attente", test.TestID)
		return nil
	default:
		return errors.New("⚠️ file d’attente pleine, veuillez réessayer plus tard")
	}
}

// Kafka écouteur (modifié pour uniquement mettre en file)
// Tu peux recevoir la config via Kafka, mais pas lancer direct Client
func ListenToTestRequestsFromKafka(db *sql.DB) {
	kafkaConfig := KafkaConfig{
		Brokers:          []string{"localhost:9092"},
		TestRequestTopic: "test-requests",
		GroupID:          "agent-group-debug",
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: kafkaConfig.Brokers,
		Topic:   kafkaConfig.TestRequestTopic,
		GroupID: kafkaConfig.GroupID,
	})
	defer reader.Close()

	ctx := context.Background()

	// Démarrer le worker (exécution tests un par un)
	go testWorker(ctx, db)

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("❌ Erreur lecture Kafka: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		var config TestConfig
		if err := json.Unmarshal(msg.Value, &config); err != nil {
			log.Printf("❌ Erreur décodage JSON: %v", err)
			continue
		}

		log.Printf("✅ TestConfig reçu via Kafka: %+v", config)

		// Ajouter le test à la file, il sera lancé séquentiellement par le worker
		err = LaunchTest(config)
		if err != nil {
			log.Printf("⚠️ Erreur ajout file test %d : %v", config.TestID, err)
		}
	}
}



type TestResult1 struct {
	TestID         int     `json:"test_id"`
	LatencyMs      float64 `json:"latency_ms"`
	JitterMs       float64 `json:"jitter_ms"`
	ThroughputKbps float64 `json:"throughput_kbps"`
}

func sendTestResultKafka(brokers []string, topic string, result TestResult1) error {
	log.Printf("📤 Envoi résultat Kafka pour TestID %d...", result.TestID)

	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer func() {
		if err := writer.Close(); err != nil {
			log.Printf("⚠️ Erreur fermeture writer Kafka: %v", err)
		}
	}()

	data, err := json.Marshal(result)
	if err != nil {
		log.Printf("❌ Erreur encodage JSON du résultat pour TestID %d: %v", result.TestID, err)
		return err
	}

	msg := kafka.Message{Value: data}

	err = writer.WriteMessages(context.Background(), msg)
	if err != nil {
		log.Printf("❌ Erreur envoi résultat Kafka pour TestID %d: %v", result.TestID, err)
		return err
	}

	log.Printf("✅ Résultat Kafka envoyé avec succès pour TestID %d", result.TestID)
	return nil
}
