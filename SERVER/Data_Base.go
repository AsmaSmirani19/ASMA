package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

// Fonction de connexion à la base de données
func connectToDB() (*sql.DB, error) {
	connStr := "host=localhost port=5432 user=postgres password=admin dbname=QoS_Results sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Erreur de connexion :", err)
		return nil, err
	}

	// ✅ Ajoute cette vérification ici
	err = db.Ping()
	if err != nil {
		log.Fatal("❌ Impossible de se connecter à la base de données :", err)
		return nil, err
	} else {
		log.Println("✅ Connexion à la base de données réussie")
	}

	return db, nil
}

func saveResultsToDB(db *sql.DB, qos QoSMetrics) error {
	_, err := db.Exec(`
		INSERT INTO qos_results (
			packet_loss_percent,
			avg_latency_ms,
			avg_jitter_ms,
			avg_throughput_kbps,
			total_jitter
		) VALUES ($1, $2, $3, $4, $5)`,
		qos.PacketLossPercent,
		qos.AvgLatencyMs,
		qos.AvgJitterMs,
		qos.AvgThroughputKbps,
		qos.TotalJitter,
	)
	if err != nil {
		log.Println("Erreur insertion dans DB :", err)
	} else {
		log.Println("Résultat inséré avec succès !")
	}
	return err
}

// Insertion dans les tables
func saveAgentToDB(db *sql.DB, agent Agent) error {
	err := db.QueryRow(`
		INSERT INTO "Agent_List"("Name", "Address", "Test_health", "Availability")
		VALUES($1, $2, $3, $4)
		RETURNING id
	`, agent.Name,
		agent.Address,
		agent.TestHealth,
		agent.Availability).Scan(&agent.ID) // 👈 récupère l'ID généré
	if err != nil {
		log.Printf("Erreur lors de l'insertion dans la base de données : %v\n", err)
		return err
	}
	log.Println("Agent inséré avec succès avec l'ID :", agent.ID)
	return nil
}

// Lecture des agents
func getAgentsFromDB(db *sql.DB) ([]Agent, error) {
	rows, err := db.Query(`SELECT id, "Name", "Address", "Test_health", "Availability" FROM "Agent_List"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		err := rows.Scan(&a.ID, &a.Name, &a.Address, &a.TestHealth, &a.Availability) // 👈 Ajouté a.ID ici
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// Mise à jour d’un agent

func updateAgentInDB(db *sql.DB, agent Agent) error {
	_, err := db.Exec(`
		UPDATE "Agent_List"
		SET "Name" = $1, "Address" = $2, "Test_health" = $3, "Availability" = $4
		WHERE id = $5`,
		agent.Name, agent.Address, agent.TestHealth, agent.Availability, agent.ID,
	)
	if err != nil {
		log.Printf("Erreur lors de la mise à jour : %v\n", err)
		return err
	}
	log.Println("Agent mis à jour avec succès.")
	return nil
}

// Suppression d’un agent
func deleteAgentFromDB(db *sql.DB, agentID int) error {
	_, err := db.Exec(`DELETE FROM "Agent_List" WHERE id = $1`, agentID)
	if err != nil {
		log.Printf("Erreur lors de la suppression : %v\n", err)
		return err
	}
	log.Println("Agent supprimé avec succès.")
	return nil
}

func saveTestToDB(db *sql.DB, test plannedtest) error {
	_, err := db.Exec(`
		INSERT INTO "planned_test"(
			"test_name", 
			"test_duration",
			"number_of_agents",
			"creation_date"
		) VALUES ($1, $2, $3, $4)
	`,
		test.TestName,       // $1
		test.TestDuration,   // $2
		test.NumberOfAgents, // $3
		test.CreationDate,   // $4
	)

	if err != nil {
		log.Println("❌ Erreur lors de l'insertion du test :", err)
		return err
	}
	log.Println("✅ Test enregistré avec succès.")
	return nil
}

func getPlannedTestsFromDB(db *sql.DB) ([]plannedtest, error) {
	rows, err := db.Query(`SELECT "Id", "test_name", "test_duration", "number_of_agents", "creation_date" FROM "planned_test"`)
	if err != nil {
		log.Printf("Erreur lors de la récupération des tests planifiés : %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var tests []plannedtest
	for rows.Next() {
		var t plannedtest
		err := rows.Scan(&t.ID, &t.TestName, &t.TestDuration, &t.NumberOfAgents, &t.CreationDate)
		if err != nil {
			log.Printf("Erreur lors de la lecture des données du test : %v\n", err)
			return nil, err
		}
		tests = append(tests, t)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Erreur lors du parcours des résultats : %v\n", err)
		return nil, err
	}

	if len(tests) == 0 {
		log.Println("Aucun test trouvé")
	}

	return tests, nil
}

func updatePlannedTestInDB(db *sql.DB, test plannedtest) error {
	_, err := db.Exec(`
		UPDATE "planned_test"
		SET "test_name" = $1, "test_duration" = $2, "number_of_agents" = $3, "creation_date" = $4
		WHERE "Id" = $5`,
		test.TestName, test.TestDuration, test.NumberOfAgents, test.CreationDate, test.ID,
	)
	if err != nil {
		log.Printf("Erreur lors de la mise à jour du test planifié avec ID %d : %v\n", test.ID, err)
		return err
	}
	log.Printf("Test planifié avec ID %d mis à jour avec succès.\n", test.ID)
	return nil
}

func deletePlannedTestFromDB(db *sql.DB, testID int) error {
	_, err := db.Exec(`DELETE FROM "planned_test" WHERE "Id" = $1`, testID)
	if err != nil {
		log.Printf("Erreur lors de la suppression du test planifié avec ID %d : %v\n", testID, err)
		return err
	}
	log.Printf("Test planifié avec ID %d supprimé avec succès.\n", testID)
	return nil
}

func saveAgentGroupToDB(db *sql.DB, group agentGroup) error {
	_, err := db.Exec(`
		INSERT INTO "agent-group"(
			"group_name", 
			"number_of_agents",
			"creation_date"
		) VALUES ($1, $2, $3)
	`,
		group.GroupName,      // $1
		group.NumberOfAgents,   // $2
		group.CreationDate,    // $3
	)

	if err != nil {
		log.Println("❌ Erreur lors de l'insertion du groupe d'agents :", err)
		return err
	}
	log.Println("✅ Groupe d'agents enregistré avec succès.")
	return nil
}

func getAgentGroupsFromDB(db *sql.DB) ([]agentGroup, error) {
	rows, err := db.Query(`SELECT "ID", "group_name", "number_of_agents", "creation_date" FROM "agent-group"`)
	if err != nil {
		log.Printf("Erreur lors de la récupération des groupes d'agents : %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var groups []agentGroup
	for rows.Next() {
		var g agentGroup
		err := rows.Scan(&g.ID, &g.GroupName, &g.NumberOfAgents, &g.CreationDate)
		if err != nil {
			log.Printf("Erreur lors de la lecture des données du groupe d'agents : %v\n", err)
			return nil, err
		}
		groups = append(groups, g)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Erreur lors du parcours des résultats : %v\n", err)
		return nil, err
	}

	if len(groups) == 0 {
		log.Println("Aucun groupe trouvé")
	}

	return groups, nil
}

func updateAgentGroupInDB(db *sql.DB, group agentGroup) error {
	_, err := db.Exec(`
		UPDATE "agent-group"
		SET "group_name" = $1, "number_of_agents" = $2, "creation_date" = $3
		WHERE "ID" = $4`,
		group.GroupName, group.NumberOfAgents, group.CreationDate, group.ID,
	)
	if err != nil {
		log.Printf("Erreur lors de la mise à jour du groupe d'agents avec ID %d : %v\n", group.ID, err)
		return err
	}
	log.Printf("Groupe d'agents avec ID %d mis à jour avec succès.\n", group.ID)
	return nil
}

func deleteAgentGroupFromDB(db *sql.DB, groupID int) error {
	_, err := db.Exec(`DELETE FROM "agent-group" WHERE "ID" = $1`, groupID)
	if err != nil {
		log.Printf("Erreur lors de la suppression du groupe d'agents avec ID %d : %v\n", groupID, err)
		return err
	}
	log.Printf("Groupe d'agents avec ID %d supprimé avec succès.\n", groupID)
	return nil
}


