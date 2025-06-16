package server

import (
	"database/sql"
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
func handleWebSocket(db *sql.DB, w http.ResponseWriter, r *http.Request) {
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

		if !json.Valid(msg) {
			log.Println("⚠️ Message reçu non JSON valide, ignoré :", string(msg))
			continue
		}

		var msgMap map[string]interface{}
		if err := json.Unmarshal(msg, &msgMap); err != nil {
			log.Println("❌ Erreur de parsing JSON :", err)
			continue
		}

		// Cas avec champ "type"
		if t, ok := msgMap["type"]; ok {
			typeStr, ok := t.(string)
			if !ok {
				log.Println("❌ Champ 'type' non string, ignoré")
				continue
			}

			switch typeStr {
			case "status":
				var rawMsg struct {
					Payload TestStatus `json:"payload"`
				}
				if err := json.Unmarshal(msg, &rawMsg); err != nil {
					log.Println("❌ Erreur parsing status :", err)
					continue
				}
				status := rawMsg.Payload
				log.Printf("📊 Test ID %d ➤ Status: %s", status.TestID, status.Status)

				var inProgress, failed, completed, errorFlag bool
				switch status.Status {
				case "In progress", "running":
					inProgress = true
				case "failed":
					failed = true
					errorFlag = true
				case "completed", "finished":
					completed = true
				default:
					log.Printf("⚠️ Statut inconnu reçu : %s", status.Status)
				}

				if err := UpdateTestStatus(db, status.TestID, inProgress, failed, completed, errorFlag); err != nil {
					log.Printf("❌ Erreur mise à jour statut TestID %d : %v", status.TestID, err)
				} else {
					log.Printf("✅ Statut mis à jour en BDD pour TestID %d", status.TestID)
				}

			case "metrics":
				var metrics AttemptResult
				if err := json.Unmarshal(msg, &metrics); err != nil {
					log.Println("❌ Erreur parsing metrics :", err)
					continue
				}
				log.Printf("📈 Metrics Test ID %d, Target ID %d ➤ Latency: %.2fms, Jitter: %.2fms, Bandwidth: %.2fKbps",
					metrics.TestID, metrics.TargetID, metrics.LatencyMs, metrics.JitterMs, metrics.ThroughputKbps)

				if err := SaveAttemptResult(db, metrics.TestID, metrics.TargetID, metrics.LatencyMs, metrics.JitterMs, metrics.ThroughputKbps); err != nil {
					log.Printf("❌ Erreur insertion base pour TestID %d (TargetID %d) : %v", metrics.TestID, metrics.TargetID, err)
				} else {
					log.Printf("✅ Données enregistrées pour TestID %d (TargetID %d)", metrics.TestID, metrics.TargetID)
				}

		    case "qos_metrics":
				var qos QoSMetrics
				if err := json.Unmarshal(msg, &qos); err != nil {
					log.Println("❌ Erreur parsing qos_metrics :", err)
					continue
				}
				log.Printf("📊 QoS Agrégées ➤ TestID: %d, TargetID: %d ➤ Latency: %.2fms, Jitter: %.2fms, Throughput: %.2fKbps, PacketLoss: %.2f%%",
					qos.TestID, qos.TargetID, qos.AvgLatencyMs, qos.AvgJitterMs, qos.AvgThroughputKbps, qos.PacketLossPercent)

				if err := saveResultsToDB(db, qos); err != nil {
					log.Printf("❌ Erreur enregistrement QoS agrégées : %v", err)
				} else {
					log.Println("✅ QoS agrégées enregistrées avec succès dans la base de données")
				}


				default:
					log.Println("⚠️ Type inconnu reçu :", typeStr)
				}
		} else {
			// Fallback : tentative de parsing en metrics simples (ex: pour compatibilité)
			var metrics AttemptResult
			if err := json.Unmarshal(msg, &metrics); err != nil {
				log.Println("❌ Erreur parsing metrics sans type :", err)
				continue
			}
			log.Printf("📈 Metrics (sans type) Test ID %d, Target ID %d ➤ Latency: %.2fms, Jitter: %.2fms, Bandwidth: %.2fKbps",
				metrics.TestID, metrics.TargetID, metrics.LatencyMs, metrics.JitterMs, metrics.ThroughputKbps)

			if err := SaveAttemptResult(db, metrics.TestID, metrics.TargetID, metrics.LatencyMs, metrics.JitterMs, metrics.ThroughputKbps); err != nil {
				log.Printf("❌ Erreur insertion base pour TestID %d (TargetID %d) : %v", metrics.TestID, metrics.TargetID, err)
			} else {
				log.Printf("✅ Données enregistrées pour TestID %d (TargetID %d)", metrics.TestID, metrics.TargetID)
			}
		}

		// Envoi de confirmation
		if err := conn.WriteMessage(websocket.TextMessage, []byte("✅ Résultat reçu avec succès")); err != nil {
			log.Println("🔴 Erreur envoi confirmation :", err)
			break
		}
	}
}


// --- Fonction pour lancer le serveur WebSocket ---

func StartWebSocketServer(db *sql.DB) {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(db, w, r)
	})

	addr := fmt.Sprintf("%s:%d", AppConfig.WebSocket.Address, AppConfig.WebSocket.Port)
	log.Printf("🚀 Serveur WebSocket lancé sur %s...", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Erreur WebSocket: %v", err)
	}
}
