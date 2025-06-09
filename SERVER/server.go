package server

import (
	"context"
	"database/sql"
	
	
	"fmt"
	"log"

	"time"
	"net/http"
	"strings"


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
	http.HandleFunc("/api/test-results", handleGetAllTests)
	http.HandleFunc("/api/test-results/", handleGetTestByID)
	http.HandleFunc("/api/planned-test", handlePlannedTest(db))
	http.HandleFunc("/api/test-results_id", getTestResultsHandler(db))


	//http.HandleFunc("/api/test-results", handleTestResults(db))
	//http.HandleFunc("/ws/health", healthWebSocketHandler)


	// 🌍 4. Middleware CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:4200",
			"http://localhost:54010",
			"http://localhost:56617", // Ajout pour résoudre ton erreur actuelle
		},
		AllowedMethods: []string{"GET", "POST", "DELETE", "PUT", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	// 🚀 5. Lancement du serveur HTTP
	go func() {
		fmt.Println("🌐 Serveur HTTP lancé sur http://localhost:5000")
		handler := c.Handler(http.DefaultServeMux)
		if err := http.ListenAndServe(":5000", handler); err != nil {
			log.Fatalf("❌ Erreur serveur HTTP : %v", err)
		}
	}()

	// 📦 6. Lancement du consommateur Kafka pour les résultats de test
	ctx := context.Background()
	go ConsumeTestResults(ctx,
		[]string{"localhost:9092"}, // brokers Kafka
		"test-results",             // topic
		"test-group",               // group ID
    db,                        // <--- Ajoute la variable db ici
	)


	// 🔁 7. Vérification des agents
	agentService := &AgentService{db: db}
	agentService.CheckAllAgents()

	// 🎧 8. Écoute active des résultats TWAMP

	// 🛑 9. Blocage principal pour garder le serveur actif
	select {}
}
