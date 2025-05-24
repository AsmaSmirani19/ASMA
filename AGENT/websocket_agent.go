package agent 

import (
	
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func StartWebSocketAgent() (*websocket.Conn, error) {
    var conn *websocket.Conn
    var err error

    for {
        conn, _, err = websocket.DefaultDialer.Dial(AppConfig.WebSocket.URL, nil)
        if err != nil {
            log.Printf("❌ Échec de la connexion WebSocket : %v. Nouvelle tentative dans 5s...", err)
            time.Sleep(5 * time.Second)
            continue
        }
        log.Println("✅ Connexion WebSocket réussie.")
        break
    }

    return conn, nil
}






