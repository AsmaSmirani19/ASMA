
package agent

import (
	"context"
	"log"
	
	"mon-projet-go/testpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)



// *** etablit la cnx avec le serveur 
 func startClientStream() {
	conn, err := grpc.Dial(
		AppConfig.Server1.Main,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Échec de connexion au serveur principal : %v", err)
	}
	defer conn.Close()

	client := testpb.NewTestServiceClient(conn)
	stream, err := client.PerformQuickTest(context.Background())
	if err != nil {
		log.Fatalf("Échec de création du stream : %v", err)
	}

	log.Println("Connexion au serveur principal établie")

	// Boucle pour maintenir la connexion ouverte
	for {
		select {
		case <-stream.Context().Done():
			log.Println("Connexion au serveur terminée")
			return
		}
	}
}


type TestResults struct {
	Latency    float64
	Jitter     float64
	Throughput float64
}


type twampAgent struct {
	testpb.UnimplementedTestServiceServer

	// Callback appelé pour lancer le test avec les paramètres reçus
	TestCallback func(params *testpb.TestParameters) (TestResults, error)
}
 
func (a *twampAgent) PerformQuickTest(stream testpb.TestService_PerformQuickTestServer) error {
	log.Println("🟢 Agent : connexion de test rapide reçue")

	for {
		in, err := stream.Recv()
		if err != nil {
			log.Printf("❌ Agent : erreur réception message : %v", err)
			return err
		}

		switch msg := in.Message.(type) {
		case *testpb.QuickTestMessage_Request:
			cmd := msg.Request

			log.Printf("📥 Commande de test reçue : test_id = %s", cmd.TestId)

			if a.TestCallback == nil {
				log.Println("⚠️ Aucun callback défini pour lancer le test")
				return nil
			}

			// Appel au callback avec les paramètres reçus
			results, err := a.TestCallback(cmd.Parameters)
			if err != nil {
				log.Printf("❌ Erreur exécution test client : %v", err)
				return err
			}

			// Envoyer les résultats
			err = stream.Send(&testpb.QuickTestMessage{
				Message: &testpb.QuickTestMessage_Response{
					Response: &testpb.QuickTestResponse{
						LatencyMs:      results.Latency,
						JitterMs:       results.Jitter,
						ThroughputKbps: results.Throughput,
					},
				},
			})
			if err != nil {
				log.Printf("❌ Erreur envoi résultats au serveur : %v", err)
				return err
			}

			log.Println("✅ Résultats envoyés au serveur.")
			// continue la boucle pour traiter d'autres commandes

		default:
			log.Println("⚠️ Type de message non reconnu")
		}
	}
}
