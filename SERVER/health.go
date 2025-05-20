package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	pb "mon-projet-go/testpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func CheckAgentHealthGRPC(address string) (bool, string) {
	// Nettoyage de l'adresse
	address = strings.TrimSpace(address)
	if address == "" {
		msg := "Adresse gRPC vide"
		log.Printf("ERREUR: %s", msg)
		return false, msg
	}

	// Si aucun port n'est fourni, on ajoute le port gRPC par défaut
	if !strings.Contains(address, ":") {
		address += ":50051"
	}

	// Vérification du format
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		msg := fmt.Sprintf("Format d'adresse invalide : '%s'", address)
		log.Printf("ERREUR: %s (%v)", msg, err)
		return false, msg
	}

	// Vérification que le port est un nombre valide
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		msg := fmt.Sprintf("Port invalide dans l'adresse : '%s'", address)
		log.Printf("ERREUR: %s", msg)
		return false, msg
	}


    log.Printf("🔄 Tentative de connexion gRPC à %s...", address)
	// Connexion gRPC avec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		msg := fmt.Sprintf("Échec de connexion à l'agent (%s)", address)
		log.Printf("ERREUR: %s (%v)", msg, err)
		return false, fmt.Sprintf("%s (%v)", msg, err)
	}
	defer conn.Close()

	// Appel HealthCheck
	client := pb.NewHealthClient(conn)
	resp, err := client.HealthCheck(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		msg := fmt.Sprintf("Échec du HealthCheck pour %s", address)
		log.Printf("ERREUR: %s (%v)", msg, err)
		return false, msg
	}

	log.Printf("✅ HealthCheck réussi pour %s → Statut: %s", address, resp.Status)
	return resp.Status == "OK", ""
}
