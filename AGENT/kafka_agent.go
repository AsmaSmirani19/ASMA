package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"time"
	"sync"
	"github.com/segmentio/kafka-go"
)

type TestKafkaMessage struct {
	TestID     int      `json:"test_id"`
	TestType   string   `json:"test_type"` // "agent-to-agent"
	Sender     string   `json:"sender"`    // IP source
	Reflectors []string `json:"reflectors"` // Liste IP destination
	Profile    struct {
		SendingInterval int64 `json:"sending_interval"` // en millisecondes
		PacketSize      int   `json:"packet_size"`
	} `json:"profile"`
}

// File d'attente pour les tests
var testQueue = make(chan TestConfig, 100)

// Worker qui lance les tests un par un, dans l’ordre
func testWorker(ctx context.Context, brokers []string) {
	for {
		select {
		case test := <-testQueue:
			log.Printf("▶️ Worker démarre test %d", test.TestID)
			
			switch test.TestOption {
			case "agent-to-agent":
				Client(test)
			case "agent-to-group":
				RunAgentToGroupTest(test, brokers)
			default:
				log.Printf("⚠️ Option test inconnue: %s", test.TestOption)
			}

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
	go testWorker(ctx, kafkaConfig.Brokers)

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("❌ Erreur lecture Kafka: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// 1) Essayer de décoder en AgentGroupTest (agent-to-group)
		var agt AgentGroupTest
		if err := json.Unmarshal(msg.Value, &agt); err == nil && agt.TestOption == "agent-to-group" {
			log.Printf("✅ AgentGroupTest reçu via Kafka: %+v", agt)

			config := agentGroupTestToTestConfig(agt)

			err = LaunchTest(config)
			if err != nil {
				log.Printf("⚠️ Erreur ajout file test %d : %v", config.TestID, err)
			}
			continue
		}

		// 2) Essayer de décoder en TestKafkaMessage (agent-to-agent / planned_test)
		var simpleMsg TestKafkaMessage
		if err := json.Unmarshal(msg.Value, &simpleMsg); err == nil && 
			(simpleMsg.TestType == "agent-to-agent" || simpleMsg.TestType == "agent-to-group") {

			log.Printf("✅ TestKafkaMessage simple reçu: %+v", simpleMsg)

			if len(simpleMsg.Reflectors) == 0 {
				log.Printf("❌ Aucune IP cible dans le message Kafka")
				continue
			}

			// Ici tu peux adapter selon le type de test
			config := TestConfig{
				TestID:     simpleMsg.TestID,
				TestOption: simpleMsg.TestType, // "agent-to-agent" ou "planned_test"
				SourceIP:   simpleMsg.Sender,
				TargetIP:   simpleMsg.Reflectors[0],
				TargetPort: 20000, // ou autre port selon ton besoin
				Duration:   int64((10 * time.Second) / time.Millisecond), // adapter si nécessaire
				IntervalMs: int(simpleMsg.Profile.SendingInterval / int64(time.Millisecond)),
				PacketSize: simpleMsg.Profile.PacketSize,
			}

			err = LaunchTest(config)
			if err != nil {
				log.Printf("⚠️ Erreur ajout file test %d : %v", config.TestID, err)
			}
			continue
		}

		log.Printf("⚠️ Message Kafka non reconnu ou invalide: %s", string(msg.Value))
	}
}



func RunAgentToGroupTest(test TestConfig, brokers []string) {
	log.Printf("▶️ Début test agent-to-group ID=%d", test.TestID)

	if len(test.Targets) == 0 {
		log.Printf("❌ Erreur : aucune cible fournie pour le test %d", test.TestID)
		return
	}
	log.Printf("📦 Targets à tester: %+v", test.Targets)

	var wg sync.WaitGroup

	for _, target := range test.Targets {
		wg.Add(1) // pour chaque goroutine
		 go func(target Target) {
			defer wg.Done()

			log.Printf("🚀 Lancement test vers %s:%d ...", target.IP, target.Port)

			targetConfig := test // copie la config globale
			targetConfig.TargetIP = target.IP
			targetConfig.TargetPort = target.Port
			targetConfig.SourceIP = AppConfig.Sender.IP

			err := Client(targetConfig)
			if err != nil {
				log.Printf("❌ Erreur TWAMP (UDP) avec %s:%d : %v", target.IP, target.Port, err)
			}
		}(target)
	}

	wg.Wait() // attend la fin de tous les tests
	log.Printf("🏁 Fin du test agent-to-group ID=%d", test.TestID)
}


//****************
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