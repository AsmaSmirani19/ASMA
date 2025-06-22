package server

import (
	"database/sql"
	"log"
	"net/http"
	"encoding/json"
)

type TestStats struct {
	Total        int             `json:"total"`
	Success      int             `json:"success"`
	Failed       int             `json:"failed"`
	TestsByType  []TestTypeStat  `json:"tests_by_type"` // ⬅️ nouveau champ
}

type TestTypeStat struct {
	TestType string `json:"test_type"`
	Total    int    `json:"total"`
}

type AgentStatus struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	TestHealth bool   `json:"test_health"`
}


func GetTestStatsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}

		var stats TestStats

		// 🟢 Requête 1 : total, success, failed
		query := `
			SELECT
				COUNT(*) AS total_tests,
				COUNT(*) FILTER (WHERE completed = TRUE) AS tests_success,
				COUNT(*) FILTER (WHERE failed = TRUE OR "Error" = TRUE) AS tests_failed
			FROM test;
		`
		err := db.QueryRow(query).Scan(&stats.Total, &stats.Success, &stats.Failed)
		if err != nil {
			log.Printf("❌ Erreur stats globales : %v", err)
			http.Error(w, "Erreur statistiques globales", http.StatusInternalServerError)
			return
		}

		// 🟠 Requête 2 : par type
		typeQuery := `
			SELECT test_type, COUNT(*) AS total
			FROM test
			WHERE test_type IN ('planned_test', 'quick_test')
			GROUP BY test_type;
		`
		rows, err := db.Query(typeQuery)
		if err != nil {
			log.Printf("❌ Erreur stats par type : %v", err)
			http.Error(w, "Erreur statistiques par type", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var tts TestTypeStat
			if err := rows.Scan(&tts.TestType, &tts.Total); err == nil {
				stats.TestsByType = append(stats.TestsByType, tts)
			}
		}

		log.Printf("📊 Statistiques complètes : %+v", stats)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

func GetAgentsStatusHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}

		query := `
			SELECT id, "Name" , "Test_health"
			FROM "Agent_List"
			ORDER BY id ASC;
		`

		rows, err := db.Query(query)
		if err != nil {
			log.Printf("❌ Erreur lors de la récupération des agents : %v", err)
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var agents []AgentStatus

		for rows.Next() {
			var agent AgentStatus
			err := rows.Scan(&agent.ID, &agent.Name, &agent.TestHealth)
			if err != nil {
				log.Printf("❌ Erreur de lecture d’un agent : %v", err)
				continue
			}
			agents = append(agents, agent)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(agents)
	}
}


