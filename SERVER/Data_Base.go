package main

import (
	"database/sql"
	"log"
)

type QoSMetrics struct {
	PacketLossPercent float64
	AvgLatencyMs      int64
	AvgJitterMs       int64
	AvgThroughputKbps float64
	TotalJitter       int64
}

func connectToDB() (*sql.DB, error) {
	connStr := "host=localhost port=5432 user=postgres password=ton_mot_de_passe dbname=nom_de_ta_db sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Erreur de connexion :", err)
	}
	return db , nil 
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