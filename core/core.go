package core

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"log"

	_ "github.com/lib/pq"
)

func InitDB() (*sql.DB, error) {
	return sql.Open("postgres", "host=localhost port=5432 user=postgres password=admin dbname=QoS_Results sslmode=disable")
}

func LoadFullTestConfiguration(db *sql.DB, testID int) (*FullTestConfiguration, error) {
	query := `
		SELECT 
			t."Id",
			t.test_name,
			t.test_duration::text,
			t.number_of_agents,
			t.source_id,
			sa."Address" AS source_ip,
			sa."Port" AS source_port,
			t.target_id,
			ta."Address" AS target_ip,
			ta."Port" AS target_port,
			t.profile_id,
			t.threshold_id,
			t.waiting,
			t.failed,
			t.completed
		FROM test t
		JOIN "Agent_List" sa ON t.source_id = sa.id
		JOIN "Agent_List" ta ON t.target_id = ta.id
		WHERE t."Id" = $1
	`

	var config FullTestConfiguration
	err := db.QueryRow(query, testID).Scan(
		&config.TestID,
		&config.Name,
		&config.RawDuration,
		&config.NumberOfAgents,
		&config.SourceID,
		&config.SourceIP,
		&config.SourcePort,
		&config.TargetID,
		&config.TargetIP,
		&config.TargetPort,
		&config.ProfileID,
		&config.ThresholdID,
		&config.Waiting,
		&config.Failed,
		&config.Completed,
	)
	if err != nil {
		return nil, fmt.Errorf("erreur récupération test: %v", err)
	}

	// Conversion de la durée Postgres en time.Duration
	config.Duration, err = ParsePGInterval(config.RawDuration)
	if err != nil {
		return nil, fmt.Errorf("durée invalide: %v", err)
	}

	// Récupération du profil (profil_id)
	profileRows, err := db.Query(`SELECT "ID", "profile_name", "packet_size", "time_between_attempts" FROM "test_profile"`)
	if err != nil {
		return nil, fmt.Errorf("erreur récupération profils: %v", err)
	}
	defer profileRows.Close()

	for profileRows.Next() {
		var p Profile
		var rawInterval string
		// On ignore profile_name ici car non utilisé dans Profile, mais tu peux l'ajouter si besoin
		var profileName string
		err := profileRows.Scan(&p.ID, &profileName, &p.PacketSize, &rawInterval)
		if err != nil {
			continue
		}
		if p.ID == config.ProfileID {
			p.SendingInterval, _ = ParsePGInterval(rawInterval)
			config.Profile = &p
			break
		}
	}
	if config.Profile == nil {
		return nil, fmt.Errorf("profil introuvable pour ID: %d", config.ProfileID)
	}

	// Récupération des seuils
	thresholdRows, err := db.Query(`
		SELECT "ID", "Name", "avg", "min", "max", "avg_status", "min_status", "avg_opr", "min_opr", "max_opr", "selected_metric"
		FROM "Threshold"
	`)
	if err != nil {
		return nil, fmt.Errorf("erreur récupération seuils: %v", err)
	}
	defer thresholdRows.Close()

	for thresholdRows.Next() {
		var t Threshold
		err := thresholdRows.Scan(
			&t.ID,
			&t.Name,
			&t.Avg,
			&t.Min,
			&t.Max,
			&t.AvgStatus,
			&t.MinStatus,
			&t.AvgOpr,
			&t.MinOpr,
			&t.MaxOpr,
			&t.SelectedMetric,
		)
		if err != nil {
			continue
		}
		if t.ID == config.ThresholdID {
			config.Threshold = &t
			break
		}
	}
	if config.Threshold == nil {
		return nil, fmt.Errorf("seuil introuvable pour ID: %d", config.ThresholdID)
	}

	return &config, nil
}

// ParsePGInterval convertit un intervalle au format texte PostgreSQL en time.Duration
func ParsePGInterval(interval string) (time.Duration, error) {
	interval = strings.TrimSpace(interval)
	dur, err := time.ParseDuration(interval)
	if err == nil {
		return dur, nil
	}

	parts := strings.Split(interval, ":")
	switch len(parts) {
	case 3:
		return time.ParseDuration(fmt.Sprintf("%sh%sm%ss", parts[0], parts[1], parts[2]))
	case 2:
		return time.ParseDuration(fmt.Sprintf("%sm%ss", parts[0], parts[1]))
	case 1:
		return time.ParseDuration(parts[0] + "s")
	}

	return 0, fmt.Errorf("format interval invalide: %q", interval)
}

func SaveAttemptResult(db *sql.DB, testID int64, latency, jitter, throughput float64) error {
	query := `
        INSERT INTO attempt_results (test_id, latency_ms, jitter_ms, throughput_kbps)
        VALUES ($1, $2, $3, $4)
    `
	_, err := db.Exec(query, testID, latency, jitter, throughput)
	return err
}



func UpdateTestStatus(db *sql.DB, testID int, waiting, failed, completed bool) error {
	result, err := db.Exec(`
		UPDATE test
		SET waiting = $1, failed = $2, completed = $3
		WHERE "Id" = $4
	`, waiting, failed, completed, testID)
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
		log.Printf("✅ Statut mis à jour pour test %d → waiting=%v, failed=%v, completed=%v", testID, waiting, failed, completed)
	}
	return nil
}
