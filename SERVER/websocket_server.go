package server

import (
	"log"
	"net/http"
	"encoding/json" 

	"fmt"
	"time"
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


	// Lance le serveur WebSocket dans une goroutine pour ne pas bloquer
	go func() {
		log.Println("🚀 Serveur WebSocket lancé sur le port 8080...")
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("Erreur WebSocket: %v", err)
		}
	}()
}

func healthWebSocketHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("Erreur d'upgrade WebSocket:", err)
        return
    }
    defer conn.Close()

    log.Println("Client WebSocket connecté")

    for {
        update := HealthUpdate{
            IP:     "192.168.1.100",
            Status: "OK", // tu peux simuler un changement ici
        }

        msg, _ := json.Marshal(update)
        err := conn.WriteMessage(websocket.TextMessage, msg)
        if err != nil {
            log.Println("Erreur d'envoi:", err)
            break
        }

        time.Sleep(5 * time.Second)
    }
}
