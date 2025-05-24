package server

import (
	"encoding/json" 
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// --- WebSocket Upgrader ---

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// --- WebSocket Handler ---

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("❌ Erreur de connexion WebSocket:", err)
		return
	}
	defer conn.Close()

	log.Println("✅ Connexion établie avec l'agent via WebSocket")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("🔌 Erreur de lecture ou connexion fermée :", err)
			break
		}

		log.Printf("📨 Résultat reçu de l'agent : %s\n", string(msg))

		var rawMsg map[string]json.RawMessage
		if err := json.Unmarshal(msg, &rawMsg); err != nil {
			log.Println("❌ Erreur de parsing JSON brut :", err)
			continue
		}

		var msgType string
		if err := json.Unmarshal(rawMsg["type"], &msgType); err != nil {
			log.Println("❌ Erreur de lecture du type :", err)
			continue
		}

		switch msgType {
		case "status":
			var status TestStatus
			if err := json.Unmarshal(rawMsg["payload"], &status); err != nil {
				log.Println("❌ Erreur de parsing du TestStatus :", err)
				continue
			}
			log.Printf("📊 Test ID %d ➤ Status: %s\n", status.TestID, status.Status)

		case "metrics":
			var metrics AttemptResult
			if err := json.Unmarshal(rawMsg["payload"], &metrics); err != nil {
				log.Println("❌ Erreur de parsing QoSMetrics :", err)
				continue
			}
			log.Printf("📈 Metrics Test ID %d ➤ Latency: %.2fms, Jitter: %.2fms, Bandwidth: %.2fMbps\n",
				metrics.TestID, metrics.LatencyMs, metrics.JitterMs, metrics.ThroughputKbps)

		default:
			log.Println("⚠️ Type de message inconnu :", msgType)
		}

		// Message de confirmation à l'agent
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Résultat reçu avec succès")); err != nil {
			log.Println("🔴 Erreur lors de l'envoi de message de confirmation à l'agent :", err)
			break
		}
	}
}

// --- Fonction pour lancer le serveur WebSocket ---

func StartWebSocketServer() {
	http.HandleFunc("/ws", handleWebSocket)

	addr := fmt.Sprintf("%s:%d", AppConfig.WebSocket.Address, AppConfig.WebSocket.Port)
	log.Printf("🚀 Serveur WebSocket lancé sur %s...", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Erreur WebSocket: %v", err)
	}
}
