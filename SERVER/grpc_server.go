package server

import (
    "context"
    "time"
	"log"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
	"strings"
	"strconv"
    "mon-projet-go/testpb"
)


func convertToProtoProfile(p *Profile) *testpb.Profile {
    if p == nil {
        return nil
    }
    return &testpb.Profile{
        Id:              int32(p.ID),
        SendingInterval: int64(p.SendingInterval.Nanoseconds()), // conversion durée en nanosecondes
        PacketSize:      int32(p.PacketSize),
    }
}

// parseDuration convertit une chaîne comme "30s" ou "2m" en time.Duration
func parseDuration(s string) time.Duration {
    if strings.Count(s, ":") == 2 {
        parts := strings.Split(s, ":")
        h, _ := strconv.Atoi(parts[0])
        m, _ := strconv.Atoi(parts[1])
        sec, _ := strconv.Atoi(parts[2])
        totalSeconds := h*3600 + m*60 + sec
        return time.Duration(totalSeconds) * time.Second
    }

    d, err := time.ParseDuration(s)
    if err != nil {
        log.Printf("⚠️ Erreur de parsing de durée '%s' : %v", s, err)
        return 0
    }
    return d
}


func convertToProtoConfig(cfg *FullTestConfiguration) *testpb.TestConfig {
    duration := parseDuration(cfg.RawDuration).Nanoseconds()

    log.Printf("🔧 [SERVER] FullTestConfiguration reçu : %+v", cfg)

    var protoProfile *testpb.Profile
    if cfg.Profile != nil {
        protoProfile = convertToProtoProfile(cfg.Profile)
        log.Printf("📦 [SERVER] Profil converti en proto : %+v", protoProfile)
    } else {
        log.Println("⚠️ [SERVER] Avertissement : cfg.Profile est nil")
    }

    protoConfig := &testpb.TestConfig{
        TestId:         int32(cfg.TestID),
        Name:           cfg.Name,
        Duration:       duration,
        NumberOfAgents: int32(cfg.NumberOfAgents),
        SourceId:       int32(cfg.SourceID),
        SourceIp:       cfg.SourceIP,
        SourcePort:     int32(cfg.SourcePort),
        TargetId:       int32(cfg.TargetID),
        TargetIp:       cfg.TargetIP,
        TargetPort:     int32(cfg.TargetPort),
        ProfileId:      int32(cfg.ProfileID),
        Profile:        protoProfile,
    }

    log.Printf("📨 [SERVER] TestConfig prêt à l'envoi : %+v", protoConfig)

    return protoConfig
}



// Envoie la config à un agent donné (client gRPC vers agent)
func sendTestConfigToAgent(agentAddress string, config *testpb.TestConfig, testID string) error {
    conn, err := grpc.Dial(agentAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return err
    }
    defer conn.Close()

    client := testpb.NewTestServiceClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    req := &testpb.QuickTestMessage{
        Message: &testpb.QuickTestMessage_Request{
            Request: &testpb.QuickTestRequest{
                TestId: testID,
                Config: config,
            },
        },
    }

    stream, err := client.PerformQuickTest(ctx)
    if err != nil {
        return err
    }

    log.Printf("🚀 [SERVER] Envoi d'une requête vers %s : %+v", agentAddress, req)
    if err := stream.Send(req); err != nil {
        return err
    }

    // Important : attendre une réponse de l'agent
    resp, err := stream.Recv()
    if err != nil {
        log.Printf("❌ Erreur lors de la lecture de la réponse : %v", err)
        return err
    }
    log.Printf("✅ Réponse reçue de l'agent : %+v", resp)

    // Ferme proprement l'envoi après avoir reçu une réponse
    return stream.CloseSend()
}



