package agent

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

func sendTestStatus(ws *websocket.Conn, testID int, status string) {
	if ws == nil {
		log.Println("❌ [sendTestStatus] Connexion WebSocket est nil")
		return
	}

	// Construction du message WebSocket
	msg := WebSocketMessage{
		Type: "status",
		Payload: TestStatus{
			TestID: testID,
			Status: status,
		},
	}
	log.Printf("🛠️ [sendTestStatus] Structure préparée : %+v\n", msg)

	// Sérialisation JSON
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("❌ [sendTestStatus] Erreur lors du marshal JSON : %v\n", err)
		return
	}
	log.Printf("📤 [sendTestStatus] JSON généré : %s\n", string(data))

	// Envoi via WebSocket
	err = ws.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Printf("❌ [sendTestStatus] Erreur lors de l'envoi WS : %v\n", err)
	} else {
		log.Printf("✅ [sendTestStatus] Message WebSocket envoyé : %s\n", string(data))
	}
}
