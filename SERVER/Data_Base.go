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
// agent ****
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


func deleteAgentFromDB(db *sql.DB, agentID int) error {
	res, err := db.Exec(`DELETE FROM "Agent_List" WHERE id = $1`, agentID)
	if err != nil {
		log.Printf("Erreur lors de la suppression : %v\n", err)
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Printf("Erreur lors de la récupération du nombre de lignes affectées : %v\n", err)
		return err
	}

	if rowsAffected == 0 {
		log.Printf("Aucun agent trouvé avec l'ID %d.\n", agentID)
		return sql.ErrNoRows // <-- permet de détecter que rien n’a été supprimé
	}

	log.Println("Agent supprimé avec succès.")
	return nil
}


// test **********
func saveTestToDB(db *sql.DB, test plannedtest) error {
	_, err := db.Exec(`
		INSERT INTO "test"(
			"test_name", 
			"test_duration",
			"number_of_agents",
			"creation_date",
			"test_type",
			"source_id",
			"target_id",
			"profile_id",
			"threshold_id",
			"waiting",
			"failed",
			"completed"
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`,
		test.TestName,       // $1
		test.TestDuration,   // $2
		test.NumberOfAgents, // $3
		test.CreationDate,   // $4
		test.TestType,
		test.SourceID,
		test.TargetID,
		test.ProfileID,
		test.ThresholdID,
		test.Waiting,
		test.Failed,
		test.Completed,
	)

	if err != nil {
		log.Println("❌ Erreur lors de l'insertion du test :", err)
		return err
	}
	log.Println("✅ Test enregistré avec succès.")
	return nil
}

func getTestsFromDB(db *sql.DB) ([]plannedtest, error) {
	rows, err := db.Query(`
		SELECT 
            "Id", 
            "test_name", 
            "test_duration", 
            "number_of_agents", 
            "creation_date", 
            "test_type",
            "source_id",
            "target_id",
            "profile_id",
            "threshold_id",
            "waiting",
            "failed",
            "completed"
        FROM "test"
    `)
	if err != nil {
		log.Printf("Erreur lors de la récupération des tests planifiés : %v\n", err)
		return nil, err
	}
	defer rows.Close()
	var tests []plannedtest
	for rows.Next() {
		var t plannedtest
		err := rows.Scan(
			&t.ID,
			&t.TestName,
			&t.TestDuration,
			&t.NumberOfAgents,
			&t.CreationDate,
			&t.TestType,         
			&t.SourceID,    
			&t.TargetID,     
			&t.ProfileID,    
			&t.ThresholdID,  
			&t.Waiting,
			&t.Failed,
			&t.Completed,
		)
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

func updateTestInDB(db *sql.DB, test plannedtest) error {
	_, err := db.Exec(`
		UPDATE "test"
		SET 
			"test_name" = $1, 
			"test_duration" = $2, 
			"number_of_agents" = $3, 
			"creation_date" = $4, 
			"test_type" = $5,                
			"source_id" = $6,           
			"target_id" = $7,           
			"profile_id" = $8,          
			"threshold_id" = $9,       
			"waiting" = $10,           
			"failed" = $11,           
			"completed" = $12           
		WHERE "Id" = $13             
	`, 
		test.TestName, test.TestDuration, test.NumberOfAgents, test.CreationDate,   
		test.TestType,  test.SourceID, test.TargetID, test.ProfileID,      
		test.ThresholdID,test.Waiting, test.Failed,test.Completed, test.ID,             
	)

	if err != nil {
		log.Printf("Erreur lors de la mise à jour du test planifié avec ID %d : %v\n", test.ID, err)
		return err
	}
	log.Printf("Test planifié avec ID %d mis à jour avec succès.\n", test.ID)
	return nil
}


func deleteTestFromDB(db *sql.DB, testID int) error {
	_, err := db.Exec(`DELETE FROM "test" WHERE "Id" = $1`, testID)
	if err != nil {
		log.Printf("Erreur lors de la suppression du test planifié avec ID %d : %v\n", testID, err)
		return err
	}
	log.Printf("Test planifié avec ID %d supprimé avec succès.\n", testID)
	return nil
}



// agent groupe

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


// test profile ***********
func saveTestProfileToDB(db *sql.DB, profile testProfile) error {
	_, err := db.Exec(`
		INSERT INTO "test_profile"(
			"profile_name", 
			"creation_date", 
			"packet_size"
		) VALUES ($1, $2, $3)
	`,
		profile.ProfileName,    // $1
		profile.CreationDate,   // $2
		profile.PacketSize,     // $3
	)

	if err != nil {
		log.Println("❌ Erreur lors de l'insertion du test profile :", err)
		return err
	}
	log.Println("✅ Test profile enregistré avec succès.")
	return nil
}

func getTestProfilesFromDB(db *sql.DB) ([]testProfile, error) {
	rows, err := db.Query(`SELECT "ID", "profile_name", "creation_date", "packet_size" FROM "test_profile"`)
	if err != nil {
		log.Printf("Erreur lors de la récupération des test profiles : %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var profiles []testProfile
	for rows.Next() {
		var p testProfile
		err := rows.Scan(&p.ID, &p.ProfileName, &p.CreationDate, &p.PacketSize)
		if err != nil {
			log.Printf("Erreur lors de la lecture des données du test profile : %v\n", err)
			return nil, err
		}
		profiles = append(profiles, p)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Erreur lors du parcours des résultats : %v\n", err)
		return nil, err
	}

	if len(profiles) == 0 {
		log.Println("Aucun test profile trouvé")
	} else {
		log.Printf("Nombre de profils récupérés: %d\n", len(profiles))
	}

	return profiles, nil
}


func updateTestProfileInDB(db *sql.DB, profile testProfile) error {
	_, err := db.Exec(`
		UPDATE "test_profile"
		SET "profile_name" = $1, "creation_date" = $2, "packet_size" = $3
		WHERE "ID" = $4`,
		profile.ProfileName, profile.CreationDate, profile.PacketSize, profile.ID,
	)
	if err != nil {
		log.Printf("Erreur lors de la mise à jour du test profile avec ID %d : %v\n", profile.ID, err)
		return err
	}
	log.Printf("Test profile avec ID %d mis à jour avec succès.\n", profile.ID)
	return nil
}

func deleteTestProfileFromDB(db *sql.DB, profileID int) error {
	_, err := db.Exec(`DELETE FROM "test_profile" WHERE "ID" = $1`, profileID)
	if err != nil {
		log.Printf("Erreur lors de la suppression du test profile avec ID %d : %v\n", profileID, err)
		return err
	}
	log.Printf("Test profile avec ID %d supprimé avec succès.\n", profileID)
	return nil
}




//threshold ***********
func saveThresholdToDB(db *sql.DB, threshold Threshold) error {
	log.Println("🟡 Insertion du threshold:", threshold)

    _, err := db.Exec(`
   INSERT INTO "Threshold"("Name", "creation_date", "avg", "min", "max", 
        "avg_status", "min_status", "max_status", "avg_opr", "min_opr", "max_opr", "selected_metric")
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,

    threshold.Name, 
	threshold.CreationDate, 
	threshold.Avg,
	threshold.Min,
	threshold.Max, 
    threshold.AvgStatus,
	threshold.MinStatus,
	threshold.MaxStatus, 
    threshold.AvgOpr,
	threshold.MinOpr, 
	threshold.MaxOpr, 
	threshold.SelectedMetric, 
)

    if err != nil {
        log.Println("❌ Erreur lors de l'insertion du Threshold :", err)
        return err
    }
    log.Println("✅ Threshold enregistré avec succès.")
    return nil
}


func setThresholdStatusLists(threshold *Threshold) {
    // Initialiser les listes comme vides
    threshold.ActiveThresholds = []string{}
    threshold.DisabledThresholds = []string{}

    // Ajouter "avg_status" à la liste Active si AvgStatus est vrai, sinon à la liste Disabled
    if threshold.AvgStatus {
        threshold.ActiveThresholds = append(threshold.ActiveThresholds, "avg_status")
    } else {
        threshold.DisabledThresholds = append(threshold.DisabledThresholds, "avg_status")
    }

    // Ajouter "min_status" à la liste Active si MinStatus est vrai, sinon à la liste Disabled
    if threshold.MinStatus {
        threshold.ActiveThresholds = append(threshold.ActiveThresholds, "min_status")
    } else {
        threshold.DisabledThresholds = append(threshold.DisabledThresholds, "min_status")
    }

    // Ajouter "max_status" à la liste Active si MaxStatus est vrai, sinon à la liste Disabled
    if threshold.MaxStatus {
        threshold.ActiveThresholds = append(threshold.ActiveThresholds, "max_status")
    } else {
        threshold.DisabledThresholds = append(threshold.DisabledThresholds, "max_status")
    }
}


func getThresholdsFromDB(db *sql.DB) ([]Threshold, error) {
	rows, err := db.Query(`
   		 SELECT "ID", "Name", "creation_date", "avg", "min", "max", 
           "avg_status", "min_status", "max_status", 
           "avg_opr", "min_opr", "max_opr" , "selected_metric"
   		 FROM "Threshold"
	`)

    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var thresholds []Threshold
    for rows.Next() {
        var t Threshold
		err := rows.Scan(&t.ID, &t.Name, &t.CreationDate,
			&t.Avg, &t.Min, &t.Max,
			&t.AvgStatus, &t.MinStatus, &t.MaxStatus,
			&t.AvgOpr, &t.MinOpr, &t.MaxOpr , &t.SelectedMetric)
		
        if err != nil {
            return nil, err
        }

        // Assure que les listes sont bien initialisées comme vides
        if t.ActiveThresholds == nil {
            t.ActiveThresholds = []string{}
        }
        if t.DisabledThresholds == nil {
            t.DisabledThresholds = []string{}
        }

        // Appel à la fonction setThresholdStatusLists pour remplir les listes
        setThresholdStatusLists(&t)

        thresholds = append(thresholds, t)
    }
    return thresholds, nil
}

func updateThresholdInDB(db *sql.DB, threshold Threshold) error {
    _, err := db.Exec(`
        UPDATE "Threshold"
        SET "Name" = $1, "creation_date" = $2,
            "avg" = $3, "min" = $4, "max" = $5,
            "avg_status" = $6, "min_status" = $7, "max_status" = $8,
            "avg_opr" = $9, "min_opr" = $10, "max_opr" = $11 , "selected_metric" = $12
        WHERE "ID" = $13
    `,
        threshold.Name,              // $1
        threshold.CreationDate,      // $2
        threshold.Avg,               // $3
        threshold.Min,               // $4
        threshold.Max,               // $5
        threshold.AvgStatus,         // $6
        threshold.MinStatus,         // $7
        threshold.MaxStatus,         // $8
        threshold.AvgOpr,            // $9
        threshold.MinOpr,            // $10
        threshold.MaxOpr,            // $11
		threshold.SelectedMetric,    // $12        
        threshold.ID,                // $13           
    )
    if err != nil {
        log.Printf("❌ Erreur lors de la mise à jour du threshold avec ID %d : %v\n", threshold.ID, err)
        return err
    }
    log.Printf("✅ Threshold avec ID %d mis à jour avec succès.\n", threshold.ID)
    return nil
}


func deleteThresholdFromDB(db *sql.DB, thresholdID int64) error {
    _, err := db.Exec(`
        DELETE FROM "Threshold" WHERE "ID" = $1
    `,
        thresholdID,  // $1
    )
    if err != nil {
        log.Printf("Erreur lors de la suppression du threshold avec ID %d : %v\n", thresholdID, err)
        return err
    }
    log.Printf("Threshold avec ID %d supprimé avec succès.\n", thresholdID)
    return nil
}







