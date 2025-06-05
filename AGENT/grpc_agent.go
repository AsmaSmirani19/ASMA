package agent

import (
	"context"
	"fmt"
	"log"
	"net"
	"mon-projet-go/testpb"
	"google.golang.org/grpc"
	"errors"
	"io"

)


// Implémentation du service Health
type healthServer struct {
	testpb.UnimplementedHealthServer
}

func (s *healthServer) HealthCheck(ctx context.Context, req *testpb.HealthCheckRequest) (*testpb.HealthCheckResponse, error) {
	log.Println("HealthCheck reçu")
	return &testpb.HealthCheckResponse{Status: "OK"}, nil
}

// twampAgent implémente le service gRPC côté agent
type twampAgent struct {
	testpb.UnimplementedTestServiceServer
}

func startAgentServer() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", AppConfig.GRPC.Port))
	if err != nil {
		log.Fatalf("Échec écoute agent sur port %s : %v", AppConfig.GRPC.Port, err)
	}
	log.Printf("Agent gRPC démarré sur le port %s", AppConfig.GRPC.Port)

    grpcServer := grpc.NewServer()
    testpb.RegisterTestServiceServer(grpcServer, &twampAgent{})

	
	// Enregistre aussi le service Health
	testpb.RegisterHealthServer(grpcServer, &healthServer{})
	
    if err := grpcServer.Serve(listener); err != nil {
        log.Fatalf("Erreur démarrage serveur agent : %v", err)
    }
}

func ConvertProtoProfileToGo(protoProfile *testpb.Profile) *Profile {
	if protoProfile == nil {
		return nil
	}
	return &Profile{
		ID:              int(protoProfile.Id),
		SendingInterval: int64(protoProfile.SendingInterval),// nanosecondes
		PacketSize:      int(protoProfile.PacketSize),
	}
}

// ConvertProtoConfigToGo convertit la config protobuf en struct Go native
func ConvertProtoConfigToGo(protoConfig *testpb.TestConfig) TestConfig {
	profile := ConvertProtoProfileToGo(protoConfig.Profile)

	return TestConfig{
		TestID:         int(protoConfig.TestId),
		Name:           protoConfig.Name,
		Duration:       protoConfig.Duration,
		NumberOfAgents: int(protoConfig.NumberOfAgents),
		SourceID:       int(protoConfig.SourceId),
		SourceIP:       protoConfig.SourceIp,
		SourcePort:     int(protoConfig.SourcePort),
		TargetID:       int(protoConfig.TargetId),
		TargetIP:       protoConfig.TargetIp,
		TargetPort:     int(protoConfig.TargetPort),
		ProfileID:      profile.ID, // ou protoConfig.Profile.Id si tu veux garder tel quel
		Profile:        profile,
	}
}



func (a *twampAgent) PerformQuickTest(stream testpb.TestService_PerformQuickTestServer) error {
	log.Println("🟢 Agent : connexion de test rapide reçue")

	for {
		in, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Println("🔚 Agent : fin normale du stream (EOF)")
				return nil
			}
			log.Printf("❌ Agent : erreur réception message : %v", err)
			return err
		}

		switch msg := in.Message.(type) {
		case *testpb.QuickTestMessage_Request:
			cmd := msg.Request
			log.Printf("📥 Commande de test reçue : test_id = %s", cmd.TestId)

			if cmd.Config == nil {
				errMsg := "config manquante dans la requête"
				log.Printf("❌ %s", errMsg)
				return errors.New(errMsg)
			}

			if cmd.Config.Profile == nil {
				errMsg := "config reçue, mais Profile est nil, test non lancé"
				log.Printf("❌ %s", errMsg)
				continue
			}

			config := ConvertProtoConfigToGo(cmd.Config)

			go func(cfg TestConfig) {
				if err := Client(cfg); err != nil {
					log.Printf("❌ Erreur Client() : %v", err)
				}
			}(config)

			err = stream.Send(&testpb.QuickTestMessage{
				Message: &testpb.QuickTestMessage_Response{
					Response: &testpb.QuickTestResponse{
						Status: "Test lancé",
					},
				},
			})
			if err != nil {
				log.Printf("❌ Erreur envoi réponse : %v", err)
				return err
			}

		default:
			log.Println("⚠️ Type de message non reconnu")
		}
	}
}





