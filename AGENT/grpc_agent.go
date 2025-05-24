
package agent

import (
	"context"
	"log"
	"net"

	"mon-projet-go/testpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)


func startGRPCServer() {
	lis, err := net.Listen("tcp", AppConfig.GRPC.Port)
	if err != nil {
		log.Fatalf("Échec d'écoute : %v", err)
	}

	grpcServer := grpc.NewServer()
	testpb.RegisterTestServiceServer(grpcServer, &twampAgent{})

	log.Println("Agent TWAMP (serveur gRPC) démarré sur le port 50052")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Échec du serveur gRPC : %v", err)
	}
}
func startClientStream() {
	conn, err := grpc.Dial(
		AppConfig.Server.Main,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Échec de connexion au serveur principal : %v", err)
	}
	defer conn.Close()

	client := testpb.NewTestServiceClient(conn)
	stream, err := client.PerformQuickTest(context.Background())
	if err != nil {
		log.Fatalf("Échec de création du stream : %v", err)
	}

	log.Println("Connexion au serveur principal établie")

	// Boucle pour maintenir la connexion ouverte
	for {
		select {
		case <-stream.Context().Done():
			log.Println("Connexion au serveur terminée")
			return
		}
	}
}