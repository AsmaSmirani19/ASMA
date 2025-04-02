package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"
)

// Structure des paquets:
type SendSessionRequestPacket_A struct {
	SenderAddress [16]byte
	ReceiverPort  uint16
	SenderPort    uint16
	PaddingLength uint32
	StartTime     uint32
	Timeout       uint32
	TypeP         uint8
}

type SessionAcceptPacket_A struct {
	Accept         uint8
	MBZ            uint8
	Port           uint16
	ReflectedOctet [16]byte
	ServerOctets   [16]byte
	SID            uint32
	HMAC           [16]byte
}
type StartSessionPacket_A struct {
	MBZ  uint8
	HMAC [16]byte
}
type StartAckPacket_A struct {
	Accept uint8
	MBZ    uint8
	HMAC   [16]byte
}

type TwampTestPacket_A struct {
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
type StopSessionPacket_A struct {
	MBZ              uint8
	NumberOfSessions uint8
	HMAC             [16]byte
}

// Fonction pour sérialiser un paquet
func SerializePacket_A(packet *SendSessionRequestPacket_A) ([]byte, error) {
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

// fonction pour la sérialisation accept session
func SerializeAcceptPacket_A(packet *SessionAcceptPacket_A) ([]byte, error) {
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

// fonction pour la sérialisation start session
func SerializeStartPacket_A(packet *StartSessionPacket_A) ([]byte, error) {
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

// fonction pour la sérialisation start-session ACK
func SerializeStartACKtPacket_A(packet *StartAckPacket_A) ([]byte, error) {
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

// fonction pour la sérialisation accept session
func SerializeTwampTestPacket_A(packet *TwampTestPacket_A) ([]byte, error) {
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

// fonction pour la sérialisation Stop session
func SerializeStopSession_A(packet *StopSessionPacket_A) ([]byte, error) {
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

// SendPacket envoie un paquet via UDP
func SendPacket_A(packet []byte, addr string, port int) error {
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

// Fonction pour recevoir des paquets
func receivePacket_A(port int) ([]byte, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: port})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	buffer := make([]byte, 1500)
	n, _, err := conn.ReadFromUDP(buffer)
	return buffer[:n], err
}

// Fonction pour gérer le Sender
func handleSender_A() {
	// Préparer un paquet SessionRequest
	packet := SendSessionRequestPacket_A{
		SenderAddress: [16]byte{192, 168, 1, 1},
		ReceiverPort:  5000,
		SenderPort:    6000,
		PaddingLength: 0,
		StartTime:     uint32(time.Now().Unix()),
		Timeout:       30,
		TypeP:         0x00, // Exemple de type de service
	}

	// 1. Envoyer le paquet Session Request
	fmt.Println("Envoi du paquet Session Request...")
	serializedPacket, err := SerializePacket_A(&packet)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Session Request : %v", err)
	}
	err = SendPacket_A(serializedPacket, "127.0.0.1", 5000)
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du paquet Session Request : %v", err)
	}

	// 2. Attendre Accept-session (simuler ici)
	fmt.Println("Attente de Accept-session...")
	time.Sleep(2 * time.Second)

	// 3. Préparer le paquet Start Session
	startSessionPacket := StartSessionPacket_A{
		MBZ:  0,
		HMAC: [16]byte{},
	}
	fmt.Println("Envoi du paquet Start Session...")
	serializedStartSessionPacket, err := SerializeStartPacket_A(&startSessionPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Start Session : %v", err)
	}
	err = SendPacket_A(serializedStartSessionPacket, "127.0.0.1", 5000)

	// 4. Attendre Start-Ack (simuler ici)
	fmt.Println("Attente de Start-Ack...")
	time.Sleep(2 * time.Second)

	// 5. Préparer le paquet TWAMP Test
	twamp_testpaquet := TwampTestPacket_A{
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
	fmt.Println("Envoi de paquet TWAMP-Test...")
	serializeTwampTestPacket, err := SerializeTwampTestPacket_A(&twamp_testpaquet)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet TWAMP-Test : %v", err)
	}
	err = SendPacket_A(serializeTwampTestPacket, "127.0.0.1", 5000)
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du paquet TWAMP-Test : %v", err)
	}

	// 6. Attendre TWAMP-Test Reflector
	fmt.Println("Attente de réflexion du paquet TWAMP-Test...")
	time.Sleep(2 * time.Second)

	// 7. Préparer le paquet Stop Session
	stopSessionPacket := StopSessionPacket_A{
		MBZ:              0,
		NumberOfSessions: 1,
		HMAC:             [16]byte{},
	}
	fmt.Println("Envoi du paquet Stop Session...")
	serializedStopSessionPacket, err := SerializeStopSession_A(&stopSessionPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Stop Session : %v", err)
	}
	err = SendPacket_A(serializedStopSessionPacket, "127.0.0.1", 5000)
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du paquet Stop Session : %v", err)
	}
}

// Fonction pour gérer le Reflector
func handleReflector_A() {
	// 1. Attendre Session Request (simuler ici)
	fmt.Println("Attente du paquet Session Request...")
	time.Sleep(2 * time.Second)

	// 2. Répondre avec Accept-Session
	acceptSessionPacket := SessionAcceptPacket_A{
		Accept:         0x01,
		MBZ:            0,
		Port:           5000,
		ReflectedOctet: [16]byte{},
		ServerOctets:   [16]byte{},
		SID:            12345,
		HMAC:           [16]byte{},
	}
	fmt.Println("Réponse au paquet Session Request avec Accept-Session...")
	serializedAcceptSessionPacket, err := SerializeAcceptPacket_A(&acceptSessionPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Accept-Session : %v", err)
	}
	err = SendPacket_A(serializedAcceptSessionPacket, "127.0.0.1", 6000)

	// 3. Attendre Start Session
	fmt.Println("Attente du paquet Start Session...")
	time.Sleep(2 * time.Second)

	// 4. Répondre avec Start Ack
	startAckPacket := StartAckPacket_A{
		Accept: 0x01,
		MBZ:    0,
		HMAC:   [16]byte{},
	}
	fmt.Println("Réponse au paquet Start Session avec Start-Ack...")
	serializedStartAckPacket, err := SerializeStartACKtPacket_A(&startAckPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet Start-Ack : %v", err)
	}
	err = SendPacket_A(serializedStartAckPacket, "127.0.0.1", 6000)

	// 5. Attente des paquets TWAMP
	fmt.Println("Attente du paquet TWAMP-Test...")
	// Simulation de réception TWAMP-Test
	time.Sleep(2 * time.Second)

	// 6. Répondre avec TWAMP-Test Reflector
	twampTestResponsePacket := TwampTestPacket_A{
		SequenceNumber:        5,
		Timestamp:             uint64(time.Now().UnixNano()),
		ErrorEstimation:       0,
		MBZ:                   0,
		ReceptionTimestamp:    uint64(time.Now().UnixNano()),
		SenderSequenceNumber:  5,
		SenderTimestamp:       uint64(time.Now().UnixNano()),
		SenderErrorEstimation: 0,
		SenderTTL:             255,
		Padding:               make([]byte, 20),
	}
	fmt.Println("Réponse avec TWAMP-Test Reflector...")
	serializeTwampTestResponsePacket, err := SerializeTwampTestPacket_A(&twampTestResponsePacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet TWAMP-Test Reflector : %v", err)
	}
	err = SendPacket_A(serializeTwampTestResponsePacket, "127.0.0.1", 6000)
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi du paquet TWAMP-Test Reflector : %v", err)
	}
}

func main() {
	// Choisir d'exécuter le Sender ou Reflector
	go handleSender_A()
	go handleReflector_A()

	// Attendre que tout se termine
	time.Sleep(30 * time.Second)
}
