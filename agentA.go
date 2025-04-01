package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Structure de paquet Request pour la session
type SendSessionRequestPacket struct {
	SenderAddress [16]byte // Adresse de l'émetteur (Session-Sender)
	ReceiverPort  uint16   // Port d'écoute du Reflector (Serveur)
	SenderPort    uint16   // Port de l'émetteur
	PaddingLength uint32   // Taille du padding
	StartTime     uint32   // Heure de début (timestamp quand le test commence)
	Timeout       uint32   // Délai d'expiration de la session
	TypeP         uint8    // Type de service (DSCP)
}

type SessionAcceptPacket struct {
	Accept         uint8    // Code d'acceptation (0 = OK, autre = erreur)
	MBZ            uint8    // Champ réservé (Must Be Zero)
	Port           uint16   // Port attribué par le serveur
	ReflectedOctet [16]byte // Adresse IP renvoyée par le serveur
	ServerOctets   [16]byte // Adresse IP du serveur
	SID            uint32   // Session ID unique
	HMAC           [16]byte // Code d'authentification HMAC
}
type StartSessionPacket struct{
	MBZ   			uint8  
	HMAC 			[16]byte
}
type StartAckPacket struct{
	Accept 			   uint8
	MBZ 			   uint8
	HMAC			  [16]byte
}

type TwampTestPacket struct {
	SequenceNumber         uint32  // Numéro de séquence du paquet
	Timestamp             uint64  // Timestamp d'envoi (format NTP)
	ErrorEstimation       uint16  // Estimation d'erreur du timestamp
	MBZ                   uint16  // Must Be Zero (champ réservé, toujours 0)
	ReceptionTimestamp    uint64  // Timestamp de réception du paquet (format NTP)
	SenderSequenceNumber  uint32  // Numéro de séquence côté émetteur
	SenderTimestamp       uint64  // Timestamp d'envoi par l'émetteur (format NTP)
	SenderErrorEstimation uint16  // Estimation d'erreur côté émetteur
	SenderTTL             uint8   // Time-To-Live (TTL) du paquet
	Padding               []byte  // Padding optionnel pour ajuster la taille du paquet
}
 type StopSessionPacket struct {
	Accept 				 uint8
	MBZ 				 uint8
	NumberOfSessions	 uint8
	HMAC				[16]byte
}
 
// Fonction pour sérialiser un paquet
func SerializePacket(packet *SessionRequestPacket) ([]byte, error) {
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.BigEndian, packet.SenderAddress)
	if err != nil {
		return nil, err
	}
	// Sérialiser les autres champs
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

// Fonction pour envoyer un paquet à une adresse UDP
func sendRequestPacket(packet *SessionRequestPacket, serverAddress string, serverPort int) error {

	serializedPacket, err := serializePacket(packet)
	if err != nil {
		return fmt.Errorf("erreur de sérialisation du paquet : %v", err)
	}

	// Connexion UDP
	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", serverAddress, serverPort))
	if err != nil {
		return fmt.Errorf("erreur de connexion UDP : %v", err)
	}
	defer conn.Close()

	// Envoi du paquet sérialisé
	_, err = conn.Write(serializedPacket)
	if err != nil {
		return fmt.Errorf("erreur lors de l'envoi du paquet UDP : %v", err)
	}

	fmt.Println("Paquet envoyé avec succès !")
	return nil
}

// Fonction pour gérer l'émetteur
func handleSender() {
	// Préparer un paquet SessionRequest
	packet := SessionRequestPacket{
		SenderAddress: [16]byte{192, 168, 1, 1}, // Exemple d'adresse IP
		ReceiverPort:  5000,                     // Port du Reflector
		SenderPort:    6000,                     // Port de l'émetteur
		PaddingLength: 0,
		StartTime:     uint32(time.Now().Unix()),
		Timeout:       30,
		TypeP:         0x00, // Exemple de type de service
	}

	// 1. Envoyer le Session Request
	fmt.Println("Envoi du paquet Session Request...")
	err := sendSessionRequestPacket(&packet, "127.0.0.1", 5000)
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du paquet : %v", err)
	}

	// 2. Attendre Start-Ack (simuler ici)
	fmt.Println("Attente de Start-Ack...")
	time.Sleep(2 * time.Second) // Simuler l'attente

	// 3. Envoyer Start Session
	startSessionPacket := SessionRequestPacket{
		SenderAddress: packet.SenderAddress,
		ReceiverPort:  packet.ReceiverPort,
		SenderPort:    packet.SenderPort,
		PaddingLength: 0,
		StartTime:     uint32(time.Now().Unix()),
		Timeout:       30,
		TypeP:         0x01, // Type de service pour Start Session
	}
	fmt.Println("Envoi du paquet Start Session...")
	err = sendSessionRequestPacket(&startSessionPacket, "127.0.0.1", 5000)
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du Start Session : %v", err)
	}

	// 4. Attendre Stop-Ack (simuler ici)
	fmt.Println("Attente de Stop-Ack...")
	time.Sleep(2 * time.Second) // Simuler l'attente

	// 5. Envoyer Stop Session
	stopSessionPacket := SessionRequestPacket{
		SenderAddress: packet.SenderAddress,
		ReceiverPort:  packet.ReceiverPort,
		SenderPort:    packet.SenderPort,
		PaddingLength: 0,
		StartTime:     uint32(time.Now().Unix()),
		Timeout:       30,
		TypeP:         0x02, // Type de service pour Stop Session
	}
	fmt.Println("Envoi du paquet Stop Session...")
	err = sendSessionRequestPacket(&stopSessionPacket, "127.0.0.1", 5000)
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du Stop Session : %v", err)
	}
}

// Fonction pour gérer le Reflector
func handleReflector() {
	// Préparer un paquet Start-Ack à envoyer en réponse
	packet := SessionRequestPacket{
		SenderAddress: [16]byte{192, 168, 1, 1},
		ReceiverPort:  5000, // Port d'écoute du Reflector
		SenderPort:    6000,
	}

	// 1. Attendre Session Request (simuler ici)
	fmt.Println("Attente du paquet Session Request...")
	time.Sleep(2 * time.Second) // Simuler l'attente

	// 2. Répondre avec Start-Ack
	fmt.Println("Envoi du Start-Ack...")
	startAckPacket := SessionRequestPacket{
		SenderAddress: packet.SenderAddress,
		ReceiverPort:  packet.ReceiverPort,
		SenderPort:    packet.SenderPort,
		PaddingLength: 0,
		StartTime:     uint32(time.Now().Unix()),
		Timeout:       30,
		TypeP:         0x01, // Type de service pour Start-Ack
	}
	err := sendSessionRequestPacket(&startAckPacket, "127.0.0.1", 6000) // Envoyer à l'émetteur
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du Start-Ack : %v", err)
	}

	// 3. Attendre Start Session (simuler ici)
	fmt.Println("Attente du Start Session...")
	time.Sleep(2 * time.Second) // Simuler l'attente

	// 4. Répondre avec Stop-Ack
	fmt.Println("Envoi du Stop-Ack...")
	stopAckPacket := SessionRequestPacket{
		SenderAddress: packet.SenderAddress,
		ReceiverPort:  packet.ReceiverPort,
		SenderPort:    packet.SenderPort,
		PaddingLength: 0,
		StartTime:     uint32(time.Now().Unix()),
		Timeout:       30,
		TypeP:         0x02, // Type de service pour Stop-Ack
	}
	err = sendSessionRequestPacket(&stopAckPacket, "127.0.0.1", 6000) // Envoyer à l'émetteur
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du Stop-Ack : %v", err)
	}
}

func main() {
	// Déterminer le rôle de l'agent (ici, un simple flag pour l'exemple)
	isSender := true // Modifier cette valeur pour tester l'autre rôle (Reflector)

	if isSender {
		// Si l'agent est un Sender, gérer en conséquence
		handleSender()
	} else {
		// Si l'agent est un Reflector, gérer en conséquence
		handleReflector()
	}

	// Connexion au serveur gRPC
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Erreur lors de la connexion au serveur gRPC : %v", err)
	}
	defer conn.Close()

	// Utilisation de la connexion gRPC ici...
}
