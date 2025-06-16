package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
	"database/sql"

	pb "mon-projet-go/testpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AgentService struct {
	db *sql.DB
}

func (s *AgentService) CheckAllAgents() {
	rows, err := s.db.Query(`SELECT id, "Name", "Address" FROM "Agent_List"`)
	if err != nil {
		log.Printf("❌ Erreur récupération des agents : %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var agent Agent
		if err := rows.Scan(&agent.ID, &agent.Name, &agent.Address); err != nil {
			log.Printf("❌ Erreur lecture agent : %v", err)
			continue
		}

		isHealthy, _ := CheckAgentHealthGRPC(agent.Address)
		agent.TestHealth = isHealthy

		_, err := s.db.Exec(`UPDATE "Agent_List" SET "Test_health" = $1 WHERE "id" = $2`, isHealthy, agent.ID)
		if err != nil {
			log.Printf("❌ Erreur mise à jour Test_health pour %s : %v", agent.Name, err)
			continue
		}

		fmt.Printf("🔎 Agent [%s @ %s] -> En ligne : %v\n", agent.Name, agent.Address, isHealthy)
	}

	if err := rows.Err(); err != nil {
		log.Printf("⚠️ Erreur post-iteration agents : %v", err)
	}
}

func CheckAgentHealthGRPC(address string) (bool, string) {
	address = strings.TrimSpace(address)
	if address == "" {
		return false, "adresse gRPC vide"
	}

	// Ajouter le port par défaut si absent
	if !strings.Contains(address, ":") {
		address += ":50051"
	}

	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		return false, fmt.Sprintf("format d'adresse invalide : '%s'", address)
	}

	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return false, fmt.Sprintf("port invalide dans l'adresse : '%s'", address)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		return false, fmt.Sprintf("échec de connexion à %s : %v", address, err)
	}
	defer conn.Close()

	client := pb.NewHealthClient(conn)
	resp, err := client.HealthCheck(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		return false, fmt.Sprintf("échec du HealthCheck pour %s : %v", address, err)
	}

	if resp.GetStatus() != "OK" {
		return false, fmt.Sprintf("statut non OK reçu de %s : %s", address, resp.GetStatus())
	}

	return true, "Agent OK"
}
