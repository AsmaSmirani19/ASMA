package agent

import (
	"encoding/json"
	"log"
	"fmt"

	"github.com/gorilla/websocket"
)
type TestStatus struct {
    TestID int    `json:"test_id"`
    Status string `json:"status"`
}

func sendTestStatus(ws *websocket.Conn, testID int, status string) error {
    if ws == nil {
        log.Println("❌ [sendTestStatus] Connexion WebSocket est nil")
        return fmt.Errorf("websocket is nil")
    }

    msg := WebSocketMessage{
        Type: "status",
        Payload: TestStatus{
            TestID: testID,
            Status: status,
        },
    }
    data, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("marshal json error: %w", err)
    }

    err = ws.WriteMessage(websocket.TextMessage, data)
    if err != nil {
        return fmt.Errorf("websocket write error: %w", err)
    }

    log.Printf("✅ [sendTestStatus] Message WebSocket envoyé : %s\n", string(data))
    return nil
}

