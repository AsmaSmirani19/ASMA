package server

import (
	"log"
	"net/http"
	"fmt"

	"github.com/gorilla/websocket"
)

type HealthUpdate struct {
    IP     string `json:"ip"`
    Status string `json:"status"` // OK or FAIL
} 

// Crée un upgrader WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler pour gérer la connexion WebSocket
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("❌ Erreur de connexion WebSocket:", err)
		return
	}
	defer conn.Close()

	log.Println("✅ Connexion établie avec l'agent via WebSocket")

	for {
		// Lecture du message envoyé par l'agent via WebSocket
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("🔌 Erreur de lecture ou connexion fermée :", err)
			break
		}

		// Afficher ou traiter le message reçu (résultats TWAMP)
		log.Printf("📨 Résultat reçu de l'agent : %s\n", string(msg))

		// Optionnellement, répondre à l'agent (si nécessaire)
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Résultat reçu avec succès")); err != nil {
			log.Println("🔴 Erreur lors de l'envoi de message de confirmation à l'agent :", err)
			break
		}
	}
}

func StartWebSocketServer() {
    // Enregistre le handler WebSocket
    http.HandleFunc("/ws", handleWebSocket)

    // Adresse configurée
    addr := fmt.Sprintf("%s:%d", AppConfig.WebSocket.Address, AppConfig.WebSocket.Port)

    log.Printf("🚀 Serveur WebSocket lancé sur %s...", addr)
    if err := http.ListenAndServe(addr, nil); err != nil {
        log.Fatalf("Erreur WebSocket: %v", err)
    }
}



