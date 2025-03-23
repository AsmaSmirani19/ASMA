package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"mon-projet-go/testpb"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
)

// Implémentation du service gRPC
type server struct {
	testpb.UnimplementedTestServiceServer
}

// Méthode gRPC pour envoyer un test QoS aux agents
func (s *server) RunQoSTest(ctx context.Context, req *testpb.QoSTestRequest) (*testpb.QoSTestResponse, error) {
	log.Printf("Envoi du test QoS : %s avec config: %s", req.TestId, req.TestParameters)
	return &testpb.QoSTestResponse{
		Status: "Réussi",       // Statut du test
		Result: "Latence 10ms", // Résultats du test
	}, nil

}

var upgrader = websocket.Upgrader{}

// Serveur WebSocket pour recevoir les résultats
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Erreur WebSocket:", err)
		return
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Erreur de lecture WebSocket:", err)
			break
		}
		fmt.Println("Résultat reçu de l'agent :", string(msg))
	}
}

func main() {
	// Démarrage du serveur gRPC
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Échec de l'écoute : %v", err)
	}

	grpcServer := grpc.NewServer()
	testpb.RegisterTestServiceServer(grpcServer, &server{})

	// Lancer le serveur WebSocket en parallèle
	go func() {
		http.HandleFunc("/ws", handleWebSocket)
		log.Println("Serveur WebSocket sur le port 8080...")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	log.Println("Serveur gRPC démarré sur le port 50051...")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Échec du démarrage du serveur : %v", err)
	}
}
