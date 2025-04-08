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


type Metrics struct {
	SentPackets        int
	ReceivedPackets    int
	TotalBytesSent     int64
	TotalBytesReceived int64
	LastLatency        int64
	TotalJitter        int64
	StartTime          time.Time
	LatencySamples     []int64
}

// Structure des paquets TWAMP-Test
type TwampTestPacket struct {
	SequenceNumber        uint32
	Timestamp             uint64
	ErrorEstimation       uint16
	MBZ                   uint16
	ReceptionTimestamp    uint64
	SenderSequenceNumber  uint64
	SenderTimestamp       uint64
	SenderErrorEstimation uint16
	SenderTTL             uint8
	Padding               []byte
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
// deserializeTwampTestPacket 
func deserializeTwampTestPacket(data []byte, pkt *TwampTestPacket) error {
	if len(data) < 42 { // Taille minimale TWAMP
		return fmt.Errorf("paquet trop court")
	}
	buf := bytes.NewReader(data)
	return binary.Read(buf, binary.BigEndian, pkt)
}

// SendPacket envoie un paquet via UDP vers une adresse et un port donnés
func SendPacket(packet []byte, addr string, port int) error {

	remoteAddr := &net.UDPAddr{
		IP:   net.ParseIP(addr),
		Port: port,
	}
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

// Lit un paquet UDP
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

func handleSender(metrics *Metrics) {
	var totalLatency int64
	var totalJitter int64
	var previousLatency int64

	//Preparer le paquet twamp-test
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

	// 7. Réception du paquet TWAMP-Test réfléchi et calcul de la latence
	receivedData, err := receivePacket() 
	if err != nil {
		log.Fatalf("Erreur de réception: %v", err)
	}

	// Désérialiser le paquet reçu
	var receivedPacket TwampTestPacket
	err = deserializeTwampTestPacket(receivedData, &receivedPacket)
	if err != nil {
		log.Fatalf("Erreur de désérialisation du paquet reçu : %v", err)
	}
	

	// Mettre à jour le timestamp de réception
	receivedPacket.ReceptionTimestamp = uint64(time.Now().UnixNano())

	// Calcul de la latence (différence entre le timestamp d'envoi et de réception)
	latency := int64(receivedPacket.ReceptionTimestamp - (receivedPacket.SenderTimestamp))
	totalLatency += latency
	metrics.LatencySamples = append(metrics.LatencySamples, latency)
	metrics.LastLatency = latency
	metrics.ReceivedPackets++

	// Calcul du jitter
	if len(metrics.LatencySamples) > 1 {
		// Calcul de la différence avec le paquet précédent
		jitter := int64(latency - previousLatency)
		totalJitter += jitter
	}
	previousLatency = latency

	// Calcul de la bande passante
	metrics.SentPackets++
	metrics.TotalBytesReceived += 1024
	metrics.TotalBytesSent += 1024

	// Calcul de la perte de paquets
	packetLoss := float64(metrics.SentPackets-metrics.ReceivedPackets) / float64(metrics.SentPackets) * 100

	// Affichage des résultats
	fmt.Printf("Latence: %d ms\n", latency/1000000)    // Convertir en millisecondes
	fmt.Printf("Jitter: %d ms\n", totalJitter/1000000) // Convertir en millisecondes
	fmt.Printf("Bande passante: %d octets\n", metrics.TotalBytesReceived)
	fmt.Printf("Perte de paquets: %.2f%%\n", packetLoss)
}	

func handleTwampTestReflector(data []byte) error {
	var receivedPacket TwampTestPacket
	err := deserializeTwampTestPacket(data, &receivedPacket)
	if err != nil {
		return fmt.Errorf("erreur de désérialisation du paquet reçu : %v", err)
	}
	receivedPacket.ReceptionTimestamp = uint64(time.Now().UnixNano())
	serializedtestPacket, err := SerializeTwampTestPacket(&receivedPacket)
	if err != nil {
		return fmt.Errorf("erreur de sérialisation du paquet réfléchi : %v", err)
	}
	//Renvoi du paquet réfléchi au sender
	err = SendPacket(serializedtestPacket, "127.0.0.1", 5000)
	if err != nil {
		return fmt.Errorf("erreur lors de l'envoi du paquet réfléchi : %v", err)
	}
	log.Println("Paquet TWAMP-Test réfléchi et renvoyé au sender")
	return nil
}

	// Fonction pour gérer le Reflector
	func handleReflector(conn net.Conn) {
		data, err := receivePacket()
		if err != nil {
			log.Println("Erreur de réception:", err)
			return
		}

		// Paquet TWAMP-Test reçu, renvoyer un paquet réfléchi
		log.Println("Paquet TWAMP-Test reçu.")
		err = handleTwampTestReflector(data)
			if err != nil {
				log.Printf("Erreur de traitement du paquet TWAMP-Test : %v", err)
			}
		}

// Exemple de construction de la réponse TWAMP-Test
func buildTwampTestResponse(data []byte) []byte {
	var receivedPacket TwampTestPacket
	err := deserializeTwampTestPacket(data, &receivedPacket)
	if err != nil {
		log.Fatalf("Erreur de désérialisation du paquet TWAMP-Test : %v", err)
	}

	// Mettre à jour le champ ReceptionTimestamp
	receivedPacket.ReceptionTimestamp = uint64(time.Now().UnixNano())

	// Sérialiser la réponse TWAMP-Test
	serializedPacket, err := SerializeTwampTestPacket(&receivedPacket)
	if err != nil {
		log.Fatalf("Erreur de sérialisation du paquet TWAMP-Test réfléchi : %v", err)
	}
	return serializedPacket
}

func main() {

	// Création d'une variable de type Metrics
	metrics := Metrics{}

	// Appel de la fonction en passant un pointeur vers metrics
	handleSender(&metrics)

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
