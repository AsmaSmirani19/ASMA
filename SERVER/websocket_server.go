package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Crée un upgrader WebSocket
var upgrader = websocket.Upgrader{
	// Permet toutes les origines pour l'exemple, mais tu devrais sécuriser ça dans une application réelle.
	CheckOrigin: func(r *http.Request) bool {return true},
}

// Handler pour gérer la connexion WebSocket
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Mise à jour de la connexion HTTP vers une connexion WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("❌ Erreur de connexion WebSocket:", err)
		return
	}
	defer conn.Close()

	log.Println("✅ Connexion établie avec l'agent via WebSocket")

	// Boucle pour recevoir les résultats en temps réel de l'agent
	for {
		// Lecture du message envoyé par l'agent via WebSocket
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("🔌 Erreur de lecture ou connexion fermée :", err)
			break
		}

		// Afficher ou traiter le message reçu (résultats TWAMP)
		log.Printf("📨 Résultat reçu de l'agent : %s\n", string(msg))

		// Tu peux ajouter des étapes ici pour traiter les résultats, les stocker ou les envoyer à d'autres services

		// Optionnellement, répondre à l'agent (si nécessaire)
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Résultat reçu avec succès")); err != nil {
			log.Println("🔴 Erreur lors de l'envoi de message de confirmation à l'agent :", err)
			break
		}
	}
}

// Fonction pour démarrer le serveur WebSocket
func StartWebSocketServer() {
	// Enregistre le handler WebSocket
	http.HandleFunc("/ws", handleWebSocket)

	// Démarre le serveur HTTP qui écoute sur le port 8080 pour les connexions WebSocket
	log.Println("🚀 Serveur WebSocket lancé sur le port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
