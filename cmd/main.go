package main

import (
	"database/sql"
	"fmt"
	"log"
	
	"mon-projet-go/server"


	"mon-projet-go/agent"
	"mon-projet-go/core"



	_ "github.com/lib/pq"
)

func main() {
	connStr := "host=localhost port=5432 user=postgres password=admin dbname=QoS_Results sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ Erreur connexion DB : %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Impossible de se connecter à la base : %v", err)
	}

	// Start de l'agent TWAMP avec la DB
	go agent.Start(db)

	testHandler := func(config agent.TestConfig) {
		// Récupérer la config complète à partir de TestID reçu
		fullConfig, err := server.GetFullTestConfig(db, config.TestID)
		if err != nil {
			log.Println("❌ Erreur récupération config complète :", err)
			return
		}

		fmt.Printf("📡 Source: %s:%d\n", fullConfig.SourceIP, fullConfig.SourcePort)
		fmt.Printf("🎯 Cible: %s:%d\n", fullConfig.TargetIP, fullConfig.TargetPort)

		// Ici on utilise la conversion personnalisée pour l'interval
		duration, err := core.ParsePGInterval(config.Duration)
		if err != nil {
			log.Println("❌ Erreur conversion durée :", err)
			return
		}

		// Juste afficher ou loguer la durée convertie, si nécessaire
		_ = duration.String() // si tu veux l'utiliser, sinon tu peux aussi enlever

		// Appel direct avec TestID et DB
		server.Client(config.TestID, db)
	
	}

	log.Println("✅ Démarrage de l’écoute Kafka...")
	go agent.ListenToTestRequestsFromKafka(db, testHandler)

	log.Println("🚀 Serveur en cours de démarrage...")
	
	go server.Start(db)

	select {}
}
