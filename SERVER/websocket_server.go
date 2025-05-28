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

		// On décode en map[string]interface{} pour vérifier la présence de "type"
		var msgMap map[string]interface{}
		if err := json.Unmarshal(msg, &msgMap); err != nil {
			log.Println("❌ Erreur de parsing JSON :", err)
			continue
		}

		// On check si "type" est présent
		if t, ok := msgMap["type"]; ok {
			// cas "type" présent : on peut utiliser ta logique initiale
			typeStr, ok := t.(string)
			if !ok {
				log.Println("❌ Type non string dans message")
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
                log.Printf("📊 Test ID %d ➤ Status: %s\n", status.TestID, status.Status)

                // Variables pour update selon status reçu
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

                // Appel à ta fonction d'update
                if err := UpdateTestStatus(db, status.TestID, inProgress, failed, completed, errorFlag); err != nil {
                    log.Printf("❌ Erreur mise à jour statut TestID %d : %v", status.TestID, err)
                } else {
                    log.Printf("✅ Statut mis à jour en BDD pour TestID %d", status.TestID)
                }

			case "metrics":
				var rawMsg struct {
					Payload AttemptResult `json:"payload"`
				}
				if err := json.Unmarshal(msg, &rawMsg); err != nil {
					log.Println("❌ Erreur parsing metrics :", err)
					continue
				}
				metrics := rawMsg.Payload
				log.Printf("📈 Metrics Test ID %d ➤ Latency: %.2fms, Jitter: %.2fms, Bandwidth: %.2fMbps\n",
					metrics.TestID, metrics.LatencyMs, metrics.JitterMs, metrics.ThroughputKbps)

				if err := SaveAttemptResult(db, metrics.TestID, metrics.LatencyMs, metrics.JitterMs, metrics.ThroughputKbps); err != nil {
					log.Printf("❌ Erreur insertion base pour TestID %d : %v", metrics.TestID, err)
				} else {
					log.Printf("✅ Données enregistrées pour TestID %d", metrics.TestID)
				}

			default:
				log.Println("⚠️ Type inconnu :", typeStr)
			}
		} else {
			// Pas de "type" dans le JSON : on essaye de parser en AttemptResult direct
			var metrics AttemptResult
			if err := json.Unmarshal(msg, &metrics); err != nil {
				log.Println("❌ Erreur parsing métrics sans type :", err)
				continue
			}
			log.Printf("📈 Metrics (sans type) Test ID %d ➤ Latency: %.2fms, Jitter: %.2fms, Bandwidth: %.2fMbps\n",
				metrics.TestID, metrics.LatencyMs, metrics.JitterMs, metrics.ThroughputKbps)

			if err := SaveAttemptResult(db, metrics.TestID, metrics.LatencyMs, metrics.JitterMs, metrics.ThroughputKbps); err != nil {
				log.Printf("❌ Erreur insertion base pour TestID %d : %v", metrics.TestID, err)
			} else {
				log.Printf("✅ Données enregistrées pour TestID %d", metrics.TestID)
			}
		}

		if err := conn.WriteMessage(websocket.TextMessage, []byte("Résultat reçu avec succès")); err != nil {
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
