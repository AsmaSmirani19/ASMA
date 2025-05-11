package main

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

// Fonction pour envoyer une demande de test via Kafka
func SendTestRequestToKafka(testCommand string) {
	// Création du writer Kafka (producteur)
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  AppConfig.Kafka.Brokers,
		Topic:    AppConfig.Kafka.TestRequestTopic,
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	// Envoi du message Kafka (demande de test)
	err := writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte("test-command"),
			Value: []byte(testCommand), // Commande de test
		},
	)
	if err != nil {
		log.Printf("Erreur lors de l'envoi de la demande de test : %v", err)
		return
	}
	log.Println("Demande de test envoyée au Kafka.")
}
