package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mon-projet-go/testpb"
	"net"

	"net/http"

	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"

	"mon-projet-go/agent"
	"mon-projet-go/core"

	"github.com/rs/cors"

	"github.com/gorilla/websocket"
)

type TestResult struct {
	AgentID    string
	Timestamp  time.Time
	Latency    float64
	Loss       float64
	Throughput float64
}

// structure des paquets
type SendSessionRequestPacket struct {
	Type          byte
	SenderAddress [16]byte
	ReceiverPort  uint16
	SenderPort    uint16
	PaddingLength uint32
	StartTime     uint32
	Timeout       uint32
	TypeP         uint8
}

type SessionAcceptPacket struct {
	Accept         uint8
	MBZ            uint8
	Port           uint16
	ReflectedOctet [16]byte
	ServerOctets   [16]byte
	SID            uint32
	HMAC           [16]byte
}

type StartSessionPacket struct {
	Type byte
	MBZ  uint8
	HMAC [16]byte
}

type StartAckPacket struct {
	Accept uint8
	MBZ    uint8
	HMAC   [16]byte
}

type StopSessionPacket struct {
	Type             byte
	Accept           uint8
	MBZ              uint8
	NumberOfSessions uint8
	HMAC             [16]byte
}

const (
	PacketTypeSessionRequest = 0x01
	PacketTypeSessionAccept  = 0x02
	PacketTypeStartSession   = 0x03
	PacketTypeStartAck       = 0x04
	PacketTypeStopSession    = 0x05
)

// Sérialisation des paquets
func SerializePacket(packet *SendSessionRequestPacket) ([]byte, error) {
	buf := new(bytes.Buffer)

	// 1. Écrire le champ Type comme premier octet
	err := binary.Write(buf, binary.BigEndian, packet.Type)
	if err != nil {
		return nil, err
	}

	// 2. Puis le reste du paquet
	err = binary.Write(buf, binary.BigEndian, packet.SenderAddress)
	if err != nil {
		return nil, err
	}

	fields := []interface{}{
		packet.ReceiverPort,
		packet.SenderPort,
		packet.PaddingLength,
		packet.StartTime,
		packet.Timeout,
		packet.TypeP,
	}
	for _, field := range fields {
		err := binary.Write(buf, binary.BigEndian, field)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func SerializeAcceptPacket(packet *SessionAcceptPacket) ([]byte, error) {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		packet.Accept,
		packet.MBZ,
		packet.Port,
		packet.ReflectedOctet,
		packet.ServerOctets,
		packet.SID,
		packet.HMAC,
	}
	for _, field := range fields {
		err := binary.Write(buf, binary.BigEndian, field)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func SerializeStartPacket(packet *StartSessionPacket) ([]byte, error) {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		packet.Type,
		packet.MBZ,
		packet.HMAC,
	}
	for _, field := range fields {
		err := binary.Write(buf, binary.BigEndian, field)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func SerializeStartACKtPacket(packet *StartAckPacket) ([]byte, error) {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		packet.Accept,
		packet.MBZ,
		packet.HMAC,
	}
	for _, field := range fields {
		err := binary.Write(buf, binary.BigEndian, field)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func SerializeStopSession(packet *StopSessionPacket) ([]byte, error) {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		packet.Type,
		packet.MBZ,
		packet.HMAC,
		packet.NumberOfSessions,
	}
	for _, field := range fields {
		err := binary.Write(buf, binary.BigEndian, field)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func SendTCPPacket(packet []byte, addr string, port int) error {

	timeout := 5 * time.Second

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("[%s]:%d", addr, port), timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	if err != nil {
		return fmt.Errorf("échec d'envoi du paquet: %w", err)
	}
	return nil
}

func identifyPacketType(data []byte) string {
	if len(data) < 1 {
		return "Unknown"
	}

	switch data[0] {
	case PacketTypeSessionRequest:
		return "SessionRequest"
	case PacketTypeSessionAccept:
		return "SessionAccept"
	case PacketTypeStartSession:
		return "StartSession"
	case PacketTypeStartAck:
		return "StartAck"
	case PacketTypeStopSession:
		return "StopSession"
	default:
		return "Unknown"
	}
}

func Client(testID int, db *sql.DB) {
	log.Println("🔵 [Client] Début d'exécution du client...")

	config, err := core.LoadFullTestConfiguration(db, testID)
	if err != nil {
		log.Fatalf("❌ [Client] Erreur chargement configuration test : %v", err)
	}
	log.Printf("✅ [Client] Configuration chargée : %+v", config)

	serverAddress := AppConfig.Network.ServerAddress
	serverPort := AppConfig.Network.ServerPort
	senderPort := AppConfig.Network.SenderPort
	receiverPort := AppConfig.Network.ReceiverPort
	timeout := AppConfig.Network.Timeout

	log.Printf("🔌 [Client] Connexion à %s:%d ...", serverAddress, serverPort)
	conn, err := net.Dial("tcp", fmt.Sprintf("[%s]:%d", serverAddress, serverPort))
	if err != nil {
		log.Fatalf("❌ [Client] Erreur de connexion au serveur TCP : %v", err)
	}
	defer conn.Close()
	log.Println("✅ [Client] Connexion TCP établie.")

	// 1. Envoi Session-Request
	packet := SendSessionRequestPacket{
		Type: PacketTypeSessionRequest,
		SenderAddress: func() [16]byte {
			var ip [16]byte
			copy(ip[:], net.ParseIP(config.SourceIP).To16())
			return ip
		}(),
		ReceiverPort:  uint16(receiverPort),
		SenderPort:    uint16(senderPort),
		PaddingLength: 0,
		StartTime:     uint32(time.Now().Unix()),
		Timeout:       uint32(timeout),
		TypeP:         0x05,
	}

	log.Println("📤 [Client] Envoi Session-Request...")
	serializedPacket, err := SerializePacket(&packet)
	if err != nil {
		log.Fatalf("❌ [Client] Erreur de sérialisation Session-Request : %v", err)
	}
	log.Printf("📦 [Client] Paquet Session-Request (hex) : %x", serializedPacket)
	_, err = conn.Write(serializedPacket)
	if err != nil {
		log.Fatalf("❌ [Client] Erreur d'envoi du Session-Request : %v", err)
	}

	// 2. Lire Accept-Session
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Fatalf("❌ [Client] Erreur de lecture (Accept-Session) : %v", err)
	}
	log.Printf("📥 [Client] Données reçues (Accept-session) : %x", buffer[:n])

	// 3. Envoyer Start-Session
	startSessionPacket := StartSessionPacket{
		Type: PacketTypeStartSession,
		MBZ:  0,
		HMAC: [16]byte{},
	}
	log.Println("📤 [Client] Envoi Start-Session...")
	serializedStart, err := SerializeStartPacket(&startSessionPacket)
	if err != nil {
		log.Fatalf("❌ [Client] Erreur de sérialisation Start-Session : %v", err)
	}
	log.Printf("📦 [Client] Paquet Start-Session (hex) : %x", serializedStart)
	_, err = conn.Write(serializedStart)
	if err != nil {
		log.Fatalf("❌ [Client] Erreur d'envoi Start-Session : %v", err)
	}

	// 4. Lire Start-Ack
	n, err = conn.Read(buffer)
	if err != nil {
		log.Fatalf("❌ [Client] Erreur de lecture (Start-Ack) : %v", err)
	}
	log.Printf("📥 [Client] Start-Ack reçu : %x", buffer[:n])

	// 5. Démarrer le test
	log.Println("🚀 [Client] Lancement du test agent.StartTest...")
	var conn2 *websocket.Conn
	stats, qos, err := agent.StartTest(db, config.TestID, conn2)
	if err != nil {
		log.Fatalf("❌ [Client] Erreur startTest : %v", err)
	}
	log.Printf("✅ [Client] Test terminé. Stats : %+v | QoS : %+v", stats, qos)

	// 6. Envoyer Stop-Session
	stopSessionPacket := StopSessionPacket{
		Type:             PacketTypeStopSession,
		Accept:           0,
		MBZ:              0,
		NumberOfSessions: 1,
		HMAC:             [16]byte{},
	}
	log.Println("📤 [Client] Envoi Stop-Session...")
	serializedStop, err := SerializeStopSession(&stopSessionPacket)
	if err != nil {
		log.Fatalf("❌ [Client] Erreur de sérialisation Stop-Session : %v", err)
	}
	log.Printf("📦 [Client] Paquet Stop-Session (hex) : %x", serializedStop)
	_, err = conn.Write(serializedStop)
	if err != nil {
		log.Fatalf("❌ [Client] Erreur d'envoi Stop-Session : %v", err)
	}

	log.Println("✅ [Client] Fin du client.")
}

func Serveur() {
	log.Println("🟢 [Serveur] Démarrage du serveur...")

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", AppConfig.Network.ServerPort))
	if err != nil {
		log.Fatalf("❌ [Serveur] Erreur de serveur TCP : %v", err)
	}
	defer listener.Close()
	log.Printf("✅ [Serveur] En attente de connexions sur le port %d...\n", AppConfig.Network.ServerPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("❌ [Serveur] Erreur d'acceptation de la connexion : %v", err)
			continue
		}
		log.Println("🔗 [Serveur] Nouvelle connexion acceptée.")

		go func(conn net.Conn) {
			defer conn.Close()
			buf := make([]byte, 1024)

			for {
				n, err := conn.Read(buf)
				if err != nil {
					if err == io.EOF {
						log.Println("📴 [Serveur] Connexion fermée par le client.")
					} else {
						log.Printf("❌ [Serveur] Erreur de lecture : %v", err)
					}
					return
				}

				data := buf[:n]
				log.Printf("📦 [Serveur] Paquet brut reçu : %x", data)

				packetType := identifyPacketType(data)
				log.Printf("🔍 [Serveur] Type de paquet identifié : %s", packetType)

				switch packetType {
				case "SessionRequest":
					log.Println("✅ [Serveur] Paquet SessionRequest reçu.")
					acceptSessionPacket := SessionAcceptPacket{
						Accept: 0,
						MBZ:    0,
						HMAC:   [16]byte{},
					}
					serializedPacket, err := SerializeAcceptPacket(&acceptSessionPacket)
					if err != nil {
						log.Printf("❌ [Serveur] Erreur de sérialisation Accept-Session : %v", err)
						return
					}
					log.Printf("📤 [Serveur] Envoi Accept-Session : %x", serializedPacket)
					_, err = conn.Write(serializedPacket)
					if err != nil {
						log.Printf("❌ [Serveur] Erreur d'envoi Accept-Session : %v", err)
						return
					}
					log.Println("✅ [Serveur] Accept-Session envoyé.")

				case "StartSession":
					log.Println("✅ [Serveur] Paquet StartSession reçu.")
					startAckPacket := StartAckPacket{
						Accept: 0,
						MBZ:    0,
						HMAC:   [16]byte{},
					}
					serializedStartAckPacket, err := SerializeStartACKtPacket(&startAckPacket)
					if err != nil {
						log.Printf("❌ [Serveur] Erreur de sérialisation Start-Ack : %v", err)
						return
					}
					log.Printf("📤 [Serveur] Envoi Start-Ack : %x", serializedStartAckPacket)
					_, err = conn.Write(serializedStartAckPacket)
					if err != nil {
						log.Printf("❌ [Serveur] Erreur d'envoi Start-Ack : %v", err)
						return
					}
					log.Println("✅ [Serveur] Start-Ack envoyé.")

				case "StopSession":
					log.Println("✅ [Serveur] Paquet StopSession reçu.")
					log.Println("✅ [Serveur] Session terminée proprement.")
					return

				default:
					log.Printf("⚠️ [Serveur] Paquet inconnu ou type non géré. Données : %x", data)
				}
			}
		}(conn)
	}
}

// TestServiceServer
type quickTestServer struct {
	testpb.UnimplementedTestServiceServer
}

// Fonction qui lance un Quick Test
func (s *quickTestServer) RunQuickTest(stream testpb.TestService_PerformQuickTestServer) error {
	log.Println("Lancement du Quick Test sur le serveur...")

	// Envoie d’une requête (commande de test)
	testCmd := &testpb.QuickTestMessage{
		Message: &testpb.QuickTestMessage_Request{
			Request: &testpb.QuickTestRequest{
				TestId:     "test_001",
				Parameters: AppConfig.QuickTest.Parameters, // ← ici avec ":"
			},
		},
	}

	if err := stream.Send(testCmd); err != nil {
		log.Printf("Erreur d'envoi de la commande: %v", err)
		return err
	}

	// Attente de la réponse du client
	for {
		in, err := stream.Recv()
		if err != nil {
			log.Printf("Erreur lors de la réception: %v", err)
			return err
		}
		// Vérifie si c'est une réponse
		if res, ok := in.Message.(*testpb.QuickTestMessage_Response); ok {
			log.Printf("Statut: %s, Résultat: %s", res.Response.Status, res.Response.Result)
			break
		}
	}

	return nil
}

func listenToTestResultsAndStore(db *sql.DB) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-results",
		GroupID: "backend-group",
	})
	defer reader.Close()

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Erreur Kafka : %v", err)
			continue
		}

		var result TestResult
		if err := json.Unmarshal(msg.Value, &result); err != nil {
			log.Printf("Erreur JSON : %v", err)
			continue
		}

		if err := saveResultsToDB(db, QoSMetrics{}); err != nil {
			log.Printf("Erreur DB : %v", err)
		} else {
			log.Printf("Résultat stocké avec succès : %+v", result)
		}
	}
}

// Implémentation du service Health côté serveur
type healthServer struct {
	testpb.UnimplementedHealthServer
}

// Méthode HealthCheck appelée par l'agent
func (s *healthServer) HealthCheck(ctx context.Context, req *testpb.HealthCheckRequest) (*testpb.HealthCheckResponse, error) {
	log.Println("Reçu une requête HealthCheck de l'agent")
	return &testpb.HealthCheckResponse{Status: "OK"}, nil
}

// startGRPCServer démarre le serveur gRPC.
func startGRPCServer() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", AppConfig.GRPC.Port))
	if err != nil {
		log.Fatalf("Échec de l'écoute sur le port %d : %v", AppConfig.GRPC.Port, err)
	}

	grpcServer := grpc.NewServer()

	// 1️⃣ Enregistrement du service QuickTest
	testpb.RegisterTestServiceServer(grpcServer, &quickTestServer{})

	// 2️⃣ Enregistrement du service HealthCheck
	testpb.RegisterHealthServer(grpcServer, &healthServer{}) // <-- C'est ça qui manquait

	log.Printf("✅ Serveur gRPC lancé sur le port %d...\n", AppConfig.GRPC.Port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Erreur lors du lancement du serveur gRPC : %v", err)
	}
}

func Start(db *sql.DB) {

	// 🔧 1. Chargement de la configuration
	LoadConfig("server/config_server.yaml")

	// 📡 2. Lancement du serveur WebSocket en arrière-plan
	go StartWebSocketServer()

	// 🌐 4. Définition des routes HTTP
	http.HandleFunc("/api/test/results", getTestResults)
	http.HandleFunc("/api/agents", handleAgents(db))
	http.HandleFunc("/api/agent-group", handleAgentGroup(db))
	http.HandleFunc("/api/agent_link", handleAgentLink(db))
	http.HandleFunc("/api/test-profile", handleTestProfile(db))
	http.HandleFunc("/api/threshold", handleThreshold(db))
	http.HandleFunc("/api/tests", handleTests(db))
	http.HandleFunc("/api/trigger-test", triggerTestHandler(db))
	//http.HandleFunc("/api/test-results", handleTestResults(db))

	//http.HandleFunc("/ws/health", healthWebSocketHandler)
	http.HandleFunc("/api/test-results", handleGetAllTests)


	// 🌍 5. Middleware CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:4200"},
		AllowedMethods: []string{"GET", "POST", "DELETE", "PUT"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})
	handler := c.Handler(http.DefaultServeMux)

	// 🚀 6. Lancement du serveur HTTP
	go func() {
		fmt.Println("🌐 Serveur HTTP lancé sur http://localhost:5000")
		log.Fatal(http.ListenAndServe(":5000", handler))
	}()

	// 🚀 7. Lancement du serveur gRPC
	go startGRPCServer()

	// 8. Création du service
	agentService := &AgentService{db: db}
	agentService.CheckAllAgents()

	// 🎧 9. Écoute des résultats de tests TWAMP
	go listenToTestResultsAndStore(db)

	// 🧪 10. Lancement du serveur et client TWAMP
	go Serveur()
	time.Sleep(1 * time.Second) // délai pour laisser le serveur démarrer

	// 🛑 11. Blocage principal pour garder le serveur actif
	select {}
}
