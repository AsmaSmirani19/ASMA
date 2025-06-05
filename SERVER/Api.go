package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"


	_ "github.com/lib/pq"
	
	
)

// Structure représentant un test
type Test struct {
	ID         int       `json:"id"`
	Status     string    `json:"status"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	TestResult string    `json:"test_result"`
}

func InitDB() (*sql.DB, error) {
	connStr := "host=localhost port=5432 user=postgres password=admin dbname=QoS_Results sslmode=disable"
	return sql.Open("postgres", connStr)
}

func enableCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

type AgentService struct {
	db *sql.DB
}

func (s *AgentService) CheckAllAgents() {
	// 1. Récupérer tous les agents
	rows, err := s.db.Query(`SELECT id, "Name", "Address" FROM "Agent_List"`)
	if err != nil {
		log.Fatalf("Erreur récupération agents: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var agent Agent // ← utiliser ta struct complète

		// 2. Récupération des données
		if err := rows.Scan(&agent.ID, &agent.Name, &agent.Address); err != nil {
			log.Printf("Erreur scan agent: %v", err)
			continue
		}

		// 3. Vérification santé via gRPC
		healthy, msg := CheckAgentHealthGRPC(agent.Address)
		agent.TestHealth = healthy

		// 4. Mise à jour en base
		_, err := s.db.Exec(`UPDATE "Agent_List" SET "Test_health" = $1 WHERE "id" = $2`, agent.TestHealth, agent.ID)
		if err != nil {
			log.Printf("Erreur mise à jour test_health pour l'agent %d: %v", agent.ID, err)
		}

		// 5. Affichage console
		status := "❌"
		if healthy {
			status = "✅"
		}
		fmt.Printf("%s Agent %d (%s @ %s): %s\n",
			status, agent.ID, agent.Name, agent.Address, msg)
	}
}

var tests = []Test{}
var mu sync.Mutex
func triggerTestHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w, r)
		log.Printf("📥 Requête reçue sur /api/trigger-test - Méthode: %s", r.Method)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		var req struct {
			TestID int `json:"test_id"`
			ID     int `json:"id"` // compatibilité avec ancien champ
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("❌ Erreur décodage JSON: %v", err)
			http.Error(w, "Format de requête invalide", http.StatusBadRequest)
			return
		}

		effectiveID := req.TestID
		if effectiveID == 0 {
			effectiveID = req.ID
		}
		if effectiveID == 0 {
			log.Println("⚠️ Aucun ID de test fourni")
			http.Error(w, "ID de test requis", http.StatusBadRequest)
			return
		}

		// ✅ Charger la config pour obtenir le type de test
		testConfig, err := LoadFullTestConfiguration(db, effectiveID)
		if err != nil {
			log.Printf("❌ Erreur GetTestConfig: %v", err)
			http.Error(w, "Configuration de test introuvable", http.StatusNotFound)
			return
		}

		switch testConfig.TestType {
		case "planned_test":
			
		var targetCount int
		err := db.QueryRow("SELECT COUNT(*) FROM test_targets WHERE test_id = $1", effectiveID).Scan(&targetCount)
		if err != nil {
			log.Printf("❌ Erreur lors du comptage des targets : %v", err)
			http.Error(w, "Erreur interne", http.StatusInternalServerError)
			return
		}

		if targetCount <= 1 {
			log.Println("📤 Déclenchement test planned SIMPLE via TriggerTestToKafka")
			if err := TriggerTestToKafka(db, effectiveID); err != nil {
				log.Printf("❌ Erreur envoi Kafka : %v", err)
				http.Error(w, "Erreur lors de l’envoi du test planned", http.StatusInternalServerError)
				return
			}
		} else {
			log.Println("📤 Déclenchement test planned GROUPE via TriggerAgentToGroupTest")
			brokers := AppConfig.Kafka.Brokers
			topic := AppConfig.Kafka.TestRequestTopic

			if err := TriggerAgentToGroupTest(db, brokers, topic, effectiveID); err != nil {
				log.Printf("❌ Erreur envoi Kafka : %v", err)
				http.Error(w, "Erreur lors de l’envoi du test groupé", http.StatusInternalServerError)
				return
			}
		}

		case "quick_test":
			log.Println("⚡ Démarrage du Quick Test via gRPC Client...")
			address := fmt.Sprintf("%s:%d", testConfig.SourceIP, testConfig.SourcePort)
			testIDStr := strconv.Itoa(testConfig.TestID)
			protoConfig := convertToProtoConfig(testConfig)

			if err := sendTestConfigToAgent(address, protoConfig, testIDStr); err != nil {
				log.Printf("❌ Erreur envoi config à l'agent %s : %v", address, err)
				http.Error(w, "Erreur d’envoi du test quick", http.StatusInternalServerError)
				return
			}

		default:
			log.Printf("❌ Type de test inconnu : %s", testConfig.TestType)
			http.Error(w, "Type de test inconnu", http.StatusBadRequest)
			return
		}

		// ✅ Réponse HTTP finale
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Test déclenché avec succès",
			"test_id": effectiveID,
			"type":    testConfig.TestType,
		})
		log.Printf("✅ Test %d (%s) déclenché avec succès", effectiveID, testConfig.TestType)
	}
}



// Route GET pour récupérer les résultats des tests
func getTestResults(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tests)
}

func handleAgents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w, r)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			// 🔍 Récupérer tous les agents
			agents, err := getAgentsFromDB(db)
			if err != nil {
				http.Error(w, "Erreur lors de la récupération des agents", http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(agents)

		case http.MethodPost:
			// ➕ Ajouter un agent
			var agent Agent
			if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
				http.Error(w, "Erreur lors de la lecture des données", http.StatusBadRequest)
				return
			}

			// Log des données reçues pour débogage
			log.Printf("Données reçues pour l'agent: %+v\n", agent)

			// Validation des données
			if agent.Name == "" || agent.Address == "" {
				http.Error(w, "Données manquantes", http.StatusBadRequest)
				return
			}

			// Si les données sont valides, on les insère dans la base
			if err := saveAgentToDB(db, &agent); err != nil {
				http.Error(w, "Erreur lors de la sauvegarde", http.StatusInternalServerError)
				return
			}

			// Retourner l'agent créé avec son ID
			json.NewEncoder(w).Encode(agent)

		case http.MethodPut:
			// ✏️ Mettre à jour un agent
			var agent Agent
			if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
				http.Error(w, "Erreur lors de la lecture des données", http.StatusBadRequest)
				return
			}
			if err := updateAgentInDB(db, agent); err != nil {
				http.Error(w, "Erreur lors de la mise à jour", http.StatusInternalServerError)
				return
			}
			w.Write([]byte(`{"message":"Agent mis à jour avec succès"}`))

		case http.MethodDelete:
			// Extraire l'ID de l'URL (dans le chemin)
			idStr := strings.TrimPrefix(r.URL.Path, "/api/agents/")
			id, err := strconv.Atoi(idStr)
			if err != nil || id <= 0 {
				http.Error(w, "ID invalide pour la suppression", http.StatusBadRequest)
				return
			}

			log.Printf("Suppression de l'agent avec ID : %d\n", id)

			// Suppression de l'agent dans la base de données
			if err := deleteAgentFromDB(db, id); err != nil {
				log.Printf("Erreur lors de la suppression de l'agent avec ID %d: %v\n", id, err)
				http.Error(w, fmt.Sprintf("Erreur lors de la suppression de l'agent avec ID %d", id), http.StatusInternalServerError)
				return
			}

			// Répondre avec un message de succès
			w.WriteHeader(http.StatusOK) // Code de statut 200 OK
			w.Write([]byte(`{"message":"Agent supprimé avec succès"}`))

		default:
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		}
	}
}

func handleAgentLink(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			groupIDStr := r.URL.Query().Get("group_id")
			if groupIDStr == "" {
				http.Error(w, "Paramètre group_id manquant", http.StatusBadRequest)
				return
			}
			groupID, err := strconv.Atoi(groupIDStr)
			if err != nil {
				http.Error(w, "ID de groupe invalide", http.StatusBadRequest)
				return
			}
			agents, err := getAgentsByGroupID(db, groupID)
			if err != nil {
				log.Println("Erreur getAgentsByGroupID:", err)
				http.Error(w, "Erreur lors de la récupération des agents du groupe", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(agents); err != nil {
				log.Println("Erreur encodage JSON:", err)
			}
		case http.MethodPost:
			var payload struct {
				GroupID  int   `json:"group_id"`
				AgentIDs []int `json:"agent_ids"`
			}

			log.Println("🔁 Requête POST reçue pour liaison agents")

			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "JSON invalide", http.StatusBadRequest)
				return
			}

			if len(payload.AgentIDs) == 0 {
				http.Error(w, "Liste d'agents vide", http.StatusBadRequest)
				return
			}

			log.Printf("🔗 Liaison agents %v au groupe %d", payload.AgentIDs, payload.GroupID)
			err := linkAgentsToGroup(db, payload.GroupID, payload.AgentIDs)
			if err != nil {
				log.Println("❌ Erreur lors de l'association des agents:", err)
				http.Error(w, "Erreur lors de l'association des agents", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"message": "Agents liés avec succès"}`))
		default:
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		}
	}
}

func handleAgentGroup(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Vérifier la méthode HTTP
		switch r.Method {
		// Méthode POST : Créer un groupe d'agents
		case http.MethodPost:
			log.Println("🔍 Début du traitement de la méthode POST pour créer un groupe d'agents")

			var agentGroup agentGroup
			if err := json.NewDecoder(r.Body).Decode(&agentGroup); err != nil {
				log.Printf("❌ Erreur de décodage des données du groupe : %v\n", err)
				http.Error(w, "Erreur de décodage des données du groupe", http.StatusBadRequest)
				return
			}

			// Log après la décodification des données
			log.Printf("📦 Groupe reçu : %+v\n", agentGroup)

			// Sauvegarder le groupe d'agents dans la base de données
			if err := saveAgentGroupToDB(db, &agentGroup); err != nil {
				log.Printf("❌ Erreur lors de l'enregistrement du groupe : %v\n", err)
				http.Error(w, "Erreur lors de l'enregistrement du groupe", http.StatusInternalServerError)
				return
			}

			// Log si l'enregistrement s'est bien passé
			log.Printf("✅ Groupe enregistré avec succès : %+v\n", agentGroup)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(agentGroup)

		// Méthode GET : Récupérer tous les groupes d'agents
		case http.MethodGet:
			log.Println("🔍 Début du traitement de la méthode GET pour récupérer les groupes d'agents")

			agentGroups, err := getAgentGroupsFromDB(db)
			if err != nil {
				log.Printf("❌ Erreur lors de la récupération des groupes : %v\n", err)
				http.Error(w, "Erreur lors de la récupération des groupes", http.StatusInternalServerError)
				return
			}

			log.Printf("📦 Groupes récupérés depuis DB : %+v\n", agentGroups)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(agentGroups)

		// Méthode PUT : Mettre à jour un groupe d'agents
		case http.MethodPut:
			log.Println("🔍 Début du traitement de la méthode PUT pour mettre à jour un groupe d'agents")

			var agentGroup agentGroup
			if err := json.NewDecoder(r.Body).Decode(&agentGroup); err != nil {
				log.Printf("❌ Erreur de décodage des données du groupe : %v\n", err)
				http.Error(w, "Erreur de décodage des données du groupe", http.StatusBadRequest)
				return
			}

			// Log après la décodification des données
			log.Printf("📦 Groupe à mettre à jour : %+v\n", agentGroup)

			if err := updateAgentGroupInDB(db, agentGroup); err != nil {
				log.Printf("❌ Erreur lors de la mise à jour du groupe : %v\n", err)
				http.Error(w, "Erreur lors de la mise à jour du groupe", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(agentGroup)

			// Méthode DELETE : Supprimer un groupe d'agents
		case http.MethodDelete:
			log.Println("🔍 Début du traitement de la méthode DELETE pour supprimer un groupe d'agents")

			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				log.Println("❌ L'ID du groupe est requis pour suppression")
				http.Error(w, "L'ID du groupe est requis", http.StatusBadRequest)
				return
			}

			id, err := strconv.Atoi(idStr)
			if err != nil {
				log.Printf("❌ L'ID du groupe doit être un entier : %v\n", err)
				http.Error(w, "L'ID du groupe doit être un entier", http.StatusBadRequest)
				return
			}

			if err := deleteAgentGroupFromDB(db, id); err != nil {
				log.Printf("❌ Erreur lors de la suppression du groupe : %v\n", err)
				http.Error(w, "Erreur lors de la suppression du groupe", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))

		default:
			log.Printf("❌ Méthode non autorisée : %s\n", r.Method)
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)

		}
	}
}

func handleTests(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w, r)

		// Vérifier la méthode HTTP
		switch r.Method {
		case http.MethodOptions:
			// CORS pré-contrôle
			w.WriteHeader(http.StatusOK)
			return

		// Méthode POST : Créer un test planifié
		case http.MethodPost:
			log.Println("🔍 Début du traitement de la méthode POST pour créer un test planifié")

			var test PlannedTest
			if err := json.NewDecoder(r.Body).Decode(&test); err != nil {
				log.Printf("❌ Erreur de décodage des données du test: %v\n", err)
				http.Error(w, "Erreur de décodage des données du test", http.StatusBadRequest)
				return
			}

			// Log après la décodification des données
			log.Printf("📦 Test reçu : %+v\n", test)

			// Vérifier si TestDuration est vide et attribuer une valeur par défaut si nécessaire
			if test.TestDuration == "" {
				test.TestDuration = "0s" // Par défaut, si vide
			}

			// Vérifier si la durée du test est dans un format valide
			if _, err := time.ParseDuration(test.TestDuration); err != nil {
				log.Printf("❌ Durée de test invalide : %v\n", err)
				http.Error(w, "Durée de test invalide", http.StatusBadRequest)
				return
			}

			// Sauvegarder le test dans la base de données
			if err := saveTestToDB(db, test); err != nil {
				log.Printf("❌ Erreur lors de l'enregistrement du test : %v\n", err)
				http.Error(w, "Erreur lors de l'enregistrement du test", http.StatusInternalServerError)
				return
			}

			// Log si l'enregistrement s'est bien passé
			log.Printf("✅ Test enregistré avec succès : %+v\n", test)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(test)

		// Méthode GET : Récupérer tous les tests planifiés
		case http.MethodGet:
			log.Println("🔍 Début du traitement de la méthode GET pour récupérer les tests planifiés")

			tests, err := getTestsFromDB(db)
			if err != nil {
				log.Printf("❌ Erreur lors de la récupération des tests : %v\n", err)
				http.Error(w, "Erreur lors de la récupération des tests", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(tests)

		// Méthode PUT : Mettre à jour un test planifié
		case http.MethodPut:
			log.Println("🔍 Début du traitement de la méthode PUT pour mettre à jour un test planifié")

			var test PlannedTest
			if err := json.NewDecoder(r.Body).Decode(&test); err != nil {
				log.Printf("❌ Erreur de décodage des données du test: %v\n", err)
				http.Error(w, "Erreur de décodage des données du test", http.StatusBadRequest)
				return
			}

			// Log après la décodification des données
			log.Printf("📦 Test à mettre à jour : %+v\n", test)

			if err := updateTestInDB(db, test); err != nil {
				log.Printf("❌ Erreur lors de la mise à jour du test : %v\n", err)
				http.Error(w, "Erreur lors de la mise à jour du test", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(test)

		// Méthode DELETE : Supprimer un test planifié
		case http.MethodDelete:
			log.Println("🔍 Début du traitement de la méthode DELETE pour supprimer un test planifié")

			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				log.Println("❌ L'ID du test est requis pour suppression")
				http.Error(w, "L'ID du test est requis", http.StatusBadRequest)
				return
			}

			id, err := strconv.Atoi(idStr)
			if err != nil {
				log.Printf("❌ L'ID du test doit être un entier : %v\n", err)
				http.Error(w, "L'ID du test doit être un entier", http.StatusBadRequest)
				return
			}

			if err := deleteTestFromDB(db, id); err != nil {
				log.Printf("❌ Erreur lors de la suppression du test : %v\n", err)
				http.Error(w, "Erreur lors de la suppression du test", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Test supprimé avec succès"))

		// Si la méthode n'est pas autorisée
		default:
			log.Printf("❌ Méthode non autorisée : %s\n", r.Method)
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		}
	}
}

func handleTestProfile(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {

		case http.MethodPost:
			log.Println("🔍 Début du traitement de la méthode POST pour créer un test profile")

			// 🔽 Ajoute ce bloc pour logguer le corps brut
			body, _ := io.ReadAll(r.Body)
			log.Printf("📥 Corps brut reçu : %s\n", string(body))
			r.Body = io.NopCloser(bytes.NewBuffer(body)) // Permet de relire le body après lecture

			var profile testProfile
			if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
				log.Printf("❌ Erreur de décodage des données du test profile : %v\n", err)
				http.Error(w, "Erreur de décodage des données du test profile", http.StatusBadRequest)
				return
			}

			log.Printf("📦 Test profile reçu : %+v\n", profile)

			// ...
			if err := saveTestProfileToDB(db, profile); err != nil {
				log.Printf("❌ Erreur lors de l'enregistrement du test profile : %v\n", err)
				http.Error(w, "Erreur lors de l'enregistrement du test profile", http.StatusInternalServerError)
				return
			}

			log.Printf("✅ Test profile enregistré avec succès : %+v\n", profile)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(profile)

		case http.MethodGet:
			log.Println("🔍 Début du traitement de la méthode GET pour récupérer les test profiles")

			profiles, err := getTestProfilesFromDB(db)
			if err != nil {
				log.Printf("❌ Erreur lors de la récupération des test profiles : %v\n", err)
				http.Error(w, "Erreur lors de la récupération des test profiles", http.StatusInternalServerError)
				return
			}

			log.Printf("📦 Test profiles récupérés depuis DB : %+v\n", profiles)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(profiles)

		case http.MethodPut:
			log.Println("🔍 Début du traitement de la méthode PUT pour mettre à jour un test profile")

			var profile testProfile
			if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
				log.Printf("❌ Erreur de décodage des données du test profile : %v\n", err)
				http.Error(w, "Erreur de décodage des données du test profile", http.StatusBadRequest)
				return
			}

			log.Printf("📦 Test profile à mettre à jour : %+v\n", profile)

			if err := updateTestProfileInDB(db, profile); err != nil {
				log.Printf("❌ Erreur lors de la mise à jour du test profile : %v\n", err)
				http.Error(w, "Erreur lors de la mise à jour du test profile", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(profile)

		case http.MethodDelete:
			log.Println("🔍 Début du traitement de la méthode DELETE pour supprimer un test profile")

			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				log.Println("❌ L'ID du test profile est requis pour suppression")
				http.Error(w, "L'ID du test profile est requis", http.StatusBadRequest)
				return
			}

			id, err := strconv.Atoi(idStr)
			if err != nil {
				log.Printf("❌ L'ID du test profile doit être un entier : %v\n", err)
				http.Error(w, "L'ID du test profile doit être un entier", http.StatusBadRequest)
				return
			}

			if err := deleteTestProfileFromDB(db, id); err != nil {
				log.Printf("❌ Erreur lors de la suppression du test profile : %v\n", err)
				http.Error(w, "Erreur lors de la suppression du test profile", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))

		default:
			log.Printf("❌ Méthode non autorisée : %s\n", r.Method)
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		}
	}
}

func handleThreshold(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {
		case http.MethodPost:
			log.Println("🔍 Début du traitement de la méthode POST pour créer un threshold")

			var threshold Threshold
			if err := json.NewDecoder(r.Body).Decode(&threshold); err != nil {
				log.Printf("❌ Erreur de décodage des données du threshold : %v\n", err)
				http.Error(w, "Erreur de décodage des données du threshold", http.StatusBadRequest)
				return
			}

			log.Printf("📦 Threshold reçu : %+v\n", threshold)

			// Enregistrer le threshold dans la base de données
			if err := saveThresholdToDB(db, threshold); err != nil {
				log.Printf("❌ Erreur lors de l'enregistrement du threshold : %v\n", err)
				http.Error(w, "Erreur lors de l'enregistrement du threshold", http.StatusInternalServerError)
				return
			}

			log.Printf("✅ Threshold enregistré avec succès : %+v\n", threshold)

			// Générer les listes activeThresholds et disabledThresholds
			setThresholdStatusLists(&threshold)

			// Répondre avec le threshold, incluant les listes générées
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(threshold)

		case http.MethodGet:
			log.Println("🔍 Début du traitement de la méthode GET pour récupérer les thresholds")

			thresholds, err := getThresholdsFromDB(db)
			if err != nil {
				log.Printf("❌ Erreur lors de la récupération des thresholds : %v\n", err)
				http.Error(w, "Erreur lors de la récupération des thresholds", http.StatusInternalServerError)
				return
			}

			log.Printf("📦 Thresholds récupérés depuis DB : %+v\n", thresholds)

			// Générer les listes activeThresholds et disabledThresholds pour chaque threshold récupéré
			for i := range thresholds {
				setThresholdStatusLists(&thresholds[i])
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(thresholds)

		case http.MethodPut:
			log.Println("🔍 Début du traitement de la méthode PUT pour mettre à jour un threshold")

			var threshold Threshold
			if err := json.NewDecoder(r.Body).Decode(&threshold); err != nil {
				log.Printf("❌ Erreur de décodage des données du threshold : %v\n", err)
				http.Error(w, "Erreur de décodage des données du threshold", http.StatusBadRequest)
				return
			}

			log.Printf("📦 Threshold à mettre à jour : %+v\n", threshold)

			// Mettre à jour le threshold dans la base de données
			if err := updateThresholdInDB(db, threshold); err != nil {
				log.Printf("❌ Erreur lors de la mise à jour du threshold : %v\n", err)
				http.Error(w, "Erreur lors de la mise à jour du threshold", http.StatusInternalServerError)
				return
			}

			// Générer les listes activeThresholds et disabledThresholds pour la mise à jour
			setThresholdStatusLists(&threshold)

			// Répondre avec le threshold mis à jour
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(threshold)

		case http.MethodDelete:
			log.Println("🔍 Début du traitement de la méthode DELETE pour supprimer un threshold")

			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				log.Println("❌ L'ID du threshold est requis pour suppression")
				http.Error(w, "L'ID du threshold est requis", http.StatusBadRequest)
				return
			}

			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				log.Printf("❌ L'ID du threshold doit être un entier : %v\n", err)
				http.Error(w, "L'ID du threshold doit être un entier", http.StatusBadRequest)
				return
			}

			if err := deleteThresholdFromDB(db, id); err != nil {
				log.Printf("❌ Erreur lors de la suppression du threshold : %v\n", err)
				http.Error(w, "Erreur lors de la suppression du threshold", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))

		default:
			log.Printf("❌ Méthode non autorisée : %s\n", r.Method)
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		}
	}
}

func handleGetAllTests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	db, err := InitDB()
	if err != nil {
		log.Printf("Erreur connexion DB : %v", err)
		http.Error(w, "Erreur de connexion à la base de données", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	tests, err := LoadAllTestsSummary(db)
	if err != nil {
		log.Printf("Erreur LoadAllTestsSummary : %v", err)
		http.Error(w, "Erreur lors de la récupération des tests", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tests); err != nil {
		log.Printf("Erreur JSON encode : %v", err)
		http.Error(w, "Erreur encodage JSON", http.StatusInternalServerError)
	}
}


func handleGetTestByID(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
        return
    }

	idStr := strings.TrimPrefix(r.URL.Path, "/api/test-results/")
	log.Printf("➡️ Requête pour l'ID : %s", idStr)

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		log.Printf("❌ ID invalide ou nul : '%s' (%d)", idStr, id)
		http.Error(w, fmt.Sprintf("ID de test invalide : %d", id), http.StatusBadRequest)
		return
	}

    db, err := InitDB()
    if err != nil {
        log.Println("❌ Connexion DB échouée")
        http.Error(w, "Erreur de connexion à la base de données", http.StatusInternalServerError)
        return
    }
    defer db.Close()

    testDetails, err := GetTestDetailsByID(db, id)
    if err != nil {
        log.Printf("❌ Test non trouvé ou erreur SQL : %v", err)
        http.Error(w, "Test non trouvé", http.StatusNotFound)
        return
    }

    log.Printf("✅ Test trouvé : %+v", testDetails)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(testDetails)
}

func handlePlannedTest(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodGet:
            idStr := r.URL.Query().Get("id")
            log.Printf("Requête GET /api/planned-test avec id=%s", idStr)
            if idStr == "" {
                log.Println("Erreur: ID du test requis mais absent")
                http.Error(w, "L'ID du test est requis", http.StatusBadRequest)
                return
            }

            id, err := strconv.Atoi(idStr)
            if err != nil {
                log.Printf("Erreur: ID non valide '%s': %v", idStr, err)
                http.Error(w, "L'ID doit être un nombre valide", http.StatusBadRequest)
                return
            }

            plannedTest, err := GetPlannedTestInfoByID(db, id)
            if err != nil {
                log.Printf("Erreur récupération test ID=%d: %v", id, err)
                http.Error(w, "Test non trouvé", http.StatusNotFound)
                return
            }

            log.Printf("Test trouvé : %+v", plannedTest)

            w.Header().Set("Content-Type", "application/json")
            if err := json.NewEncoder(w).Encode(plannedTest); err != nil {
                log.Printf("Erreur encodage JSON : %v", err)
            }

        default:
            log.Printf("Méthode non autorisée: %s", r.Method)
            http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
        }
    }
}



