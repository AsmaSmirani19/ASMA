package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

// Structure des résultats de test à envoyer au backend
type TestResult struct {
	AgentID           string  `json:"agent_id"`
	Target            string  `json:"target"`
	Port              int     `json:"port"`
	PacketLossPercent float64 `json:"packet_loss_percent"`
	AvgLatencyMs      int64   `json:"avg_latency_ms"`
	AvgJitterMs       int64   `json:"avg_jitter_ms"`
	AvgThroughputKbps float64 `json:"avg_throughput_Kbps"`
}

// Fonction qui écoute les demandes de test depuis Kafka
func listenToTestRequestsFromKafka() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"127.0.0.1:9092"},
		Topic:   "test-requests",
		GroupID: "test-group",
	})
	defer reader.Close()

	for {
		message, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Erreur lors de la lecture du message Kafka : %v", err)
			continue
		}

		log.Printf("Message reçu : %s", message.Value)

		if string(message.Value) == "START-TEST" {
			runTestAndSendResult()
		} else {
			log.Printf("Commande non reconnue : %s", message.Value)
		}
	}
}

// Fonction pour exécuter un test et envoyer le résultat via Kafka
func runTestAndSendResult() {
	log.Println("Début du test QoS...")

	ctx := context.Background()
	params := "target=127.0.0.1&port=9000&duration=10s&interval=1s"

	stats, qos, err := startTest(params)
	if err != nil {
		log.Printf("Erreur pendant le test : %v", err)
		return
	}

	log.Println("Test terminé.")
	log.Printf("Envoyés: %d | Reçus: %d", stats.SentPackets, stats.ReceivedPackets)
	log.Printf("Latence moyenne: %d ms", qos.AvgLatencyMs)
	log.Printf("Jitter moyen: %d ms", qos.AvgJitterMs)

	// Construction de l'objet de résultat
	result := TestResult{
		AgentID:           "agent-001", // tu peux le rendre dynamique si t'as plusieurs agents
		Target:            "127.0.0.1",
		Port:              9000,
		AvgThroughputKbps: qos.AvgThroughputKbps, // si tu as ce champ
		AvgLatencyMs:      qos.AvgLatencyMs,
		AvgJitterMs:       qos.AvgJitterMs,
		PacketLossPercent: qos.PacketLossPercent, // si tu calcules ça

	}

	// Sérialisation en JSON
	resultBytes, err := json.Marshal(result)
	if err != nil {
		log.Printf("Erreur lors de la sérialisation du résultat : %v", err)
		return
	}

	// Envoi via Kafka
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{"127.0.0.1:9092"},
		Topic:    "test-results",
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte("test-result"),
		Value: resultBytes,
	})
	if err != nil {
		log.Printf("Erreur lors de l'envoi du résultat via Kafka : %v", err)
		return
	}

	log.Println("Résultat du test envoyé au backend via Kafka.")
}
