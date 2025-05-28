package server

import (
	"context"
	"database/sql"
	
	"encoding/json"
	"fmt"
	"log"
	"mon-projet-go/testpb"

	"time"
	"net/http"
	"strings"


	"github.com/segmentio/kafka-go"
	"github.com/rs/cors"	
)


func ParsePGInterval(interval string) (time.Duration, error) {
    interval = strings.TrimSpace(interval)
    dur, err := time.ParseDuration(interval)
    if err == nil {
        return dur, nil
    }

    parts := strings.Split(interval, ":")
    switch len(parts) {
    case 3:
        // ex: "HH:MM:SS(.fraction)"
        h := parts[0]
        m := parts[1]
        s := parts[2]
        return time.ParseDuration(fmt.Sprintf("%sh%sm%ss", h, m, s))
    case 2:
        // ex: "MM:SS(.fraction)"
        m := parts[0]
        s := parts[1]
        return time.ParseDuration(fmt.Sprintf("%sm%ss", m, s))
    case 1:
        // ex: "SS(.fraction)" ou nombre seul en secondes
        s := parts[0]
        return time.ParseDuration(s + "s")
    default:
        return 0, fmt.Errorf("format interval invalide: %q", interval)
    }
}

type TestResult struct {
	AgentID    string
	Timestamp  time.Time
	Latency    float64
	Loss       float64
	Throughput float64
}

func listenToTestResultsAndStore(db *sql.DB) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-results",
		GroupID: "backend-group",
	})
	defer reader.Close()

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Erreur Kafka : %v", err)
			continue
		}

		var result TestResult
		if err := json.Unmarshal(msg.Value, &result); err != nil {
			log.Printf("Erreur JSON : %v", err)
			continue
		}

		if err := saveResultsToDB(db, QoSMetrics{}); err != nil {
			log.Printf("Erreur DB : %v", err)
		} else {
			log.Printf("Résultat stocké avec succès : %+v", result)
		}
	}
}

// Implémentation du service Health côté serveur
type healthServer struct {
	testpb.UnimplementedHealthServer
}

// Méthode HealthCheck appelée par l'agent
func (s *healthServer) HealthCheck(ctx context.Context, req *testpb.HealthCheckRequest) (*testpb.HealthCheckResponse, error) {
	log.Println("Reçu une requête HealthCheck de l'agent")
	return &testpb.HealthCheckResponse{Status: "OK"}, nil
}



func Start(db *sql.DB) {

	// 🔧 1. Chargement de la configuration
	LoadConfig("server/config_server.yaml")

	// 📡 2. Lancement du serveur WebSocket en arrière-plan
	go StartWebSocketServer(db)

	// 🌐 4. Définition des routes HTTP
	http.HandleFunc("/api/test/results", getTestResults)
	http.HandleFunc("/api/agents", handleAgents(db))
	http.HandleFunc("/api/agent-group", handleAgentGroup(db))
	http.HandleFunc("/api/agent_link", handleAgentLink(db))
	http.HandleFunc("/api/test-profile", handleTestProfile(db))
	http.HandleFunc("/api/threshold", handleThreshold(db))
	http.HandleFunc("/api/tests", handleTests(db))
	http.HandleFunc("/api/trigger-test", triggerTestHandler(db))
	//http.HandleFunc("/api/test-results", handleTestResults(db))

	//http.HandleFunc("/ws/health", healthWebSocketHandler)

	http.HandleFunc("/api/test-results", handleGetAllTests)
	http.HandleFunc("/api/test-results/", handleGetTestByID)

	http.HandleFunc("/api/planned-test", handlePlannedTest(db))


	// 🌍 5. Middleware CORS
	c := cors.New(cors.Options{
    AllowedOrigins: []string{"http://localhost:4200", "http://localhost:54010"},
    AllowedMethods: []string{"GET", "POST", "DELETE", "PUT", "OPTIONS"},
    AllowedHeaders: []string{"Content-Type", "Authorization"},
})

	// Exemple d'utilisation avec http.DefaultServeMux
	handler := c.Handler(http.DefaultServeMux)

	http.ListenAndServe(":5000", handler)


	// 🚀 6. Lancement du serveur HTTP
	go func() {
		fmt.Println("🌐 Serveur HTTP lancé sur http://localhost:5000")
		log.Fatal(http.ListenAndServe(":5000", handler))
	}()

	// 🚀 7. Lancement du serveur gRPC
	go startGRPCServer()

	// 8. Création du service
	agentService := &AgentService{db: db}
	agentService.CheckAllAgents()

	// 🎧 9. Écoute des résultats de tests TWAMP
	go listenToTestResultsAndStore(db)
	 
	// 🧪 10. Lancement du serveur et client TWAMP
	//go Serveur()
	time.Sleep(1 * time.Second) // délai pour laisser le serveur démarrer


	// 🛑 11. Blocage principal pour garder le serveur actif
	select {}
}