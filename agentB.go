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
type StartSessionPacket struct {
	MBZ  uint8
	HMAC [16]byte
}
type StartAckPacket struct {
	Accept uint8
	MBZ    uint8
	HMAC   [16]byte
}

type TwampTestPacket struct {
	SequenceNumber        uint32 // Numéro de séquence du paquet
	Timestamp             uint64 // Timestamp d'envoi (format NTP)
	ErrorEstimation       uint16 // Estimation d'erreur du timestamp
	MBZ                   uint16 // Must Be Zero (champ réservé, toujours 0)
	ReceptionTimestamp    uint64 // Timestamp de réception du paquet (format NTP)
	SenderSequenceNumber  uint32 // Numéro de séquence côté émetteur
	SenderTimestamp       uint64 // Timestamp d'envoi par l'émetteur (format NTP)
	SenderErrorEstimation uint16 // Estimation d'erreur côté émetteur
	SenderTTL             uint8  // Time-To-Live (TTL) du paquet
	Padding               []byte // Padding optionnel pour ajuster la taille du paquet
}
type StopSessionPacket struct {
	Accept           uint8
	MBZ              uint8
	NumberOfSessions uint8
	HMAC             [16]byte
}

// Fonction pour sérialiser un paquet
func SerializePacket(packet *SendSessionRequestPacket) ([]byte, error) {
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

// fonction pour la sérialition accept session
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

// fonction pour la sérialition start session
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

// fonction pour la sérialition start session ACK
func SerializeStartACKtPacket(packet *StartAckPacket) ([]byte, error) {
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

// fonction pour la sérialition accept session
func SerializeTwampTestPacket(packet *TwampTestPacket) ([]byte, error) {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		packet.SequenceNumber,
		packet.Timestamp,
		packet.ErrorEstimation,
		packet.MBZ,
		packet.ReceptionTimestamp,
		packet.SenderSequenceNumber,
		packet.SenderTimestamp,
		packet.SenderErrorEstimation,
		packet.SenderTTL,
		packet.Padding,
	}
	for _, field := range fields {
		err := binary.Write(buf, binary.BigEndian, field)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// fonction pour la sérialition Stop session
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

// SendPacket envoie un paquet via UDP vers une adresse et un port donnés
func SendPacket(packet []byte, addr string, port int) error {
	// Résoudre l'adresse IP et le port de destination
	remoteAddr := &net.UDPAddr{
		IP:   net.ParseIP(addr),
		Port: port,
	}
	// Créer une connexion UDP
	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Envoyer le paquet
	_, err = conn.Write(packet)
	if err != nil {
		return err
	}
	return nil
}

// receivePacket - Lit un paquet UDP
func receivePacket() ([]byte, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 5000})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	buffer := make([]byte, 1500)
	n, _, err := conn.ReadFromUDP(buffer)
	return buffer[:n], err
}

// deserializeTwampTestPacket - Désérialisation TWAMP
func deserializeTwampTestPacket(data []byte, pkt *TwampTestPacket) error {
	if len(data) < 42 { // Taille minimale TWAMP
		return fmt.Errorf("paquet trop court")
	}

	buf := bytes.NewReader(data)
	return binary.Read(buf, binary.BigEndian, pkt)
}

// serializeTwampTestPacket - Sérialisation TWAMP
func serializeTwampTestPacket(pkt *TwampTestPacket) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, pkt)
	return buf.Bytes(), err
}
func handleSender() {
	// Préparer un paquet SessionRequest
	packet := SendSessionRequestPacket{
		SenderAddress: [16]byte{192, 168, 1, 1}, // Exemple d'adresse IP
		ReceiverPort:  5000,                     // Port du Reflector
		SenderPort:    6000,                     // Port de l'émetteur
		PaddingLength: 0,
		StartTime:     uint32(time.Now().Unix()),
		Timeout:       30,
		TypeP:         0x00, // Exemple de type de service
	}

	// 1. Envoyer le paquet Session Request
	fmt.Println("Envoi du paquet Session Request...")
	serializedPacket, err := SerializePacket(&packet)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Session Request : %v", err)
	}
	err = SendPacket(serializedPacket, "127.0.0.1", 5000)

	// 2. Attendre Accept-session (simuler ici)
	fmt.Println("Attente de Accept-session...")
	time.Sleep(2 * time.Second) // Simuler l'attente

	// 3. Préparer le paquet Start Session
	startSessionPacket := StartSessionPacket{
		MBZ:  0,
		HMAC: [16]byte{},
	}
	fmt.Println("Envoi du paquet Start Session...")
	serializedStartSessionPacket, err := SerializeStartPacket(&startSessionPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Start Session : %v", err)
	}
	err = SendPacket(serializedStartSessionPacket, "127.0.0.1", 5000)

	// 4. Attendre Start-Ack (simuler ici)
	fmt.Println("Attente de Start-Ack...")
	time.Sleep(2 * time.Second) // Simuler l'attente

	//5.Preparer le paquet twamp-test
	twamp_testpaquet := TwampTestPacket{
		SequenceNumber:        0,
		Timestamp:             uint64(time.Now().UnixNano()),
		ErrorEstimation:       0,
		MBZ:                   0,
		ReceptionTimestamp:    0,
		SenderSequenceNumber:  5,
		SenderTimestamp:       uint64(time.Now().UnixNano()),
		SenderErrorEstimation: 0,
		SenderTTL:             255,
		Padding:               make([]byte, 20),
	}
	fmt.Println("Envoi de paquet twaamp-test...")
	serializeTwampTestPacket, err := SerializeTwampTestPacket(&twamp_testpaquet)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Start Session : %v", err)
	}
	err = SendPacket(serializeTwampTestPacket, "127.0.0.1", 5000)
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du paquet TWAMP-Test : %v", err)
	}

	//6.Attendre twamp-test reflecter
	fmt.Println("Attente de reflaction du paquet twamp-test...")
	time.Sleep(2 * time.Second)

	// 7. Préparer le paquet Stop Session
	stopSessionPacket := StopSessionPacket{
		Accept:           0,          // Par exemple, 0 pour indiquer un refus ou une valeur spécifique
		MBZ:              0,          // Champ réservé (initialisé à 0)
		NumberOfSessions: 1,          // Par exemple, 1 pour un nombre de sessions
		HMAC:             [16]byte{}, // Initialisation du champ HMAC (avec un code ou une valeur de hachage)
	}
	fmt.Println("Envoi du paquet Stop Session...")
	serializedStopSessionPacket, err := SerializeStopSession(&stopSessionPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du Stop-Ack : %v", err)
	}
	err = SendPacket(serializedStopSessionPacket, "127.0.0.1", 5000) // Utilisation de la fonction d'envoi correcte
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du Stop Session : %v", err)
	}
}

// Fonction pour gérer le Reflector
func handleReflector() {

	// 1. Attendre Session Request (simuler ici)
	fmt.Println("Attente du paquet Session Request...")
	time.Sleep(2 * time.Second)

	// 2. Répondre avec Accept-Session
	fmt.Println("Envoi du Accept-Session ...")
	acceptSessionPacket := SessionAcceptPacket{
		Accept: 0, // Exemple : acceptation de la session
		MBZ:    0, // Champ réservé, toujours égal à 0
		HMAC:   [16]byte{},
	}
	serializedPacket, err := SerializeAcceptPacket(&acceptSessionPacket)
	err = SendPacket(serializedPacket, "127.0.0.1", 6000)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du Start-Session : %v", err)
	}

	// 3. Attendre Start Session (simuler ici)
	fmt.Println("Attente du Start Session...")
	time.Sleep(2 * time.Second)

	// 4. Répondre avec Start-Ack
	fmt.Println("Envoi du Start-Ack...")
	startAckPacket := StartAckPacket{
		Accept: 0,
		MBZ:    0,
		HMAC:   [16]byte{},
	}
	serializeStartACKtPacket, err := SerializeStartACKtPacket(&startAckPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Start-Ack  : %v", err)
	}
	err = SendPacket(serializeStartACKtPacket, "127.0.0.1", 5000)

	//5.Attendre twamp-test
	fmt.Println("Attente  du paquet twamp-test...")
	time.Sleep(2 * time.Second)

	//6.Rependre avec reflacted twamp-test
	receivedData, err := receivePacket() // reçoit le paquet brut
	if err != nil {
		log.Fatalf("Erreur de réception: %v", err)
	}

	// Désérialiser le paquet brut en une structure
	var receivedPacket TwampTestPacket
	err = deserializeTwampTestPacket(receivedData, &receivedPacket)
	if err != nil {
		log.Fatalf("Erreur de désérialisation du paquet reçu : %v", err)
	}

	// Mettre à jour le champ ReceptionTimestamp
	receivedPacket.ReceptionTimestamp = uint64(time.Now().UnixNano())

	// Sérialiser le paquet réfléchi avec le champ mis à jour
	fmt.Println("Réflexion du paquet TWAMP-Test...")
	serializedtestPacket, err := SerializeTwampTestPacket(&receivedPacket)

	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet réfléchi : %v", err)
	}
	//Renvoi du paquet réfléchi au sender (l'IP et le port sont ceux du sender)
	err = SendPacket(serializedtestPacket, "127.0.0.1", 6000) // Exemple d'adresse et de port
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du paquet réfléchi : %v", err)
	}
	//7.Attendre  paquet Stop-session
	fmt.Println("Attente  du paquet twamp-test...")
	time.Sleep(2 * time.Second)

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
		log.Fatalf("Impossible de se connecter : %v", err)
	}
	defer conn.Close()

	client := testpb.NewTestServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Demande d'un test QoS au serveur (ajustez en fonction de votre .proto)
	req := &testpb.QoSTestRequest{
		TestId:         "Latence",                      // Assurez-vous d'utiliser la bonne casse
		TestParameters: "10 paquets ICMP vers 8.8.8.8", // Assurez-vous que ces champs existent dans votre .proto
	}

	// Appel RunQoSTest
	res, err := client.RunQoSTest(ctx, req)
	if err != nil {
		log.Fatalf("Erreur lors de la demande du test QoS : %v", err)
	}

	// Utilisation des champs corrects dans la réponse : Status et Result
	fmt.Println("Test QoS reçu du serveur :", res.Status, res.Result)

	// Simuler un test QoS
	time.Sleep(2 * time.Second)
	testResult := fmt.Sprintf("Test ID: %s - Latence mesurée: 50ms", res.Status)

	// Connexion WebSocket au serveur
	ws, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatalf("Erreur de connexion WebSocket : %v", err)
	}
	defer ws.Close()

	// Envoi du résultat via WebSocket
	err = ws.WriteMessage(websocket.TextMessage, []byte(testResult))
	if err != nil {
		log.Fatalf("Erreur d'envoi WebSocket : %v", err)
	}

	fmt.Println("Résultat envoyé au serveur via WebSocket")
	defer func() {
		ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Fin de communication"))
		ws.Close()
	}()

}
