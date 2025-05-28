11  func getAgentsFromDB(db *sql.DB) ([]Agent, error) {
	rows, err := db.Query(`SELECT id, "Name", "Address", "Test_health", "Availability" FROM "Agent_List"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		err := rows.Scan(&a.ID, &a.Name, &a.Address, &a.TestHealth, &a.Availability)
		if err != nil {
			return nil, err
		}

		// Utiliser `a.ID` au lieu de `agents[i].ID` car `a` n’est pas encore ajouté à `agents`
		checks, err := GetHealthChecksByAgentID(db, a.ID)
		if err != nil {
			log.Printf("❌ Erreur récupération health checks agent %d: %v", a.ID, err)
		} else {
			a.HealthChecks = checks // Assure-toi que le type est bien []AgentHealthCheck dans la struct Agent
		}

		// Ajouter l’agent avec ses health checks
		agents = append(agents, a)
	}
	return agents, nil
}
22  // ----- Insert un nouveau health check ----- gestion d'errur 
func InsertHealthCheck(db *sql.DB, hc AgentHealthCheck) error {
	query := `INSERT INTO "agent_health_checks" (agent_id, timestamp, status) VALUES ($1, $2, $3)`
	_, err := db.Exec(query, hc.AgentID, hc.Timestamp, hc.Status)
	return err
}

func GetHealthChecksByAgentID(db *sql.DB, agentID int) ([]AgentHealthCheck, error) {
	query := `SELECT id, agent_id, timestamp, status FROM "agent_health_checks" WHERE agent_id = $1 ORDER BY timestamp DESC`
	rows, err := db.Query(query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []AgentHealthCheck
	for rows.Next() {
		var hc AgentHealthCheck
		err := rows.Scan(&hc.ID, &hc.AgentID, &hc.Timestamp, &hc.Status)
		if err != nil {
			return nil, err
		}
		checks = append(checks, hc)
	}
	return checks, nil
}

//** hediiii jawha behyyyy 

func CheckAndUpdateAllAgentsHealth(db *sql.DB) error {
    agents, err := getAgentsFromDB(db)
    if err != nil {
        return fmt.Errorf("échec récupération agents: %w", err)
    }

    for _, agent := range agents {
        // Vérification plus stricte de l'adresse
        if strings.TrimSpace(agent.Address) == "" {
            log.Printf("⚠️ Agent ID %d a une adresse vide - marqué comme indisponible", agent.ID)
            agent.TestHealth = false
            agent.Availability = 0.0
        } else {
            // Vérification que l'adresse contient un port
            if !strings.Contains(agent.Address, ":") {
                log.Printf("⚠️ Agent ID %d a une adresse mal formatée: %s", agent.ID, agent.Address)
                agent.TestHealth = false
                agent.Availability = 0.0
            } else {
                agent.TestHealth = CheckAgentHealthGRPC(agent.Address)
                agent.Availability = 100.0
                if !agent.TestHealth {
                    agent.Availability = 0.0
                }
            }
        }
        if err := updateAgentInDB(db, agent); err != nil {
            log.Printf("❌ Échec mise à jour agent ID %d: %v", agent.ID, err)
            continue
        }
        log.Printf("✅ Agent ID %d - Santé: %t - Adresse: %s", agent.ID, agent.TestHealth, agent.Address)
    }
    
    return nil
}

// Availability*******
func updateAgentAvailability(db *sql.DB, agentID int) (float64, error) {
	now := time.Now()
	oneDayAgo := now.Add(-24 * time.Hour)

	rows, err := db.Query(`
		SELECT status FROM "agent_health_checks" 
		WHERE agent_id = $1 AND timestamp >= $2
	`, agentID, oneDayAgo)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var total, success int
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return 0, err
		}
		total++
		if status == "OK" {
			success++
		}
	}
	availability := 0.0
	if total > 0 {
		availability = float64(success) / float64(total) * 100
	}

	err = updateAgentInDB(db, Agent{ID: agentID, Availability: int(availability)})
	if err != nil {
		return 0, err
	}

	return availability, nil
}



var db *sql.DB
func runHealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    go func() {
        err := CheckAndUpdateAllAgentsHealth(db)
        if err != nil {
            log.Printf("❌ Erreur lors de la vérification santé manuelle: %v", err)
        } else {
            log.Println("✅ Vérification santé manuelle terminée avec succès.")
        }
    }()

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Vérification santé lancée"))
}

type AgentService struct {
    db *sql.DB
}
func (s *AgentService) CheckAndUpdateAllAgentsHealth() error {
    agents, err := getAgentsFromDB(s.db)
    if err != nil {
        return err
    }
    for _, agent := range agents {
        isHealthy := CheckAgentHealthGRPC(agent.Address)  // test santé via gRPC
        // Crée un enregistrement de health check
        hc := AgentHealthCheck{
            AgentID:   agent.ID,
            Timestamp: time.Now(),
            Status:    "KO",
        }
        if isHealthy {
            hc.Status = "OK"
        }
        // Sauvegarde health check en DB
        err := InsertHealthCheck(s.db, hc)
        if err != nil {
            log.Printf("Erreur InsertHealthCheck: %v", err)
        }
        // Mets à jour disponibilité de l’agent
        _, err = updateAgentAvailability(s.db, agent.ID)
        if err != nil {
            log.Printf("Erreur updateAgentAvailability: %v", err)
        }
    }
    return nil
}

	// 🩺 4. Lancement périodique des vérifications de santé des agents
	agentService := &AgentService{db: db}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			log.Println("⏱️ Vérification de santé des agents...")
			err := agentService.CheckAndUpdateAllAgentsHealth()
			if err != nil {
				log.Printf("❌ Erreur vérification santé agents : %v", err)
			} else {
				log.Println("✅ Health checks mis à jour")
			}
			// attend 30s avant prochaine vérif
		}
	}()






	// Fonction pour exécuter un test et envoyer le résultat via Kafka
func runTestAndSendResult() {
	log.Println("Début du test QoS...")
	ctx := context.Background()

	// Récupérer les informations du test à partir de la configuration
	target := AppConfig.DefaultTest.TargetIP
	port := AppConfig.DefaultTest.TargetPort
	duration := AppConfig.DefaultTest.Duration
	interval := AppConfig.DefaultTest.Interval

	// Créer une chaîne avec les paramètres nécessaires
	params := fmt.Sprintf("target=%s&port=%d&duration=%s&interval=%s", target, port, duration, interval)

	// Appeler la fonction startTest avec une chaîne formatée
	stats, qos, err := startTest(params)
	if err != nil {
		log.Printf("Erreur pendant le test : %v", err)
		return
	}

	log.Println("Test terminé.")
	log.Printf("Envoyés: %d | Reçus: %d", stats.SentPackets, stats.ReceivedPackets)
	log.Printf("Latence moyenne: %f ms", qos.AvgLatencyMs)
	log.Printf("Jitter moyen: %f ms", qos.AvgJitterMs)

	// Construction de l'objet de résultat
	result := TestResult{
		AgentID:           AppConfig.Sender.ID,
		Target:            AppConfig.DefaultTest.TargetIP,
		Port:              AppConfig.DefaultTest.TargetPort,
		AvgThroughputKbps: qos.AvgThroughputKbps,
		AvgLatencyMs:      qos.AvgLatencyMs,
		AvgJitterMs:       qos.AvgJitterMs,
		PacketLossPercent: qos.PacketLossPercent,
	}

	// Sérialisation en JSON
	resultBytes, err := json.Marshal(result)
	if err != nil {
		log.Printf("Erreur lors de la sérialisation du résultat : %v", err)
		return
	}

	// Envoi via Kafka
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  AppConfig.Kafka.Brokers,
		Topic:    AppConfig.Kafka.TestResultTopic,
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte("test-result"),
		Value: resultBytes,
	})
	if err != nil {
		log.Printf("Erreur lors de l'envoi du résultat via Kafka : %v", err)
		return
	}

	log.Println("Résultat du test envoyé au backend via Kafka.")
}









*****************************************//////********************************

// RunQuickTest avec gestion de contexte et streaming
func (a *twampAgent) PerformQuickTest(stream testpb.TestService_PerformQuickTestServer) error {
	// Réception d'un message (devrait être une requête)
	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("échec réception message: %v", err)
	}

	// Extraction de la requête de test
	reqMsg, ok := msg.Message.(*testpb.QuickTestMessage_Request)
	if !ok {
		return fmt.Errorf("message reçu n’est pas une requête de test")
	}

	req := reqMsg.Request

	log.Printf("Nouveau test reçu - ID: %s, Paramètres: %s", req.GetTestId(), req.GetParameters())

	// Création d'un contexte annulable pour le test
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	a.mu.Lock()
	if a.currentTestCancel != nil {
		a.currentTestCancel() // Annule tout test précédent
	}
	a.currentTestCancel = cancel
	a.mu.Unlock()

	// Canal pour les résultats
	results := make(chan *QoSMetrics, 10)

	// Exécution du test dans une goroutine
	go func() {
		defer close(results)
		_, metrics, err := StartTest(req.GetParameters())
		if err != nil {
			log.Printf("Test %s échoué: %v", req.GetTestId(), err)
			return
		}
		results <- metrics
	}()

	// Envoi des résultats au serveur
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case qos, ok := <-results:
			if !ok {
				return nil
			}
			respMsg := &testpb.QuickTestMessage{
				Message: &testpb.QuickTestMessage_Response{
					Response: &testpb.QuickTestResponse{
						Status: testpb.TestStatus_COMPLETE.String(),
						Result: fmt.Sprintf("loss:%.2f,latency:%.2f,throughput:%.2f",
							float64(qos.PacketLossPercent),
							float64(qos.AvgLatencyMs),
							float64(qos.AvgThroughputKbps)),
					},
				},
			}

			if err := stream.Send(respMsg); err != nil {
				return fmt.Errorf("échec envoi résultats: %v", err)
			}
		}
	}
}




func startGRPCServer() {
	lis, err := net.Listen("tcp", AppConfig.GRPC.Port)
	if err != nil {
		log.Fatalf("Échec d'écoute : %v", err)
	}

	grpcServer := grpc.NewServer()
	testpb.RegisterTestServiceServer(grpcServer, &twampAgent{})

	log.Println("Agent TWAMP (serveur gRPC) démarré sur le port 50052")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Échec du serveur gRPC : %v", err)
	}
}