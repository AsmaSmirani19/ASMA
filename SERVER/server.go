package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"mon-projet-go/testpb"
	"net"
	"net/http"
	"time"

	"io"

	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"

	"github.com/rs/cors"
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
	Type   byte
	MBZ   uint8
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

// sérialition accept session
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

// sérialition start session
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


// Sérialisation start-ack
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

func client() {

	var (
		serverAddress = AppConfig.Network.ServerAddress
		serverPort    = AppConfig.Network.ServerPort
		senderPort    = AppConfig.Network.SenderPort
		receiverPort  = AppConfig.Network.ReceiverPort
		timeout       = AppConfig.Network.Timeout
	)

	// 🔁 Connexion TCP unique au serveur
	conn, err := net.Dial("tcp", fmt.Sprintf("[%s]:%d", serverAddress, serverPort))
	if err != nil {
		log.Fatalf("Erreur de connexion au serveur TCP : %v", err)
	}
	defer conn.Close()

	// 1. Envoyer Session-Request
	packet := SendSessionRequestPacket{
		Type:          PacketTypeSessionRequest,
		SenderAddress: func() [16]byte {
			var ip [16]byte
			copy(ip[:], net.ParseIP("127.0.0.1").To16())
			return ip
		}(),
		ReceiverPort:  uint16(receiverPort),
		SenderPort:    uint16(senderPort),
		PaddingLength: 0,
		StartTime:     uint32(time.Now().Unix()),
		Timeout:       uint32(timeout),
		TypeP:         0x05,
	}
	log.Println("Envoi Session-Request...")
	serializedPacket, err := SerializePacket(&packet)
	if err != nil {
		log.Fatalf("Erreur de sérialisation : %v", err)
	}
	_, err = conn.Write(serializedPacket)
	if err != nil {
		log.Fatalf("Erreur d'envoi du Session-Request : %v", err)
	}

	// 2. Lire Accept-Session
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Fatalf("Erreur de lecture (Accept-Session) : %v", err)
	}
	log.Printf("Données reçues (Accept-session) : %x", buffer[:n])

	// 3. Envoyer Start-Session
	startSessionPacket := StartSessionPacket{
		Type: PacketTypeStartSession,
		MBZ:  0,
		HMAC: [16]byte{},
	}
	log.Println("Envoi Start-Session...")
	serializedStart, err := SerializeStartPacket(&startSessionPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation Start-Session : %v", err)
	}
	_, err = conn.Write(serializedStart)
	if err != nil {
		log.Fatalf("Erreur d'envoi Start-Session : %v", err)
	}

	// 4. Lire Start-Ack
	n, err = conn.Read(buffer)
	if err != nil {
		log.Fatalf("Erreur de lecture (Start-Ack) : %v", err)
	}
	log.Println("✅ Start-Ack reçu. Déclenchement test via Kafka...")
	SendTestRequestToKafka("START-TEST")

	// 5. Envoyer Stop-Session
	stopSessionPacket := StopSessionPacket{
		Type:             PacketTypeStopSession,
		Accept:           0,
		MBZ:              0,
		NumberOfSessions: 1,
		HMAC:             [16]byte{},
	}
	log.Println("Envoi Stop-Session...")
	serializedStop, err := SerializeStopSession(&stopSessionPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation Stop-Session : %v", err)
	}
	_, err = conn.Write(serializedStop)
	if err != nil {
		log.Fatalf("Erreur d'envoi Stop-Session : %v", err)
	}
}

func Serveur() {
	// Démarrer un serveur TCP sur le port configuré
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", AppConfig.Network.ServerPort))
	if err != nil {
		log.Fatalf("Erreur de serveur TCP : %v", err)
	}
	defer listener.Close()

	log.Printf("Serveur en attente de connexions sur le port %d...\n", AppConfig.Network.ServerPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Erreur d'acceptation de la connexion : %v", err)
			continue
		}

		// Gérer la connexion dans une nouvelle goroutine
		go func(conn net.Conn) {
			defer conn.Close()

			buf := make([]byte, 1024)

			for {
				n, err := conn.Read(buf)
				if err != nil {
					if err == io.EOF {
						log.Println("Connexion fermée par le client.")
					} else {
						log.Printf("Erreur de lecture de la connexion : %v", err)
					}
					return
				}

				data := buf[:n]
				packetType := identifyPacketType(data)

				switch packetType {
				case "SessionRequest":
					log.Println("Paquet SessionRequest reçu.")
					acceptSessionPacket := SessionAcceptPacket{
						Accept: 0,
						MBZ:    0,
						HMAC:   [16]byte{},
					}
					serializedPacket, err := SerializeAcceptPacket(&acceptSessionPacket)
					if err != nil {
						log.Printf("Erreur de sérialisation du paquet Accept-Session : %v", err)
						return
					}
					_, err = conn.Write(serializedPacket)
					if err != nil {
						log.Printf("Erreur d'envoi du paquet Accept-Session : %v", err)
						return
					}
					log.Println("Paquet Accept-Session envoyé.")

				case "StartSession":
					log.Println("Paquet StartSession reçu.")
					startAckPacket := StartAckPacket{
						Accept: 0,
						MBZ:    0,
						HMAC:   [16]byte{},
					}
					serializedStartAckPacket, err := SerializeStartACKtPacket(&startAckPacket)
					if err != nil {
						log.Printf("Erreur de sérialisation du paquet Start-Ack : %v", err)
						return
					}
					_, err = conn.Write(serializedStartAckPacket)
					if err != nil {
						log.Printf("Erreur d'envoi du paquet Start-Ack : %v", err)
						return
					}
					log.Println("Paquet Start-Ack envoyé.")

				case "StopSession":
					log.Println("Paquet StopSession reçu.")
					log.Println("Session terminée.")
					return // Fermer la session proprement

				default:
					log.Println("Paquet inconnu reçu.")
				}
			}
		}(conn)
	}
}

type quickTestServer struct {
	testpb.UnimplementedTestServiceServer
}

//TestServiceServer

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

// startGRPCServer démarre le serveur gRPC.
func startGRPCServer() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", AppConfig.GRPC.Port))
	if err != nil {
		log.Fatalf("Échec de l'écoute sur le port 50051 : %v", err)
	}

	grpcServer := grpc.NewServer()
	testpb.RegisterTestServiceServer(grpcServer, &quickTestServer{})

	log.Println("Serveur gRPC lancé sur le port 50051...")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Erreur lors du lancement du serveur gRPC : %v", err)
	}

}

func main() {
	LoadConfig("config_server.yaml")

	// 1. 📡 Lancement du serveur WebSocket sur un port séparé
	go StartWebSocketServer()

	// 2. 🔌 Connexion à la base de données
	db, err := connectToDB()
	if err != nil {
		log.Fatalf("Erreur de connexion DB : %v", err)
	}
	defer db.Close()

	// 3. 🌐 Définition des routes REST HTTP
	http.HandleFunc("/api/test/start", startTest)
	http.HandleFunc("/api/test/results", getTestResults)
	http.HandleFunc("/api/agents", handleAgents(db))
	http.HandleFunc("/api/agent-group", handleAgentGroup(db))
	http.HandleFunc("/api/test-profile", handleTestProfile(db))
	http.HandleFunc("/api/threshold", handleThreshold(db))
	http.HandleFunc("/api/tests", handleTests(db))

	// 4. 🌍 Configuration CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:4200"},
		AllowedMethods: []string{"GET", "POST", "DELETE", "PUT"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})
	handler := c.Handler(http.DefaultServeMux)

	// 5. 🚀 Lancement du serveur HTTP (REST API)
	go func() {
		fmt.Println("🌐 Serveur HTTP lancé sur http://localhost:5000")
		log.Fatal(http.ListenAndServe(":5000", handler))
	}()

	// 6. 🚀 Lancement du serveur gRPC
	go startGRPCServer()

	// 7. 🎧 Écoute des résultats de test
	go listenToTestResultsAndStore(db)

	// 8. 🔄 Démarrage de composants spécifiques (testeurs, etc.)
	go Serveur()                // Lancement du listener d’abord
	time.Sleep(1 * time.Second) // Attente pour s’assurer que le port est bien en écoute
	go client()                 // Ensuite envoyer le paquet vers 61000

	// 9. 🛑 Empêche le programme de se terminer
	select {}
}
