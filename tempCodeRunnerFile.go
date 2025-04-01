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

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Structure de paquet Request pour la session
type SessionRequestPacket struct {
	SenderAddress [16]byte // Adresse de l'émetteur (Session-Sender)
	ReceiverPort  uint16   // Port d'écoute du Reflector (Serveur)
	SenderPort    uint16   // Port de l'émetteur
	PaddingLength uint32   // Taille du padding
	StartTime     uint32   // Heure de début (timestamp quand le test commence)
	Timeout       uint32   // Délai d'expiration de la session
	TypeP         uint8    // Type de service (DSCP)
}

// Fonction pour sérialiser un paquet
func serializePacket(packet *SessionRequestPacket) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Liste des champs à sérialiser
	fields := []interface{}{
		packet.SenderAddress,
		packet.ReceiverPort,
		packet.SenderPort,
		packet.PaddingLength,
		packet.StartTime,
		packet.Timeout,
		packet.TypeP,
	}

	// Sérialisation des champs dans le buffer via une boucle
	for _, field := range fields {
		err := binary.Write(buf, binary.BigEndian, field)
		if err != nil {
			return nil, err
		}
	}

	// Retourner le buffer sérialisé
	return buf.Bytes(), nil
}

// Fonction pour envoyer un paquet à une adresse UDP
func sendSessionRequestPacket(packet *SessionRequestPacket, serverAddress string, serverPort int) error {
	// Sérialiser le paquet
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

	// 3. Envoyer Start Session (paquet manquant)
	// TODO: Implémenter l'envoi du paquet Start Session

	// 4. Attendre Stop-Ack (simuler ici)
	fmt.Println("Attente de Stop-Ack...")

	// 5. Envoyer Stop Session (paquet manquant)
	// TODO: Implémenter l'envoi du paquet Stop Session
}

func handleReflector() {
	// Préparer un paquet Start-Ack à envoyer en réponse
	packet := SessionRequestPacket{
		SenderAddress: [16]byte{192, 168, 1, 1},
		ReceiverPort:  5000, // Port d'écoute du Reflector
		SenderPort:    6000,
	}

	// 1. Attendre Session Request (simuler ici)
	fmt.Println("Attente du paquet Session Request...")

	// 2. Répondre avec Start-Ack (paquet manquant)
	// TODO: Implémenter l'envoi du paquet Start-Ack

	// 3. Attendre Start Session (simuler ici)
	fmt.Println("Attente du Start Session...")

	// 4. Répondre avec Stop-Ack (paquet manquant)
	// TODO: Implémenter l'envoi du paquet Stop-Ack
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

	// Code pour envoyer un paquet et effectuer un test QoS
	// Création d'un paquet SessionRequest
	packet := &SessionRequestPacket{
		SenderAddress: [16]byte{192, 168, 1, 1},  // Exemple d'adresse IP
		ReceiverPort:  8000,                      // Port du Reflector
		SenderPort:    9000,                      // Port de l'émetteur
		PaddingLength: 0,                         // Pas de padding pour l'exemple
		StartTime:     uint32(time.Now().Unix()), // Timestamp actuel
		Timeout:       30,                        // Délai d'expiration de 30 secondes
		TypeP:         0x00,                      // Type de service (exemple)
	}

	// Envoi du paquet via UDP
	err := sendSessionRequestPacket(packet, "127.0.0.1", 5000)
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du paquet : %v", err)
	}

	// Connexion au serveur gRPC
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Impossible de se connecter : %v", err)
	}
	defer conn.Close()

	client := testpb.NewTestServiceClient(conn)

	// Appel au serveur gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &testpb.QoSTestRequest{
		TestId:         "Latence",
		TestParameters: "10 paquets ICMP vers 8.8.8.8",
	}

	res, err := client.RunQoSTest(ctx, req)
	if err != nil {
		log.Fatalf("Erreur lors de la demande du test QoS : %v", err)
	}

	// Affichage des résultats du test QoS
	fmt.Println("Test QoS reçu du serveur :", res.Status, res.Result)

	// Connexion WebSocket au serveur
	ws, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatalf("Erreur de connexion WebSocket : %v", err)
	}
	defer ws.Close()

	// Envoi du résultat via WebSocket
	testResult := fmt.Sprintf("Test ID: %s - Latence mesurée: 50ms", res.Status)
	err = ws.WriteMessage(websocket.TextMessage, []byte(testResult))
	if err != nil {
		log.Fatalf("Erreur d'envoi WebSocket : %v", err)
	}

	fmt.Println("Résultat envoyé au serveur via WebSocket")

	// Fermeture propre de la connexion WebSocket
	defer func() {
		ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Fin de communication"))
		ws.Close()
	}()
}