package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"mon-projet-go/testpb"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
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
func SendPacket_(packet []byte, addr string, port int) error {
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

// Lit un paquet UDP
func receivePacket_() ([]byte, error) {
    listener, err := net.Listen("tcp", ":5000")
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
	if err := SendPacket_(serializedPacket, "127.0.0.1", 5000)
	err != nil {
		log.Fatalf("Erreur envoi Session-Request : %v", err)
	}

	// 2. Réception du paquet Accept-session
	receivedData, err := receivePacket_()
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
	if err = SendPacket(serializedStartSessionPacket, "127.0.0.1", 5000); err != nil {
		log.Fatalf("Erreur lors de l'envoi du paquet Start Session : %v", err)
	}
	// 4. Réception du paquet Start-ACK et validation
	receivedData, err = receivePacket()
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
	if err = SendPacket(serializedStopSessionPacket, "127.0.0.1", 5000); err != nil {
		log.Fatalf("Erreur lors de l'envoi du Stop Session : %v", err)
	}
}

func Server() {
	// Démarrer un serveur TCP sur le port 5000
	listener, err := net.Listen("tcp", ":5000")
	if err != nil {
		log.Fatalf("Erreur de serveur TCP : %v", err)
	}
	defer listener.Close()

	log.Println("Serveur en attente de connexions sur le port 5000...")

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

// Méthode gRPC pour envoyer un test QoS aux agents
func (s *server) RunQoSTest(ctx context.Context, req *testpb.QoSTestRequest) (*testpb.QoSTestResponse, error) {
	log.Printf("Envoi du test QoS : %s avec config: %s", req.TestId, req.TestParameters)
	// Retourner une réponse avec des résultats fictifs
	return &testpb.QoSTestResponse{
		Status: "Réussi",       // Statut du test
		Result: "Latence 10ms", // Résultats du test
	}, nil
}

// WebSocket : gestion des messages entrants
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // Permet d'accepter les connexions cross-origin
}

// Serveur WebSocket pour recevoir les résultats
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Erreur WebSocket:", err)
		return
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Erreur de lecture WebSocket:", err)
			break
		}
		// Afficher le message reçu de l'agent
		fmt.Println("Résultat reçu de l'agent :", string(msg))
		// Envoyer une réponse au client WebSocket
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Message reçu avec succès")); err != nil {
			log.Println("Erreur lors de l'envoi de message WebSocket:", err)
			break
		}
	}
}

func main() {
	// Démarrage du serveur gRPC
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Échec de l'écoute : %v", err)
	}

	grpcServer := grpc.NewServer()
	testpb.RegisterTestServiceServer(grpcServer, &server{})

	// Lancer le serveur WebSocket en parallèle
	go func() {
		http.HandleFunc("/ws", handleWebSocket)
		log.Println("Serveur WebSocket sur le port 8080...")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
	go Server()
	go client()

	log.Println("Serveur gRPC démarré sur le port 50051...")

	// Démarrer le serveur gRPC
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Échec du démarrage du serveur gRPC : %v", err)
	}
}
