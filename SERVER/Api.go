package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
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

// Fonction qui envoie un paquet pour un test
func sendTestPacket() {
	fmt.Println("Envoi du paquet de test...")
	time.Sleep(2 * time.Second) // Simuler l'envoi
	fmt.Println("Paquet envoyé avec succès!")
}

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

	// Démarre l'envoi du paquet de test
	go sendTestPacket()

	// Simuler un délai pour le test
	time.Sleep(5 * time.Second)

	// Mettre à jour le test
	newTest.Status = "Terminé"
	newTest.EndTime = time.Now()
	newTest.TestResult = "Résultat du test ici..."

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
        enableCORS(w,r)

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
            // Supprimer un agent
            idStr := r.URL.Query().Get("id")
            id, err := strconv.Atoi(idStr)
            if err != nil || id == 0 {
                http.Error(w, "ID invalide pour la suppression", http.StatusBadRequest)
                return
            }
            if err := deleteAgentFromDB(db, id); err != nil {
                http.Error(w, "Erreur lors de la suppression", http.StatusInternalServerError)
                return
            }
            w.Write([]byte(`{"message":"Agent supprimé avec succès"}`))

        default:
            http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
        }
    }
}

func handlePlannedTests(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w,r)

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
				test.TestDuration = "0s"  // Par défaut, si vide
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

			tests, err := getPlannedTestsFromDB(db)
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

			if err := updatePlannedTestInDB(db, test); err != nil {
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

			if err := deletePlannedTestFromDB(db, id); err != nil {
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
	
		// ✅ CORRECTION IMPORTANTE ICI
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	
	default:
		log.Printf("❌ Méthode non autorisée : %s\n", r.Method)
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
	
		}
	}
}




