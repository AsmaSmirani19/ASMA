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
	MBZ  uint8
	HMAC [16]byte
}
type StartAckPacket struct {
	Accept uint8
	MBZ    uint8
	HMAC   [16]byte
}
type StopSessionPacket struct {
	Accept           uint8
	MBZ              uint8
	NumberOfSessions uint8
	HMAC             [16]byte
}

// Sérialisation des paquets
func SerializePacket(packet *SendSessionRequestPacket) ([]byte, error) {
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.BigEndian, packet.SenderAddress)
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

// receivePacket_ reçoit un paquet via TCP
func receiveTCPPacket() ([]byte, error) {
	listener, err := net.Listen("tcp", ":60000")
	if err != nil {
		return nil, fmt.Errorf("erreur de création du listener: %v", err)
	}
	defer listener.Close()
	conn, err := listener.Accept()
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'acceptation de la connexion: %v", err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	buffer := make([]byte, 1500)
	n, err := conn.Read(buffer)
	if err != nil {
		// Si la lecture échoue, vérifier si c'est un timeout
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, fmt.Errorf("timeout lors de la lecture du paquet")
		}
		return nil, fmt.Errorf("erreur de lecture du paquet: %v", err)
	}
	return buffer[:n], nil
}

func identifyPacketType(data []byte) string {
	if len(data) == 45 {
		return "SessionRequest"
	} else if len(data) == 17 {
		return "StartSession"
	} else if len(data) == 19 {
		return "StopSession"
	}
	return "Unknown"
}

func client() {

	const (
		serverAddress = "127.0.0.1"
		serverPort    = 60001
		senderPort    = 6000
		receiverPort  = 5000
	)

	// 1. Envoyer le paquet Session Request
	packet := SendSessionRequestPacket{
		SenderAddress: func() [16]byte {
			var ip [16]byte
			copy(ip[:], net.ParseIP("127.0.0.1").To16())
			return ip
		}(),
		ReceiverPort:  receiverPort,
		SenderPort:    senderPort,
		PaddingLength: 0,
		StartTime:     uint32(time.Now().Unix()),
		Timeout:       30,
		TypeP:         0x00,
	}

	fmt.Println("Envoi du paquet Session Request...")
	serializedPacket, err := SerializePacket(&packet)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Session-Request : %v", err)
	}
	if err := SendTCPPacket(serializedPacket, serverAddress, serverPort); err != nil {
		log.Fatalf("Erreur envoi Session-Request : %v", err)
	}

	// 2. Réception du paquet Accept-session
	receivedData, err := receiveTCPPacket()
	if err != nil {
		log.Fatalf("Erreur lors de la réception d'Accept-session : %v", err)
	}
	fmt.Printf("Données reçues (Accept-session) : %x\n", receivedData)

	// 3. Envoyer le paquet Start-Session
	startSessionPacket := StartSessionPacket{
		MBZ:  0,
		HMAC: [16]byte{},
	}
	fmt.Println("Envoi du paquet Start Session...")
	serializedStartSessionPacket, err := SerializeStartPacket(&startSessionPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Start Session : %v", err)
	}
	if err = SendTCPPacket(serializedStartSessionPacket, serverAddress, serverPort); err != nil {
		log.Fatalf("Erreur lors de l'envoi du paquet Start Session : %v", err)
	}

	// 4. Réception du paquet Start-ACK et validation
	receivedData, err = receiveTCPPacket()
	if err != nil {
		log.Fatalf("Erreur lors de la réception Start-ACK : %v", err)
	}

	log.Println("✅ Start-ACK reçu. Déclenchement du test via Kafka...")

	// Envoi d'une commande de test à l'agent via Kafka
	SendTestRequestToKafka("START-TEST")

	// 8. Préparer le paquet Stop Session
	stopSessionPacket := StopSessionPacket{
		Accept:           0,
		MBZ:              0,
		NumberOfSessions: 1,
		HMAC:             [16]byte{},
	}
	fmt.Println("Envoi du paquet Stop Session...")
	serializedStopSessionPacket, err := SerializeStopSession(&stopSessionPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du Stop-Ack : %v", err)
	}
	if err = SendTCPPacket(serializedStopSessionPacket, serverAddress, serverPort); err != nil {
		log.Fatalf("Erreur lors de l'envoi du Stop Session : %v", err)
	}
}

func Serveur() {
	// Démarrer un serveur TCP sur le port 5001
	listener, err := net.Listen("tcp", ":60000")
	if err != nil {
		log.Fatalf("Erreur de serveur TCP : %v", err)
	}
	defer listener.Close()

	log.Println("Serveur en attente de connexions sur le port 60000...")

	for {
		// Accepter une nouvelle connexion
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Erreur d'acceptation de la connexion : %v", err)
			continue
		}
		// Gérer la connexion dans une nouvelle goroutine
		go func(conn net.Conn) {
			defer conn.Close()

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				log.Printf("Erreur de lecture de la connexion : %v", err)
				return
			}
			data := buf[:n]
			packetType := identifyPacketType(data)

			switch packetType {
			case "SessionRequest":
				log.Println("Paquet Request-Session reçu.")
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
				log.Println("Paquet Start-Session reçu.")
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
				log.Println("Paquet Stop-Session reçu.")

			default:
				log.Println("Paquet inconnu reçu.")
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
				Parameters: "ping 8.8.8.8",
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
	listener, err := net.Listen("tcp", ":50051")
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
	// Initialisation de la connexion à la base de données
	db, err := connectToDB()
	if err != nil {
		log.Fatalf("Erreur de connexion DB : %v", err)
	}
	defer db.Close()

	// Définir les routes HTTP
	http.HandleFunc("/api/test/start", startTest)
	http.HandleFunc("/api/test/results", getTestResults)
	http.HandleFunc("/api/agents", handleAgents(db))
	
	http.HandleFunc("/api/agent-group", handleAgentGroup(db))
	http.HandleFunc("/api/test-profile", handleTestProfile(db))
	http.HandleFunc("/api/threshold", handleThreshold(db))

	http.HandleFunc("/api/tests", handleTests(db))

	// Configurer CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:4200"}, // Autoriser l'origine Angular
		AllowedMethods: []string{"GET", "POST", "DELETE", "PUT"}, // Méthodes autorisées
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	// Wrap du serveur avec le handler CORS
	handler := c.Handler(http.DefaultServeMux)

	// Démarrer le serveur HTTP dans une goroutine
	go func() {
		fmt.Println("Serveur HTTP lancé sur http://localhost:5000")
		log.Fatal(http.ListenAndServe(":5000", handler))
	}()

	// Démarrer le serveur gRPC dans une goroutine
	go startGRPCServer()
	














	

	// Écouter les résultats des tests et les stocker dans la base de données
	//go listenToTestResultsAndStore(db)

	// Démarrer des serveurs ou clients
	//go Serveur()
	//go client()

	// Bloquer le programme pour qu'il reste actif
	select {}
}
