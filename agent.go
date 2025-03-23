package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"mon-projet-go/testpb"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connexion au serveur gRPC
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Impossible de se connecter : %v", err)
	}
	defer conn.Close()

	client := testpb.NewTestServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Demande d'un test QoS au serveur (ajustez en fonction de votre .proto)
	req := &testpb.QoSTestRequest{
		TestId:         "Latence",                      // Assurez-vous d'utiliser la bonne casse
		TestParameters: "10 paquets ICMP vers 8.8.8.8", // Assurez-vous que ces champs existent dans votre .proto
	}

	// Appel RunQoSTest
	res, err := client.RunQoSTest(ctx, req)
	if err != nil {
		log.Fatalf("Erreur lors de la demande du test QoS : %v", err)
	}

	// Utilisation des champs corrects dans la réponse : Status et Result
	fmt.Println("Test QoS reçu du serveur :", res.Status, res.Result)

	// Simuler un test QoS
	time.Sleep(2 * time.Second)
	testResult := fmt.Sprintf("Test ID: %s - Latence mesurée: 50ms", res.Status)

	// Connexion WebSocket au serveur
	ws, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatalf("Erreur de connexion WebSocket : %v", err)
	}
	defer ws.Close()

	// Envoi du résultat via WebSocket
	err = ws.WriteMessage(websocket.TextMessage, []byte(testResult))
	if err != nil {
		log.Fatalf("Erreur d'envoi WebSocket : %v", err)
	}

	fmt.Println("Résultat envoyé au serveur via WebSocket")
	defer func() {
		ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Fin de communication"))
		ws.Close()
	}()
	
}
