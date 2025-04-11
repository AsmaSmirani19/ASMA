package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// Fonction qui gère la connexion WebSocket et l'envoi des résultats
func StartWebSocketAgent() {
	// Connexion WebSocket au serveur
	ws, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatalf("Erreur WebSocket: %v", err)
	}
	defer ws.Close()

	// Envoi des résultats en continu
	for {
		// Récupérer les dernières métriques calculées par l'agent
		metrics := GetLatestMetrics()

		// Vérifier si des métriques sont disponibles
		if metrics != nil {
			// Sérialiser en JSON
			data, err := json.Marshal(metrics)
			if err != nil {
				log.Printf("Erreur de sérialisation JSON: %v", err)
				continue
			}

			// Envoyer les données au serveur via WebSocket
			err = ws.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Printf("Erreur d'envoi WebSocket: %v", err)
				break
			}

			fmt.Println("Métriques QoS envoyées au serveur via WebSocket:", string(data))
		} else {
			fmt.Println("Aucune métrique disponible pour l'instant.")
		}

		// Attendre avant le prochain envoi
		time.Sleep(5 * time.Second)
	}

	// Nettoyage à la fin
	err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Fin de communication"))
	if err != nil {
		log.Printf("Erreur lors de la fermeture de la connexion WebSocket: %v", err)
	}
}
