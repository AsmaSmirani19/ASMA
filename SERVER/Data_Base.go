package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
	"strconv"
	"github.com/lib/pq"
)

// / update etat **********
func UpdateTestStatus(db *sql.DB, testID int, inProgress, failed, completed, errorFlag bool) error {
	result, err := db.Exec(`
		UPDATE test
		SET "In_progress" = $1, "failed" = $2, "completed" = $3, "Error" = $4
		WHERE "Id" = $5
	`, inProgress, failed, completed, errorFlag, testID)
	if err != nil {
		return fmt.Errorf("❌ Erreur mise à jour statut test %d : %v", testID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("⚠️ Impossible de récupérer les lignes affectées : %v", err)
	}
	if rowsAffected == 0 {
		log.Printf("⚠️ Aucun test avec ID %d trouvé pour mise à jour", testID)
	} else {
		log.Printf("✅ Statut mis à jour pour test %d → In_progress=%v, failed=%v, completed=%v, Error=%v", testID, inProgress, failed, completed, errorFlag)
	}
	return nil
}

func SaveAttemptResult(db *sql.DB, testID int64, targetID int64, latency, jitter, throughput float64) error {
	query := `
        INSERT INTO attempt_results (test_id, target_id, latency_ms, jitter_ms, throughput_kbps)
        VALUES ($1, $2, $3, $4, $5)
    `
	_, err := db.Exec(query, testID, targetID, latency, jitter, throughput)
	return err
}

func saveResultsToDB(db *sql.DB, qos QoSMetrics) error {
	_, err := db.Exec(`
		INSERT INTO "Test_Results" (
			test_id,
			target_id,
			"PacketLossPercent",
			"AvgLatencyMs",
			"AvgJitterMs",
			"AvgThroughputKbps"
		) VALUES ($1, $2, $3, $4, $5, $6)`,
		qos.TestID,
		qos.TargetID,
		qos.PacketLossPercent,
		qos.AvgLatencyMs,
		qos.AvgJitterMs,
		qos.AvgThroughputKbps,
	)
	if err != nil {
		log.Println("❌ Erreur insertion dans DB :", err)
	} else {
		log.Println("✅ Résultat inséré avec succès !")
	}
	return err
}

func nullToZero(n sql.NullFloat64) float64 {
	if n.Valid {
		return n.Float64
	}
	return 0.0
}

func getResultsFromDB(db *sql.DB, testID int) ([]QoSMetrics, error) {
	rows, err := db.Query(`
		SELECT 
			test_id,
			target_id,
			"PacketLossPercent",
			"AvgLatencyMs",
			"AvgJitterMs",
			"AvgThroughputKbps"
		FROM "Test_Results"
		WHERE test_id = $1
	`, testID)

	if err != nil {
		log.Println("❌ Erreur lors de la lecture des résultats depuis la DB :", err)
		return nil, err
	}
	defer rows.Close()

	var results []QoSMetrics

	for rows.Next() {
		var qos QoSMetrics
		err := rows.Scan(
			&qos.TestID,
			&qos.TargetID,
			&qos.PacketLossPercent,
			&qos.AvgLatencyMs,
			&qos.AvgJitterMs,
			&qos.AvgThroughputKbps,
		)
		if err != nil {
			log.Println("❌ Erreur pendant le scan des résultats :", err)
			return nil, err
		}
		results = append(results, qos)
	}

	if err = rows.Err(); err != nil {
		log.Println("❌ Erreur de lecture finale :", err)
		return nil, err
	}

	log.Printf("✅ %d résultats QoS récupérés avec succès pour test_id = %d", len(results), testID)
	return results, nil
}


func GetAttemptResultsByTestAndTargetID(db *sql.DB, testID int64, targetID int64) ([]AttemptResult, error) {
	query := `
		SELECT target_id, latency_ms, jitter_ms, throughput_kbps
		FROM attempt_results
		WHERE test_id = $1 AND target_id = $2
		ORDER BY id ASC
	`
	rows, err := db.Query(query, testID, targetID)
	if err != nil {
		log.Println("[DB QUERY ERROR]", err)
		return nil, err
	}
	defer rows.Close()

	var results []AttemptResult
	for rows.Next() {
		var targetIDFromDB sql.NullInt64
		var latency sql.NullFloat64
		var jitter sql.NullFloat64
		var throughput sql.NullFloat64

		if err := rows.Scan(&targetIDFromDB, &latency, &jitter, &throughput); err != nil {
			log.Println("[ROW SCAN ERROR]", err)
			return nil, err
		}

		r := AttemptResult{
			TargetID:       targetIDFromDB.Int64,
			LatencyMs:      nullToZero(latency),
			JitterMs:       nullToZero(jitter),
			ThroughputKbps: nullToZero(throughput),
		}

		results = append(results, r)
	}

	log.Printf("[DB SUCCESS] Found %d attempts for test_id = %d and target_id = %d", len(results), testID, targetID)
	return results, nil
}

func GetAllTargetIdsFromAttemptResults(db *sql.DB, testID int64) ([]int64, error) {
	query := `SELECT DISTINCT target_id FROM attempt_results WHERE test_id = $1`
	rows, err := db.Query(query, testID)
	if err != nil {
		log.Println("[DB QUERY ERROR]", err)
		return nil, err
	}
	defer rows.Close()

	var targetIds []int64
	for rows.Next() {
		var tid int64
		if err := rows.Scan(&tid); err != nil {
			log.Println("[ROW SCAN ERROR]", err)
			return nil, err
		}
		targetIds = append(targetIds, tid)
	}
	log.Printf("[DB SUCCESS] Found %d distinct target_id(s) for test_id = %d", len(targetIds), testID)
	return targetIds, nil
}



// Agent_List
func saveAgentToDB(db *sql.DB, agent *Agent) error {
	err := db.QueryRow(`
		INSERT INTO "Agent_List"("Name", "Address", "Port", "Test_health")
		VALUES($1, $2, $3, $4)
		RETURNING id
	`,
		agent.Name,
		agent.Address,
		agent.Port, // <-- ajouté ici
		agent.TestHealth,
	).Scan(&agent.ID) // 👈 modifie bien le champ de l'objet original

	if err != nil {
		log.Printf("Erreur lors de l'insertion dans la base de données : %v\n", err)
		return err
	}
	log.Println("Agent inséré avec succès avec l'ID :", agent.ID)
	return nil
}

// Récupère tous les agents depuis Agent_List
func getAgentsFromDB(db *sql.DB) ([]Agent, error) {
	rows, err := db.Query(`SELECT id, "Name", "Address",  "Port", "Test_health" FROM "Agent_List"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		err := rows.Scan(&a.ID, &a.Name, &a.Address, &a.Port, &a.TestHealth)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// Met à jour un agent entier dans Agent_List (nom, adresse, etc.)
func updateAgentInDB(db *sql.DB, agent Agent) error {
	_, err := db.Exec(`
		UPDATE "Agent_List"
		SET "Name" = $1, "Address" = $2, "Port" = $3, "Test_health" = $4
		WHERE id = $5`,
		agent.Name, agent.Address, agent.Port, agent.TestHealth, agent.ID,
	)
	if err != nil {
		log.Printf("Erreur lors de la mise à jour : %v\n", err)
		return err
	}
	log.Println("Agent mis à jour avec succès.")
	return nil
}

// Supprime un agent de Agent_List
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

// agent_group
func saveAgentGroupToDB(db *sql.DB, group *agentGroup) error {
	if group.GroupName == "" {
		return fmt.Errorf("❌ le nom du groupe ne peut pas être vide")
	}
	tx, err := db.Begin()
	if err != nil {
		log.Printf("❌ Erreur début transaction : %v", err)
		return err
	}
	defer tx.Rollback()
	creationDate := time.Now()
	if !group.CreationDate.IsZero() {
		creationDate = group.CreationDate
	}
	numberOfAgents := len(group.AgentIDs)
	err = tx.QueryRow(`
        INSERT INTO "agent-group" ("group_name", "creation_date", "number_of_agents")
        VALUES ($1, $2, $3)
        RETURNING "ID"
    `, group.GroupName, creationDate, numberOfAgents).Scan(&group.ID)
	if err != nil {
		log.Printf("❌ Erreur création groupe : %v", err)
		return err
	}
	if numberOfAgents > 0 {
		_, err = tx.Exec(`
            INSERT INTO agent_link (group_id, agent_id)
            SELECT $1, unnest($2::int[])
            ON CONFLICT (group_id, agent_id) DO NOTHING
        `, group.ID, pq.Array(group.AgentIDs))
		if err != nil {
			log.Printf("❌ Erreur lien agents au groupe : %v", err)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("❌ Erreur commit : %v", err)
		return err
	}
	group.NumberOfAgents = numberOfAgents
	group.CreationDate = creationDate
	log.Printf("✅ Groupe '%s' enregistré avec ID %d", group.GroupName, group.ID)
	return nil
}

func getAgentGroupsFromDB(db *sql.DB) ([]agentGroup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			g."ID", 
			g."group_name", 
			g."creation_date",
			COUNT(al.agent_id) AS number_of_agents,
			COALESCE(
				ARRAY_AGG(al.agent_id) FILTER (WHERE al.agent_id IS NOT NULL),
				'{}'::int[]
			) as agent_ids
		FROM "agent-group" g
		LEFT JOIN agent_link al ON g."ID" = al.group_id
		GROUP BY g."ID", g."group_name", g."creation_date"
		ORDER BY g."creation_date" DESC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("❌ Erreur requête groupes : %v", err)
		return nil, err
	}
	defer rows.Close()

	var groups []agentGroup
	for rows.Next() {
		var g agentGroup

		if err := rows.Scan(&g.ID, &g.GroupName, &g.CreationDate, &g.NumberOfAgents, &g.AgentIDs); err != nil {
			log.Printf("❌ Erreur lecture ligne groupe : %v", err)
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func updateAgentGroupInDB(db *sql.DB, group agentGroup) error {
	_, err := db.Exec(`
		UPDATE "agent-group"
		up_name" = $1, "number_of_agents" = $2, "creation_date" = $3

	`, group.GroupName, group.CreationDate, group.ID)

	if err != nil {
		log.Printf("❌ Erreur mise à jour groupe ID %d : %v", group.ID, err)
		return err
	}
	log.Printf("✅ Groupe ID %d mis à jour avec succès", group.ID)
	return nil
}
func deleteAgentGroupFromDB(db *sql.DB, groupID int) error {
	_, err := db.Exec(`DELETE FROM "agent-group" WHERE "ID" = $1`, groupID)
	if err != nil {
		log.Printf("❌ Erreur suppression groupe ID %d : %v", groupID, err)
		return err
	}
	log.Printf("✅ Groupe ID %d supprimé avec succès", groupID)
	return nil
}

// agent/group_link ************

func linkAgentsToGroup(db *sql.DB, groupID int, agentIDs []int) error {
	if len(agentIDs) == 0 {
		return fmt.Errorf("aucun agent à lier")
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO agent_link (group_id, agent_id)
		VALUES ($1, $2)
		ON CONFLICT (group_id, agent_id) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, agentID := range agentIDs {
		if _, err := stmt.Exec(groupID, agentID); err != nil {
			return err
		}
	}

	// Mise à jour du nombre d'agents liés dans la table agent-group
	_, err = tx.Exec(`
		UPDATE "agent-group"
		SET number_of_agents = (
			SELECT COUNT(*)
			FROM agent_link
			WHERE group_id = $1
		)
		WHERE "ID" = $1
	`, groupID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("✅ %d agents liés au groupe %d", len(agentIDs), groupID)
	return nil
}

func getAgentsByGroupID(db *sql.DB, groupID int) ([]Agent, error) {
	query := `
		SELECT a.id, a."Name", a."Address", a."Test_health"
		FROM "Agent_List" a
		JOIN agent_link al ON a.id = al.agent_id
		WHERE al.group_id = $1
	`
	rows, err := db.Query(query, groupID)
	if err != nil {
		log.Printf("❌ Erreur récupération agents du groupe %d : %v", groupID, err)
		return nil, err
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.Name, &a.Address, &a.TestHealth); err != nil {
			log.Printf("❌ Erreur lecture agent : %v", err)
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// test **********
func saveTestToDB(db *sql.DB, test PlannedTest) error {
	log.Printf("Début insertion du test : %+v\n", test) // <-- 1. Juste après réception du test

	firstTargetID := 0
	if len(test.TargetAgentIDs) > 0 {
		firstTargetID = test.TargetAgentIDs[0]
	}
	log.Printf("Premier target_id utilisé dans la table test: %d\n", firstTargetID) // <-- 2. Premier target_id

	var testID int
	err := db.QueryRow(`
		INSERT INTO "test" (
			"test_name", "test_duration", "number_of_agents", "creation_date",
			"test_type", "source_id", "target_id", "profile_id", "threshold_id",
			"In_progress", "failed", "completed", "Error"
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING "Id"
	`,
		test.TestName,
		test.TestDuration,
		test.NumberOfAgents,
		test.CreationDate,
		test.TestType,
		test.SourceID,
		test.TargetID,
		test.ProfileID,
		test.ThresholdID,
		test.InProgress,
		test.Failed,
		test.Completed,
		test.Error,
	).Scan(&testID)

	if err != nil {
		log.Printf("Erreur lors de l'insertion du test : %v\n", err) // <-- 3. Erreur insertion test
		return err
	}
	log.Printf("Test inséré avec ID : %d\n", testID) // <-- 4. ID inséré

	log.Printf("Liste des target_agent_ids à insérer dans test_targets : %+v\n", test.TargetAgentIDs) // <-- 5. Liste cible

	stmt, err := db.Prepare("INSERT INTO test_targets (test_id, target_id) VALUES ($1, $2)")
	if err != nil {
		log.Printf("Erreur préparation requête insertion test_targets : %v\n", err) // <-- 6. Préparation requête
		return err
	}
	defer stmt.Close()

	for _, targetID := range test.TargetAgentIDs {
		_, err := stmt.Exec(testID, targetID)
		if err != nil {
			log.Printf("Erreur insertion test_targets (testID=%d, targetID=%d) : %v\n", testID, targetID, err) // <-- 7. Erreur insertion cible
			return err
		}
		log.Printf("Inséré target_id %d pour test_id %d\n", targetID, testID) // <-- 8. Confirmation insertion cible
	}

	log.Println("Insertion test et targets terminée avec succès.") // <-- 9. Fin insertion OK

	return nil
}

func getTestsFromDB(db *sql.DB) ([]PlannedTest, error) {
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
			"In_progress",
			"failed",
			"completed",
			"Error"
		FROM "test"
	`)
	if err != nil {
		log.Printf("Erreur lors de la récupération des tests planifiés : %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var tests []PlannedTest
	for rows.Next() {
		var t PlannedTest
		err := rows.Scan(
			&t.ID,
			&t.TestName,
			&t.TestDuration,
			&t.NumberOfAgents,
			&t.CreationDate,
			&t.TestType,
			&t.SourceID,
			&t.TargetID, // on récupère directement dans TargetID
			&t.ProfileID,
			&t.ThresholdID,
			&t.InProgress,
			&t.Failed,
			&t.Completed,
			&t.Error,
		)

		if err != nil {
			log.Printf("Erreur lors de la lecture des données du test : %v\n", err)
			return nil, err
		}

		// Récupérer les cibles associées à ce test dans test_targets
		targetRows, err := db.Query(`SELECT target_id FROM test_targets WHERE test_id = $1`, t.ID)
		if err != nil {
			log.Printf("Erreur récupération cibles pour test %d : %v\n", t.ID, err)
			return nil, err
		}

		var targetIDs []int
		for targetRows.Next() {
			var targetID int
			if err := targetRows.Scan(&targetID); err != nil {
				log.Printf("Erreur lecture target_id : %v\n", err)
				targetRows.Close()
				return nil, err
			}
			targetIDs = append(targetIDs, targetID)
		}
		targetRows.Close()

		// Si aucune cible dans test_targets, fallback vers TargetID dans la struct
		if len(targetIDs) == 0 && t.TargetID != 0 {
			targetIDs = []int{t.TargetID}
		}
		t.TargetAgentIDs = targetIDs
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

func updateTestInDB(db *sql.DB, test PlannedTest) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	firstTargetID := 0
	if len(test.TargetAgentIDs) > 0 {
		firstTargetID = test.TargetAgentIDs[0]
	}
	_, err = tx.Exec(`
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
            "threshold_id" = $8,
            "In_progress" = $9,
            "failed" = $10,
            "completed" = $11,
            "Error" = $12
        WHERE "Id" = $13
    `,
		test.TestName, test.TestDuration, test.NumberOfAgents, test.CreationDate,
		test.TestType, test.SourceID, test.TargetID, test.ProfileID,
		test.ThresholdID, firstTargetID, // <-- ici
		test.InProgress, test.Failed, test.Completed, test.Error, test.ID,
	)
	if err != nil {
		tx.Rollback()
		return err
	}
	// Supprimer anciennes cibles
	_, err = tx.Exec(`DELETE FROM test_targets WHERE test_id = $1`, test.ID)
	if err != nil {
		tx.Rollback()
		return err
	}
	// Insérer nouvelles cibles
	stmt, err := tx.Prepare(`INSERT INTO test_targets (test_id, target_id) VALUES ($1, $2)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, targetID := range test.TargetAgentIDs {
		_, err := stmt.Exec(test.ID, targetID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
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

// test profile ***********
func saveTestProfileToDB(db *sql.DB, profile testProfile) error {
	// Validation des champs
	if profile.ProfileName == "" || profile.PacketSize <= 0 || profile.TimeBetweenAttempts <= 0 || profile.CreationDate.IsZero() {
		log.Println("❌ Profil invalide : champs obligatoires manquants ou invalides")
		return errors.New("profil invalide : champs obligatoires manquants ou invalides")
	}

	_, err := db.Exec(`
		INSERT INTO "test_profile"(
			"profile_name", 
			"creation_date", 
			"packet_size",
			"time_between_attempts"
		) VALUES ($1, $2, $3 ,  $4)
	`,
		profile.ProfileName,
		profile.CreationDate,
		profile.PacketSize,
		profile.TimeBetweenAttempts,
	)
	if err != nil {
		log.Println("❌ Erreur lors de l'insertion du test profile :", err)
		return err
	}
	log.Println("✅ Test profile enregistré avec succès.")
	return nil
}

func getTestProfilesFromDB(db *sql.DB) ([]testProfile, error) {
	rows, err := db.Query(`
		SELECT "ID", "profile_name", "creation_date", "packet_size", "time_between_attempts"
		FROM "test_profile"
	`)
	if err != nil {
		log.Printf("Erreur lors de la récupération des test profiles : %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var profiles []testProfile
	for rows.Next() {
		var p testProfile
		err := rows.Scan(&p.ID, &p.ProfileName, &p.CreationDate, &p.PacketSize, &p.TimeBetweenAttempts)
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
		SET "profile_name" = $1,
			"creation_date" = $2,
			"packet_size" = $3,
			"time_between_attempts" = $4
		WHERE "ID" = $5
	`,
		profile.ProfileName,
		profile.CreationDate,
		profile.PacketSize,
		profile.TimeBetweenAttempts,
		profile.ID,
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

// threshold ***********
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
			&t.AvgOpr, &t.MinOpr, &t.MaxOpr, &t.SelectedMetric)

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
		threshold.Name,           // $1
		threshold.CreationDate,   // $2
		threshold.Avg,            // $3
		threshold.Min,            // $4
		threshold.Max,            // $5
		threshold.AvgStatus,      // $6
		threshold.MinStatus,      // $7
		threshold.MaxStatus,      // $8
		threshold.AvgOpr,         // $9
		threshold.MinOpr,         // $10
		threshold.MaxOpr,         // $11
		threshold.SelectedMetric, // $12
		threshold.ID,             // $13
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
		thresholdID, // $1
	)
	if err != nil {
		log.Printf("Erreur lors de la suppression du threshold avec ID %d : %v\n", thresholdID, err)
		return err
	}
	log.Printf("Threshold avec ID %d supprimé avec succès.\n", thresholdID)
	return nil
}

func LoadAllTestsSummary(db *sql.DB) ([]DisplayedTest, error) {
	query := `
	SELECT 
		t."Id",
		t.test_name,
		t.test_type,
		t.creation_date,
		t.test_duration::text,
		sa."Name" AS source_agent,
		STRING_AGG(al."Name", ', ') AS target_agent,
		STRING_AGG(tt.target_id::text, ',') AS target_ids,  -- Ajout pour récupérer les target_id
		th."Name" AS threshold_name,
		CASE
			WHEN th."avg_status" THEN th."avg"
			WHEN th."min_status" THEN th."min"
			WHEN th."max_status" THEN th."max"
			ELSE NULL
		END AS threshold_value,
		t."In_progress",
		t."completed",
		t."failed",
		t."Error"
	FROM test t
	LEFT JOIN "Agent_List" sa ON t.source_id = sa."id"
	LEFT JOIN test_targets tt ON tt.test_id = t."Id"
	LEFT JOIN "Agent_List" al ON tt.target_id = al."id"
	LEFT JOIN "Threshold" th ON th."ID" = t.threshold_id
	GROUP BY t."Id", t.test_name, t.test_type, t.creation_date, t.test_duration, sa."Name", th."Name", th."avg_status", th."avg", th."min_status", th."min", th."max_status", th."max", t."In_progress", t."completed", t."failed", t."Error"
	`

	log.Println("Exécution de la requête SQL pour charger les tests...")
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("❌ Erreur requête SQL : %v", err)
		return nil, fmt.Errorf("échec requête : %v", err)
	}
	defer rows.Close()

	var results []DisplayedTest

	for rows.Next() {
		var test DisplayedTest
		var targetAgent sql.NullString
		var targetIDsRaw sql.NullString  // Pour récupérer les target_id en brut

		log.Println("🧪 Tentative de Scan dans LoadAllTestsSummary")
		err := rows.Scan(
			&test.TestID,
			&test.TestName,
			&test.TestType,
			&test.CreationDate,
			&test.TestDuration,
			&test.SourceAgent,
			&targetAgent,
			&targetIDsRaw,  // Ajouté ici
			&test.ThresholdName,
			&test.ThresholdValue,
			&test.InProgress,
			&test.Completed,
			&test.Failed,
			&test.Error,
		)

		if err != nil {
			log.Printf("❌ Erreur lecture ligne (Scanload) : %v", err)
			return nil, fmt.Errorf("échec lecture ligne : %v", err)
		}

		if targetAgent.Valid {
			test.TargetAgent = targetAgent.String
		} else {
			// Requête secondaire si targetAgent est NULL
			targetRows, err := db.Query(`
				SELECT al."Name"
				FROM test_targets tt
				LEFT JOIN "Agent_List" al ON al."id" = tt.target_id
				WHERE tt.test_id = $1
			`, test.TestID)

			if err != nil {
				log.Printf("❌ Erreur récupération targets secondaires : %v", err)
				return nil, fmt.Errorf("échec récupération targets secondaires : %v", err)
			}

			var names []string
			for targetRows.Next() {
				var name sql.NullString
				if err := targetRows.Scan(&name); err == nil && name.Valid {
					names = append(names, name.String)
				}
			}
			targetRows.Close()

			if len(names) > 0 {
				test.TargetAgent = strings.Join(names, ", ")
			} else {
				test.TargetAgent = "<aucun agent>"
			}
		}

		// Parse target_ids en slice d'int
		if targetIDsRaw.Valid {
			idStrings := strings.Split(targetIDsRaw.String, ",")
			for _, idStr := range idStrings {
				idStr = strings.TrimSpace(idStr)
				if idStr == "" {
					continue
				}
				id, err := strconv.Atoi(idStr)
				if err == nil {
					test.TargetIDs = append(test.TargetIDs, id)
				} else {
					log.Printf("⚠️ Ignoré target_id non valide: %q", idStr)
				}
			}
		}

		// Définir le statut
		switch {
		case test.InProgress:
			test.Status = "In_progress"
		case test.Completed:
			test.Status = "completed"
		case test.Failed:
			test.Status = "failed"
		case test.Error:
			test.Status = "Error"
		default:
			test.Status = "unknown"
		}

		log.Printf("✅ Test chargé : %+v", test)
		results = append(results, test)
	}

	log.Printf("✅ %d tests chargés avec succès.", len(results))
	return results, nil
}


func GetTestDetailsByID(db *sql.DB, id int) (*TestDetails, error) {
	log.Printf("📥 Appel GetTestDetailsByID avec testID = %d", id)

	 query := `
SELECT
    t."Id",
    t.test_name,
    CASE
        WHEN t."In_progress" THEN 'in_progress'
        WHEN t."completed" THEN 'completed'
        WHEN t."failed" THEN 'failed'
        WHEN t."Error" THEN 'error'
        ELSE 'unknown'
    END AS status,
    t.creation_date,
    t.test_duration,
    sa."Name" AS source_agent,

    -- 🧠 Agent cible principal : soit depuis test.target_id, soit depuis test_targets
    COALESCE(
        (SELECT a."Name"
         FROM "Agent_List" a
         WHERE a.id = t.target_id
         LIMIT 1),
         
        (SELECT string_agg(a2."Name", ', ')
         FROM test_targets tt
         JOIN "Agent_List" a2 ON tt.target_id = a2.id
         WHERE tt.test_id = t."Id")
    ) AS target_agent,

    th."Name" AS threshold_name,
    CASE
        WHEN th."avg_status" THEN th."avg"
        WHEN th."min_status" THEN th."min"
        WHEN th."max_status" THEN th."max"
        ELSE NULL
    END AS threshold_value,
    CASE
        WHEN th."avg_status" THEN 'avg'
        WHEN th."min_status" THEN 'min'
        WHEN th."max_status" THEN 'max'
        ELSE NULL
    END AS threshold_type,
    CASE
        WHEN th."avg_status" THEN th."avg_opr"
        WHEN th."min_status" THEN th."min_opr"
        WHEN th."max_status" THEN th."max_opr"
        ELSE NULL
    END AS threshold_operator,
    th.selected_metric 
FROM test t
LEFT JOIN "Agent_List" sa ON t.source_id = sa.id
LEFT JOIN "Threshold" th ON th."ID" = t.threshold_id
WHERE t."Id" = $1
`


	var details TestDetails
	row := db.QueryRow(query, id)
	err := row.Scan(
		&details.TestID,
		&details.TestName,
		&details.Status,
		&details.CreationDate,
		&details.TestDuration,
		&details.SourceAgent,
		&details.TargetAgent,
		&details.ThresholdName,
		&details.ThresholdValue,
		&details.ThresholdType,     // ✅ doit venir AVANT
		&details.ThresholdOperator, // ✅ doit venir APRÈS
		&details.SelectedMetric,    // ✅ doit être DERNIER comme dans le SELECT
	)
	log.Printf("📊 Seuil récupéré : value=%v, type=%v, operator=%v, metric=%v", 
	details.ThresholdValue, details.ThresholdType, details.ThresholdOperator, details.SelectedMetric)


	if err == sql.ErrNoRows {
		log.Printf("⚠️ Aucun résultat trouvé pour le test ID=%d", id)
		return nil, fmt.Errorf("aucun test trouvé avec ID=%d", id)
	} else if err != nil {
		log.Printf("❌ Erreur dans GetTestDetailsByID pour test ID=%d : %v", id, err)
		return nil, err
	}

	log.Printf("✅ Test ID=%d trouvé avec nom: %s, status: %s", details.TestID, details.TestName, details.Status)
	return &details, nil
}

// //****************************************/////////////
func LoadFullTestConfiguration(db *sql.DB, testID int) (*FullTestConfiguration, error) {
	log.Printf("➡️ Début du chargement de la configuration complète du test ID=%d", testID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//// ==== Étape 1 : Récupération des données principales du test ====
	query := `
		SELECT 
			t."Id",
			t.test_name,
			t.test_type,
			t.test_duration::text,
			t.number_of_agents,
			t.source_id,
			sa."Address" AS source_ip,
			sa."Port" AS source_port,
			t.target_id,
			ta."Address" AS target_ip,
			ta."Port" AS target_port,
			t.profile_id
		FROM test t
		JOIN "Agent_List" sa ON t.source_id = sa.id
		LEFT JOIN "Agent_List" ta ON t.target_id = ta.id
		WHERE t."Id" = $1
	`

	var config FullTestConfiguration
	var targetID sql.NullInt64
	var targetIP sql.NullString
	var targetPort sql.NullInt64

	err := db.QueryRowContext(ctx, query, testID).Scan(
		&config.TestID,
		&config.Name,
		&config.TestType,
		&config.RawDuration,
		&config.NumberOfAgents,
		&config.SourceID,
		&config.SourceIP,
		&config.SourcePort,
		&targetID,
		&targetIP,
		&targetPort,
		&config.ProfileID,
	)

	if err != nil {
		log.Printf("❌ Erreur récupération test ID=%d : %v", testID, err)
		return nil, fmt.Errorf("erreur récupération test: %w", err)
	}

	//// ==== Étape 2 : Traitement des champs NULL ====
	config.TargetID = 0
	if targetID.Valid {
		config.TargetID = int(targetID.Int64)
	}
	config.TargetIP = strings.TrimSpace(targetIP.String)
	if !targetIP.Valid {
		config.TargetIP = ""
	}
	if targetPort.Valid {
		config.TargetPort = int(targetPort.Int64)
	} else {
		config.TargetPort = 0
	}

	//// ==== Étape 3 : Validation des adresses IP et ports ====
	// Source
	config.SourceIP = strings.TrimSpace(config.SourceIP)
	if config.SourceIP == "" || net.ParseIP(config.SourceIP) == nil {
		return nil, fmt.Errorf("adresse IP source invalide ou vide: %s", config.SourceIP)
	}
	if config.SourcePort <= 0 || config.SourcePort > 65535 {
		return nil, fmt.Errorf("port source invalide: %d", config.SourcePort)
	}

	// Target (si défini)
	if config.TargetID != 0 {
		if config.TargetIP == "" || net.ParseIP(config.TargetIP) == nil {
			return nil, fmt.Errorf("adresse IP cible invalide ou vide: %s", config.TargetIP)
		}
		if config.TargetPort <= 0 || config.TargetPort > 65535 {
			return nil, fmt.Errorf("port cible invalide: %d", config.TargetPort)
		}
	}

	log.Printf("📦 Configuration réseau - Source: %s:%d, Target: %s:%d",
		config.SourceIP, config.SourcePort, config.TargetIP, config.TargetPort)

	//// ==== Étape 4 : Conversion de la durée du test ====
	config.Duration, err = ParsePGInterval(config.RawDuration)
	if err != nil {
		log.Printf("❌ Erreur conversion durée: %v", err)
		return nil, fmt.Errorf("durée invalide: %w", err)
	}
	log.Printf("⏳ Durée du test ID=%d convertie : %v", testID, config.Duration)

	//// ==== Étape 5 : Chargement du profil ====
	profileQuery := `
		SELECT "ID", "profile_name", "packet_size", "time_between_attempts"
		FROM "test_profile"
		WHERE "ID" = $1
	`
	var profileName string
	var rawInterval string
	var profile Profile

	err = db.QueryRowContext(ctx, profileQuery, config.ProfileID).Scan(
		&profile.ID, &profileName, &profile.PacketSize, &rawInterval,
	)
	if err != nil {
		log.Printf("❌ Profil introuvable ID=%d : %v", config.ProfileID, err)
		return nil, fmt.Errorf("profil introuvable ID=%d: %w", config.ProfileID, err)
	}

	profile.SendingInterval, err = ParsePGInterval(rawInterval)
	if err != nil {
		log.Printf("❌ Intervalle invalide pour profil ID=%d : %v", profile.ID, err)
		return nil, fmt.Errorf("durée invalide pour profil ID=%d: %w", profile.ID, err)
	}

	log.Printf("✅ Profil chargé: ID=%d, Name=%s, PacketSize=%d, Interval=%v",
		profile.ID, profileName, profile.PacketSize, profile.SendingInterval)
	config.Profile = &profile

	//// ==== Étape 6 : Récupération des cibles supplémentaires (test_targets) ====
	targetQuery := `SELECT target_id FROM test_targets WHERE test_id = $1`
	rows, err := db.QueryContext(ctx, targetQuery, testID)
	if err != nil {
		log.Printf("❌ Erreur récupération target_ids pour test_id=%d : %v", testID, err)
		return nil, fmt.Errorf("erreur récupération test_targets: %w", err)
	}
	defer rows.Close()

	targetIDSet := make(map[int]struct{})
	if config.TargetID != 0 {
		targetIDSet[config.TargetID] = struct{}{}
	}

	for rows.Next() {
		var tid int
		if err := rows.Scan(&tid); err != nil {
			return nil, err
		}
		targetIDSet[tid] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Ajout dans TargetAgentIDs
	config.TargetAgentIDs = make([]int, 0, len(targetIDSet))
	for tid := range targetIDSet {
		config.TargetAgentIDs = append(config.TargetAgentIDs, tid)
	}

	log.Printf("🎯 TargetAgentIDs pour test %d : %v", testID, config.TargetAgentIDs)

	//// ==== Étape 7 : Complétion IP/port si TargetID manquant ====
	if config.TargetID == 0 && len(config.TargetAgentIDs) > 0 {
		firstTargetID := config.TargetAgentIDs[0]
		agentQuery := `SELECT "Address", "Port" FROM "Agent_List" WHERE "id" = $1`
		err = db.QueryRowContext(ctx, agentQuery, firstTargetID).Scan(&config.TargetIP, &config.TargetPort)
		if err != nil {
			log.Printf("❌ Erreur récupération IP/Port du target_id=%d: %v", firstTargetID, err)
			return nil, fmt.Errorf("erreur récupération IP/Port target_id=%d: %w", firstTargetID, err)
		}
		config.TargetID = firstTargetID
		config.TargetIP = strings.TrimSpace(config.TargetIP)
	}

	//// ==== Étape 8 : Charger les détails des agents cibles ====

	if len(config.TargetAgentIDs) > 0 {
		queryAgents := `SELECT id, "Address", "Port" FROM "Agent_List" WHERE id = ANY($1)`
		rows, err := db.QueryContext(ctx, queryAgents, pq.Array(config.TargetAgentIDs))
		if err != nil {
			log.Printf("❌ Erreur récupération détails agents cibles : %v", err)
			return nil, fmt.Errorf("erreur récupération détails agents cibles: %w", err)
		}
		defer rows.Close()

		var agents []AgentInfo
		for rows.Next() {
			var agent AgentInfo
			if err := rows.Scan(&agent.ID, &agent.IP, &agent.Port); err != nil {
				return nil, err
			}
			agents = append(agents, agent)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}

		config.TargetAgents = agents
	}

	log.Printf("🎉 Configuration complète du test ID=%d chargée avec succès", testID)
	return &config, nil
}

func LoadFullTGroupTest(db *sql.DB, testID int) (*FullTestConfiguration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var config FullTestConfiguration
	var rawDuration string

	// 🔍 Étape 1 : Charger la configuration générale du test
	query := `SELECT t."Id", t.test_name, t.test_type, t.test_duration::text, t.number_of_agents, t.source_id, sa."Address", sa."Port", t.profile_id
              FROM test t
              JOIN "Agent_List" sa ON t.source_id = sa.id
              WHERE t."Id" = $1`

	err := db.QueryRowContext(ctx, query, testID).Scan(
		&config.TestID, &config.Name, &config.TestType, &rawDuration,
		&config.NumberOfAgents, &config.SourceID, &config.SourceIP,
		&config.SourcePort, &config.ProfileID,
	)
	if err != nil {
		return nil, err
	}

	log.Printf("📥 Test ID=%d chargé. Source: %s:%d, ProfileID=%d", config.TestID, config.SourceIP, config.SourcePort, config.ProfileID)

	config.Duration, err = ParsePGInterval(rawDuration)
	if err != nil {
		return nil, err
	}

	// 🔍 Étape 2 : Récupérer les IDs des agents cibles
	rows, err := db.QueryContext(ctx, `SELECT target_id FROM test_targets WHERE test_id = $1`, testID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targetIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		targetIDs = append(targetIDs, id)
	}
	config.TargetAgentIDs = targetIDs

	log.Printf("🎯 TargetAgentIDs récupérés : %v", targetIDs)

	// 🔍 Étape 3 : Récupérer les détails (IP, port) de chaque agent cible
	if len(targetIDs) > 0 {
		log.Printf("🔍 Exécution requête récupération détails agents cibles avec targetIDs via pq.Array")

		targetQuery := `SELECT id, "Address", "Port" FROM "Agent_List" WHERE id = ANY($1)`
		targetRows, err := db.QueryContext(ctx, targetQuery, pq.Array(targetIDs))
		if err != nil {
			log.Printf("❌ Erreur récupération détails agents : %v", err)
			return nil, err
		}
		defer targetRows.Close()

		var targetAgents []AgentInfo
		for targetRows.Next() {
			var a AgentInfo
			if err := targetRows.Scan(&a.ID, &a.IP, &a.Port); err != nil {
				log.Printf("❌ Erreur scan agent : %v", err)
				return nil, err
			}
			log.Printf("📡 Agent cible ID=%d : IP=%s, Port=%d", a.ID, a.IP, a.Port)
			targetAgents = append(targetAgents, a)
		}
		log.Printf("✅ Total agents cibles récupérés : %d", len(targetAgents))

		config.TargetAgents = targetAgents
	} else {
		log.Printf("⚠️ Aucun agent cible trouvé pour test ID=%d", testID)
	}

	// 🔍 Étape 4 : Charger le profil du test
	profileQuery := `SELECT "ID", "packet_size", "time_between_attempts" FROM "test_profile" WHERE "ID" = $1`
	var p Profile
	var rawInterval string
	err = db.QueryRowContext(ctx, profileQuery, config.ProfileID).Scan(&p.ID, &p.PacketSize, &rawInterval)
	if err != nil {
		return nil, err
	}

	p.SendingInterval, err = ParsePGInterval(rawInterval)
	if err != nil {
		return nil, err
	}
	config.Profile = &p

	log.Printf("✅ Profil chargé : ID=%d, PacketSize=%d, Interval=%v", p.ID, p.PacketSize, p.SendingInterval)

	return &config, nil
}
