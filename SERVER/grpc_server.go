package server

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"

	"mon-projet-go/core"
	"mon-projet-go/testpb"
)

// *** Server démarre le serveur gRPC.
func startGRPCServer() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", AppConfig.GRPC.Port))
	if err != nil {
		log.Fatalf("Échec de l'écoute sur le port %d : %v", AppConfig.GRPC.Port, err)
	}

	grpcServer := grpc.NewServer()

	// 1️⃣ Enregistrement du service QuickTest
	testpb.RegisterTestServiceServer(grpcServer, &quickTestServer{})

	// 2️⃣ Enregistrement du service HealthCheck
	testpb.RegisterHealthServer(grpcServer, &healthServer{})

	log.Printf("✅ Serveur gRPC lancé sur le port %d...\n", AppConfig.GRPC.Port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Erreur lors du lancement du serveur gRPC : %v", err)
	}
}

// TestServiceServer
type quickTestServer struct {
	testpb.UnimplementedTestServiceServer
}

func convertTestIDToInt(testID string) (int, error) {
	return strconv.Atoi(testID)
}

func (s *quickTestServer) RunQuickTest(stream testpb.TestService_PerformQuickTestServer) error {
	log.Println("💡 Lancement du Quick Test sur le serveur...")

	// 1️⃣ Recevoir test_id
	in, err := stream.Recv()
	if err != nil {
		log.Printf("❌ Erreur lors de la réception initiale: %v", err)
		return err
	}

	req, ok := in.Message.(*testpb.QuickTestMessage_Request)
	if !ok {
		return fmt.Errorf("❌ Message initial invalide, QuickTestRequest attendu")
	}
	testID := req.Request.TestId
	log.Printf("📥 Test ID reçu : %s", testID)

	// 2️⃣ Charger la configuration de test
	db, err := core.InitDB()
	if err != nil {
		return fmt.Errorf("❌ Connexion BDD échouée: %v", err)
	}
	defer db.Close()

	testIDInt, err := convertTestIDToInt(testID)
	if err != nil {
		return fmt.Errorf("❌ Test ID invalide : %v", err)
	}

	config, err := core.LoadFullTestConfiguration(db, testIDInt)
	if err != nil {
		return fmt.Errorf("❌ Erreur chargement config test : %v", err)
	}

	// 3️⃣ Créer les paramètres à envoyer
	parameters := &testpb.TestParameters{
		SourceIp:       config.SourceIP,
		SourcePort:     int32(config.SourcePort),
		TargetIp:       config.TargetIP,
		TargetPort:     int32(config.TargetPort),
		DurationSec:    int32(config.Duration.Seconds()),
		PacketSize:     int32(config.Profile.PacketSize),
		IntervalMillis: int32(config.Profile.SendingInterval.Milliseconds()),
	}

	// 4️⃣ Envoyer QuickTestRequest à l'agent
	err = stream.Send(&testpb.QuickTestMessage{
		Message: &testpb.QuickTestMessage_Request{
			Request: &testpb.QuickTestRequest{
				TestId:     testID,
				Parameters: parameters,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("❌ Erreur d'envoi de QuickTestRequest : %v", err)
	}

	// 5️⃣ Attendre les résultats
	for {
		in, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("❌ Erreur de réception du résultat : %v", err)
		}

		if res, ok := in.Message.(*testpb.QuickTestMessage_Response); ok {
			log.Printf("✅ Résultats reçus : Latence=%.2f ms, Jitter=%.2f ms, Débit=%.2f kbps",
				res.Response.LatencyMs, res.Response.JitterMs, res.Response.ThroughputKbps)

			// 6️⃣ Sauvegarder les résultats
			err := core.SaveAttemptResult(db, int64(testIDInt), res.Response.LatencyMs, res.Response.JitterMs, res.Response.ThroughputKbps)
			if err != nil {
				log.Printf("⚠️ Erreur sauvegarde résultat : %v", err)
			}

			// 7️⃣ Mise à jour du statut
			err = core.UpdateTestStatus(db, testIDInt, false, false, true, false )
			if err != nil {
				log.Printf("⚠️ Erreur mise à jour statut : %v", err)
			}

			break // on sort après réception d'un seul résultat
		}
	}

	return nil
}
