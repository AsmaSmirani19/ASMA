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

// Structure des paquets:
type SendSessionRequestPacket_B struct {
	SenderAddress [16]byte
	ReceiverPort  uint16
	SenderPort    uint16
	PaddingLength uint32
	StartTime     uint32
	Timeout       uint32
	TypeP         uint8
}

type SessionAcceptPacket_B struct {
	Accept         uint8
	MBZ            uint8
	Port           uint16
	ReflectedOctet [16]byte
	ServerOctets   [16]byte
	SID            uint32
	HMAC           [16]byte
}
type StartSessionPacket_B struct {
	MBZ  uint8
	HMAC [16]byte
}
type StartAckPacket_B struct {
	Accept uint8
	MBZ    uint8
	HMAC   [16]byte
}

type TwampTestPacket_B struct {
	SequenceNumber        uint32
	Timestamp             uint64
	ErrorEstimation       uint16
	MBZ                   uint16
	ReceptionTimestamp    uint64
	SenderSequenceNumber  uint32
	SenderTimestamp       uint64
	SenderErrorEstimation uint16
	SenderTTL             uint8
	Padding               []byte
}
type StopSessionPacket_B struct {
	Accept           uint8
	MBZ              uint8
	NumberOfSessions uint8
	HMAC             [16]byte
}

// Fonction pour sérialiser un paquet
func SerializePacket_B(packet *SendSessionRequestPacket_B) ([]byte, error) {
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
func SerializeAcceptPacket_B(packet *SessionAcceptPacket_B) ([]byte, error) {
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
func SerializeStartPacket_B(packet *StartSessionPacket_B) ([]byte, error) {
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
func SerializeStartACKtPacket_B(packet *StartAckPacket_B) ([]byte, error) {
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
func SerializeTwampTestPacket_B(packet *TwampTestPacket_B) ([]byte, error) {
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
func SerializeStopSession_B(packet *StopSessionPacket_B) ([]byte, error) {
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
func SendPacket_B(packet []byte, addr string, port int) error {
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
func receivePacket_B() ([]byte, error) {
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
func deserializeTwampTestPacket_B(data []byte, pkt *TwampTestPacket_B) error {
	if len(data) < 42 { // Taille minimale TWAMP
		return fmt.Errorf("paquet trop court")
	}

	buf := bytes.NewReader(data)
	return binary.Read(buf, binary.BigEndian, pkt)
}

func handleSender_B() {
	// Préparer un paquet SessionRequest
	packet := SendSessionRequestPacket_B{
		SenderAddress: [16]byte{192, 168, 1, 1},
		ReceiverPort:  5000,
		SenderPort:    6000,
		PaddingLength: 0,
		StartTime:     uint32(time.Now().Unix()),
		Timeout:       30,
		TypeP:         0x00,
	}

	// 1. Envoyer le paquet Session Request
	fmt.Println("Envoi du paquet Session Request...")
	serializedPacket, err := SerializePacket_B(&packet)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Session Request : %v", err)
	}
	err = SendPacket_B(serializedPacket, "127.0.0.1", 5000)

	// 2. Attendre Accept-session (simuler ici)
	fmt.Println("Attente de Accept-session...")
	time.Sleep(2 * time.Second) // Simuler l'attente

	// 3. Préparer le paquet Start Session
	startSessionPacket := StartSessionPacket_B{
		MBZ:  0,
		HMAC: [16]byte{},
	}
	fmt.Println("Envoi du paquet Start Session...")
	serializedStartSessionPacket, err := SerializeStartPacket_B(&startSessionPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Start Session : %v", err)
	}
	err = SendPacket_B(serializedStartSessionPacket, "127.0.0.1", 5000)

	// 4. Attendre Start-Ack (simuler ici)
	fmt.Println("Attente de Start-Ack...")
	time.Sleep(2 * time.Second) // Simuler l'attente

	//5.Preparer le paquet twamp-test
	twamp_testpaquet := TwampTestPacket_B{
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
	serializeTwampTestPacket, err := SerializeTwampTestPacket_B(&twamp_testpaquet)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Start Session : %v", err)
	}
	err = SendPacket_B(serializeTwampTestPacket, "127.0.0.1", 5000)
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du paquet TWAMP-Test : %v", err)
	}

	//6.Attendre twamp-test reflecter
	fmt.Println("Attente de reflaction du paquet twamp-test...")
	time.Sleep(2 * time.Second)

	// 7. Préparer le paquet Stop Session
	stopSessionPacket := StopSessionPacket_B{
		Accept:           0,
		MBZ:              0,
		NumberOfSessions: 1,
		HMAC:             [16]byte{},
	}
	fmt.Println("Envoi du paquet Stop Session...")
	serializedStopSessionPacket, err := SerializeStopSession_B(&stopSessionPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du Stop-Ack : %v", err)
	}
	err = SendPacket_B(serializedStopSessionPacket, "127.0.0.1", 5000)
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du Stop Session : %v", err)
	}
}

// Fonction pour gérer le Reflector
func handleReflector_B() {

	// 1. Attendre Session Request (simuler ici)
	fmt.Println("Attente du paquet Session Request...")
	time.Sleep(2 * time.Second)

	// 2. Répondre avec Accept-Session
	fmt.Println("Envoi du Accept-Session ...")
	acceptSessionPacket := SessionAcceptPacket_B{
		Accept: 0,
		MBZ:    0,
		HMAC:   [16]byte{},
	}
	serializedPacket, err := SerializeAcceptPacket_B(&acceptSessionPacket)
	err = SendPacket_B(serializedPacket, "127.0.0.1", 6000)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du Start-Session : %v", err)
	}

	// 3. Attendre Start Session (simuler ici)
	fmt.Println("Attente du Start Session...")
	time.Sleep(2 * time.Second)

	// 4. Répondre avec Start-Ack
	fmt.Println("Envoi du Start-Ack...")
	startAckPacket := StartAckPacket_B{
		Accept: 0,
		MBZ:    0,
		HMAC:   [16]byte{},
	}
	serializeStartACKtPacket, err := SerializeStartACKtPacket_B(&startAckPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Start-Ack  : %v", err)
	}
	err = SendPacket_B(serializeStartACKtPacket, "127.0.0.1", 5000)

	//5.Attendre twamp-test
	fmt.Println("Attente  du paquet twamp-test...")
	time.Sleep(2 * time.Second)

	//6.Rependre avec reflacted twamp-test
	receivedData, err := receivePacket_B() // reçoit le paquet brut
	if err != nil {
		log.Fatalf("Erreur de réception: %v", err)
	}

	// Désérialiser le paquet brut en une structure
	var receivedPacket TwampTestPacket_B
	err = deserializeTwampTestPacket_B(receivedData, &receivedPacket)
	if err != nil {
		log.Fatalf("Erreur de désérialisation du paquet reçu : %v", err)
	}

	// Mettre à jour le champ ReceptionTimestamp
	receivedPacket.ReceptionTimestamp = uint64(time.Now().UnixNano())

	// Sérialiser le paquet réfléchi avec le champ mis à jour
	fmt.Println("Réflexion du paquet TWAMP-Test...")
	serializedtestPacket, err := SerializeTwampTestPacket_B(&receivedPacket)

	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet réfléchi : %v", err)
	}
	//Renvoi du paquet réfléchi au sender (l'IP et le port sont ceux du sender)
	err = SendPacket_B(serializedtestPacket, "127.0.0.1", 6000) // Exemple d'adresse et de port
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du paquet réfléchi : %v", err)
	}
	//7.Attendre  paquet Stop-session
	fmt.Println("Attente  du paquet twamp-test...")
	time.Sleep(2 * time.Second)

}

func main() {

	go handleSender_B()
	go handleReflector_B()

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
