package agent 

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func StartWebSocketAgent() {
	var ws *websocket.Conn
	var err error

	for {
		ws, _, err = websocket.DefaultDialer.Dial(AppConfig.WebSocket.URL, nil)
		if err != nil {
			log.Printf("❌ Échec de la connexion WebSocket : %v. Nouvelle tentative dans 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("✅ Connexion WebSocket réussie.")
		break
	}
	defer ws.Close()

	for {
		metrics := GetLatestMetrics()
		if metrics != nil {
			data, err := json.Marshal(metrics)
			if err != nil {
				log.Printf("Erreur JSON: %v", err)
				continue
			}
			err = ws.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Printf("Erreur WebSocket: %v", err)
				break // Reconnexion si erreur
			}
			fmt.Println("✅ Métriques envoyées via WebSocket :", string(data))
		}
		time.Sleep(5 * time.Second)
	}
}

