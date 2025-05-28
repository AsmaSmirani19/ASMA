package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq" // driver PostgreSQL

	"mon-projet-go/server"
	 "mon-projet-go/agent"  // commenter ou supprimer si pas utilisé
)

func setupDB() (*sql.DB, error) {
	connStr := "host=localhost port=5432 user=postgres password=admin dbname=QoS_Results sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	// vérifier la connexion
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}


func main() {
	db, err := setupDB()
	if err != nil {
		log.Fatalf("Erreur ouverture DB : %v", err)
	}
	defer db.Close()

	go server.Start(db)
	go agent.Start(db)

	select {}
}
