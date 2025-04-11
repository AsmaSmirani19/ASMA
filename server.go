package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"mon-projet-go/testpb"
	"net"

	"time"

	"google.golang.org/grpc"

	"google.golang.org/grpc/credentials/insecure"
)

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
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	if err != nil {
		return err
	}

	return nil
}

// receivePacket_ reçoit un paquet via TCP
func receiveTCPPacket() ([]byte, error) {
	listener, err := net.Listen("tcp", ":60000")
	if err != nil {
		return nil, err
	}
	defer listener.Close()
	conn, err := listener.Accept()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	buffer := make([]byte, 1500)
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}

func identifyPacketType(data []byte) string {
	// Simuler l'identification selon la taille
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
	// 1. Envoyer le paquet Session Request
	packet := SendSessionRequestPacket{
		SenderAddress: func() [16]byte {
			var ip [16]byte
			copy(ip[:], net.ParseIP("127.0.0.1").To16())
			return ip
		}(),
		ReceiverPort:  5000,
		SenderPort:    6000,
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
	if err := SendTCPPacket(serializedPacket, "127.0.0.1", 60001); err != nil {
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
	if err = SendTCPPacket(serializedStartSessionPacket, "127.0.0.1", 60001); err != nil {
		log.Fatalf("Erreur lors de l'envoi du paquet Start Session : %v", err)
	}
	// 4. Réception du paquet Start-ACK et validation
	receivedData, err = receiveTCPPacket()
	if err != nil {
		log.Fatalf("Erreur lors de la réception Start-ACK : %v", err)
	}

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
	if err = SendTCPPacket(serializedStopSessionPacket, "127.0.0.1", 60001); err != nil {
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

// Implémentation du service gRPC
type server struct {
	testpb.UnimplementedTestServiceServer
}

// Implémentation du serveur qui lance le test dès la connexion avec l'agent
func (s *server) startTestWithAgent() {
	const (
		initialReconnectDelay = 1 * time.Second
		maxReconnectDelay     = 30 * time.Second
		connectionTimeout     = 10 * time.Second
	)

	var reconnectDelay time.Duration = initialReconnectDelay

	for {
		// Création d'un contexte avec timeout pour la connexion
		connCtx, connCancel := context.WithTimeout(context.Background(), connectionTimeout)
		conn, err := grpc.DialContext(
			connCtx,
			"localhost:50051", // Adresse de l'agent
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		connCancel() // Libérer les ressources du contexte

		if err != nil {
			log.Printf("Connexion impossible: %v - Nouvelle tentative dans %v", err, reconnectDelay)
			time.Sleep(reconnectDelay)

			// Backoff exponentiel
			reconnectDelay = min(reconnectDelay*2, maxReconnectDelay)
			continue
		}

		// Réinitialiser le délai de reconnexion après une connexion réussie
		reconnectDelay = initialReconnectDelay

		client := testpb.NewTestServiceClient(conn)

		// Envoi immédiat d'une commande de test à l'agent
		testCommand := &testpb.TestCommand{
			TestId:     "START_TEST",
			Parameters: "test_parameters_here", // Paramètres du test (exemple)
		}

		res, err := client.StreamTestCommands(context.Background(), testCommand)
		if err != nil {
			log.Printf("Erreur lors de l'exécution du test: %v", err)
			conn.Close()
			continue
		}

		log.Printf("Test lancé avec succès. Résultat: %s, Statut: %s", res.Result)

		conn.Close()
	}
}

// Fonction utilitaire pour la valeur minimale
func min1(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func main() {

	// Démarrage du serveur gRPC
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Échec de l'écoute : %v", err)
	}

	grpcServer := grpc.NewServer()
	testpb.RegisterTestServiceServer(grpcServer, &server{})

	go StartWebSocketServer()

	go Serveur()
	go client()

	log.Println("Serveur gRPC démarré sur le port 50051...")

	// Démarrer le serveur gRPC
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Échec du démarrage du serveur gRPC : %v", err)
	}
}
