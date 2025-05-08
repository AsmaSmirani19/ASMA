package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Structure représentant un test
type Test struct {
	ID         int       `json:"id"`
	Status     string    `json:"status"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	TestResult string    `json:"test_result"`
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

var tests = []Test{}
var mu sync.Mutex

// Route POST pour démarrer un test
func startTest(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	testID := len(tests) + 1
	newTest := Test{
		ID:        testID,
		Status:    "En cours",
		StartTime: time.Now(),
	}

	// Démarre l'envoi du paquet de test avec la fonction client
	go client()

	time.Sleep(5 * time.Second)

	// Mettre à jour le test
	newTest.Status = "Terminé"
	newTest.EndTime = time.Now()
	newTest.TestResult = "Résultat du test ici..." // Remplacer par les résultats réels

	// Ajouter le test à la liste
	tests = append(tests, newTest)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newTest)
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
			if agent.Name == "" || agent.Address == "" || agent.Availability == 0 {
				http.Error(w, "Données manquantes", http.StatusBadRequest)
				return
			}

			// Si les données sont valides, on les insère dans la base
			if err := saveAgentToDB(db, agent); err != nil {
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

			var test plannedtest
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

			var test plannedtest
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
			if err := saveAgentGroupToDB(db, agentGroup); err != nil {
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

func handleTestProfile(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {

		case http.MethodPost:
			log.Println("🔍 Début du traitement de la méthode POST pour créer un test profile")

			var profile testProfile
			if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
				log.Printf("❌ Erreur de décodage des données du test profile : %v\n", err)
				http.Error(w, "Erreur de décodage des données du test profile", http.StatusBadRequest)
				return
			}

			log.Printf("📦 Test profile reçu : %+v\n", profile)

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
