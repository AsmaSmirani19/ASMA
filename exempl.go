package main

import (
	"fmt"
	"time"
)

func main() {
	// Simuler un horodatage
	Timestamp := time.Now().UnixNano()          // Timestamp actuel en nanosecondes
	ReceptionTimestamp := Timestamp 			 // ReceptionTimestamp simulé après 100 ms

	// Affichage des valeurs
	fmt.Printf("Timestamp: %d\n", Timestamp)
	fmt.Printf("ReceptionTimestamp: %d\n", ReceptionTimestamp)

	// Calcul de la latence
	latency := ReceptionTimestamp - Timestamp
	fmt.Printf("Latence: %d nanosecondes\n", latency)
}
