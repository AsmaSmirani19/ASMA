package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"log"
	"mon-projet-go/testpb"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PacketStats struct {
	SentPackets        int
	ReceivedPackets    int
	TotalBytesSent     int64
	TotalBytesReceived int64
	LastLatency        int64
	StartTime          time.Time
	LatencySamples     []int64
	TargetAddress      string
	TargetPort         int
}

type QoSMetrics struct {
	PacketLossPercent float64
	AvgLatencyMs      int64
	AvgJitterMs       int64
	AvgThroughputKbps float64
	TotalJitter       int64
}

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

func deserializeTwampTestPacket(data []byte, pkt *TwampTestPacket) error {
	if len(data) < 42 {
		return fmt.Errorf("paquet trop court")
	}
	buf := bytes.NewReader(data)
	return binary.Read(buf, binary.BigEndian, pkt)
}

// Envoi d'un paquet UDP
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

	_, err = conn.Write(packet)
	return err
}

// Réception d'un paquet UDP
func receivePacket() ([]byte, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 5000})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	buffer := make([]byte, 1500)

	// Lire les données envoyées via UDP
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, fmt.Errorf("échec de la réception du paquet UDP: %v", err)
	}
	if n > len(buffer) {
		return nil, fmt.Errorf("paquet trop grand: %d octets", n)
	}
	return buffer[:n], nil
}

// Démarrage du test QoS
func startTest(testParams string) (*PacketStats, *QoSMetrics, error) {

	params, err := parseTestParameters(testParams)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid test parameters: %v", err)
	}

	Stats := &PacketStats{
		StartTime:      time.Now(),
		TargetAddress:  params.TargetIP,
		TargetPort:     params.TargetPort,
		LatencySamples: make([]int64, 0),
	}
	qos := &QoSMetrics{}

	testEnd := Stats.StartTime.Add(params.Duration)
	for time.Now().Before(testEnd) {
		err := handleSender(Stats, qos)
		if err != nil {
			return nil, nil, fmt.Errorf("erreur lors de l'envoi du paquet: %v", err)
		}
		time.Sleep(params.PacketInterval)
	}

	if Stats.SentPackets > 0 {
		qos.PacketLossPercent = float64(Stats.SentPackets-Stats.ReceivedPackets) / float64(Stats.SentPackets) * 100
	}

	if len(Stats.LatencySamples) > 0 {
		var totalLatency int64
		for _, lat := range Stats.LatencySamples {
			totalLatency += lat
		}
		qos.AvgLatencyMs = totalLatency / int64(len(Stats.LatencySamples)) / 1e6
	}

	if len(Stats.LatencySamples) > 1 {
		var totalJitter int64
		for i := 1; i < len(Stats.LatencySamples); i++ {
			totalJitter += abs(Stats.LatencySamples[i] - Stats.LatencySamples[i-1])
		}
		qos.TotalJitter = totalJitter
		qos.AvgJitterMs = qos.TotalJitter / int64(len(Stats.LatencySamples)-1) / 1e6
	} else {
		qos.AvgJitterMs = 0
	}

	if params.Duration.Seconds() > 0 {
		qos.AvgThroughputKbps = float64(Stats.TotalBytesReceived*8) / params.Duration.Seconds() / 1024
	}

	SetLatestMetrics(qos)

	return Stats, qos, nil
}

// Struct pour les paramètres parsés
type TestParams struct {
	TargetIP       string
	TargetPort     int
	Duration       time.Duration
	PacketInterval time.Duration
}

// Extraire  les paramètres
func parseTestParameters(input string) (*TestParams, error) {
	params := &TestParams{
		// Valeurs par défaut
		TargetIP:       "127.0.0.1",
		TargetPort:     5000,
		Duration:       30 * time.Second,
		PacketInterval: 1 * time.Second,
	}

	// Exemple de parsing simple (à adapter)
	parts := strings.Split(input, ",")
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			continue
		}

		switch strings.TrimSpace(kv[0]) {
		case "target":
			params.TargetIP = strings.TrimSpace(kv[1])
		case "port":
			if port, err := strconv.Atoi(kv[1]); err == nil {
				params.TargetPort = port
			}
		case "duration":
			if dur, err := time.ParseDuration(kv[1]); err == nil {
				params.Duration = dur
			}
		case "interval":
			if interval, err := time.ParseDuration(kv[1]); err == nil {
				params.PacketInterval = interval
			}
		}
	}
	// Validation des paramètres
	if params.TargetIP == "" || params.TargetPort <= 0 {
		return nil, fmt.Errorf("paramètres invalides : adresse ou port non spécifiés")
	}

	if params.Duration <= 0 {
		return nil, fmt.Errorf("durée du test doit être supérieure à 0")
	}

	return params, nil
}

// Fonction utilitaire pour calculer la valeur absolue
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// Déclaration d'une variable globale pour stocker les métriques
var (
	latestMetrics *QoSMetrics
	metricsMutex  sync.RWMutex
)

// Fonction pour enregistrer les métriques les plus récentes
func SetLatestMetrics(metrics *QoSMetrics) {
	latestMetrics = metrics
	defer metricsMutex.Unlock()
	fmt.Println("Les métriques ont été mises à jour :", latestMetrics)
}

func GetLatestMetrics() *QoSMetrics {
	metricsMutex.RLock()
	defer metricsMutex.RUnlock()
	return latestMetrics
}

// Envoi de paquets dans le test TWAMP
func handleSender(Stats *PacketStats, qos *QoSMetrics) error {

	twamp_testpaquet := TwampTestPacket{
		SequenceNumber:        uint32(Stats.SentPackets),
		Timestamp:             uint64(time.Now().UnixNano()),
		ErrorEstimation:       0,
		MBZ:                   0,
		ReceptionTimestamp:    0,
		SenderSequenceNumber:  uint64(Stats.SentPackets),
		SenderTimestamp:       uint64(time.Now().UnixNano()),
		SenderErrorEstimation: 0,
		SenderTTL:             255,
		Padding:               make([]byte, 20),
	}
	// envoi paquet twamp-test
	serializeTwampTestPacket, err := SerializeTwampTestPacket(&twamp_testpaquet)
	if err != nil {
		log.Printf("Erreur de sérialisation: %v", err)
		return fmt.Errorf("erreur de sérialisation du paquet TWAMP: %w", err)
	}

	err = SendPacket(serializeTwampTestPacket, Stats.TargetAddress, Stats.TargetPort)
	if err != nil {
		log.Printf("Erreur d'envoi: %v", err)
		return fmt.Errorf("erreur lors de l'envoi du paquet TWAMP: %w", err)
	}
	//recevoir twamp-test reflecté
	receivedData, err := receivePacket()
	if err != nil {
		log.Printf("Erreur de réception: %v", err)
		return fmt.Errorf("réception paquet %d échouée: %w", Stats.SentPackets+1, err)
	}

	var receivedPacket TwampTestPacket
	err = deserializeTwampTestPacket(receivedData, &receivedPacket)
	if err != nil {
		log.Printf("Erreur de désérialisation: %v", err)
		return fmt.Errorf("erreur de désérialisation du paquet reçu: %w", err)
	}

	receivedPacket.ReceptionTimestamp = uint64(time.Now().UnixNano())

	// Calcul de la latence
	latency := int64(receivedPacket.ReceptionTimestamp - receivedPacket.SenderTimestamp)
	Stats.LatencySamples = append(Stats.LatencySamples, latency)
	Stats.LastLatency = latency

	// Calcul du jitter
	if len(Stats.LatencySamples) > 1 {
		prev := Stats.LatencySamples[len(Stats.LatencySamples)-2]
		jitter := abs(latency - prev)
		qos.TotalJitter += jitter
	}

	Stats.ReceivedPackets++
	Stats.SentPackets++

	fmt.Printf("[Paquet %d] Latence: %d ms | Jitter: %d ms\n",
		Stats.SentPackets,
		latency/1e6,
		qos.TotalJitter/1e6/int64(len(Stats.LatencySamples)))
	return nil
}

func handleReflector(data []byte) error {
	var receivedPacket TwampTestPacket

	// 1. Désérialisation du paquet reçu
	err := deserializeTwampTestPacket(data, &receivedPacket)
	if err != nil {
		return fmt.Errorf("erreur de désérialisation: %v", err)
	}
	// 2. Ajout du timestamp de réception
	receivedPacket.ReceptionTimestamp = uint64(time.Now().UnixNano())
	// 3. Sérialisation du paquet modifié
	serializedtestPacket, err := SerializeTwampTestPacket(&receivedPacket)
	if err != nil {
		return fmt.Errorf("erreur de sérialisation: %v", err)
	}

	return SendPacket(serializedtestPacket, "127.0.0.1", 5000)
}
func listenAsReflector() {
	addr := net.UDPAddr{
		Port: 9000, // le port où le sender envoie les paquets
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Erreur écoute UDP: %v", err)
	}
	defer conn.Close()

	buffer := make([]byte, 1500)

	log.Println("Reflector en écoute sur", addr.String())

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Erreur lecture: %v", err)
			continue
		}

		go func(data []byte, addr *net.UDPAddr) {
			err := handleReflector(data[:n])
			if err != nil {
				log.Printf("Erreur handleReflector: %v", err)
			}
		}(buffer[:n], remoteAddr)
	}
}

// Ajouter cette définition quelque part dans votre code
type twampAgent struct {
	testpb.UnimplementedTestServiceServer
	mu                sync.Mutex
	currentTestCancel context.CancelFunc
}

// RunQuickTest avec gestion de contexte et streaming
func (a *twampAgent) PerformQuickTest(stream testpb.TestService_PerformQuickTestServer) error {
	// Réception d'un message (devrait être une requête)
	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("échec réception message: %v", err)
	}

	// Extraction de la requête de test
	reqMsg, ok := msg.Message.(*testpb.QuickTestMessage_Request)
	if !ok {
		return fmt.Errorf("message reçu n’est pas une requête de test")
	}

	req := reqMsg.Request

	log.Printf("Nouveau test reçu - ID: %s, Paramètres: %s", req.GetTestId(), req.GetParameters())

	// Création d'un contexte annulable pour le test
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	a.mu.Lock()
	if a.currentTestCancel != nil {
		a.currentTestCancel() // Annule tout test précédent
	}
	a.currentTestCancel = cancel
	a.mu.Unlock()

	// Canal pour les résultats
	results := make(chan *QoSMetrics, 10)

	// Exécution du test dans une goroutine
	go func() {
		defer close(results)
		_, metrics, err := startTest(req.GetParameters())
		if err != nil {
			log.Printf("Test %s échoué: %v", req.GetTestId(), err)
			return
		}
		results <- metrics
	}()

	// Envoi des résultats au serveur
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case qos, ok := <-results:
			if !ok {
				return nil
			}

			respMsg := &testpb.QuickTestMessage{
				Message: &testpb.QuickTestMessage_Response{
					Response: &testpb.QuickTestResponse{
						Status: testpb.TestStatus_COMPLETE.String(),
						Result: fmt.Sprintf("loss:%.2f,latency:%.2f,throughput:%.2f",
							float64(qos.PacketLossPercent),
							float64(qos.AvgLatencyMs),
							float64(qos.AvgThroughputKbps)),
					},
				},
			}

			if err := stream.Send(respMsg); err != nil {
				return fmt.Errorf("échec envoi résultats: %v", err)
			}
		}
	}
}

func startGRPCServer() {
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Échec d'écoute : %v", err)
	}

	grpcServer := grpc.NewServer()
	testpb.RegisterTestServiceServer(grpcServer, &twampAgent{})

	log.Println("Agent TWAMP (serveur gRPC) démarré sur le port 50052")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Échec du serveur gRPC : %v", err)
	}
}
func startClientStream() {
	conn, err := grpc.Dial(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Échec de connexion au serveur principal : %v", err)
	}
	defer conn.Close()

	client := testpb.NewTestServiceClient(conn)
	stream, err := client.PerformQuickTest(context.Background())
	if err != nil {
		log.Fatalf("Échec de création du stream : %v", err)
	}

	log.Println("Connexion au serveur principal établie")

	// Boucle pour maintenir la connexion ouverte
	for {
		select {
		case <-stream.Context().Done():
			log.Println("Connexion au serveur terminée")
			return
		}
	}
}

func main() {

	go listenToTestRequestsFromKafka()
	go listenAsReflector()

	// 1. Démarrage WebSocket pour les résultats temps réel
	go StartWebSocketAgent()

	// 2. Démarrage du serveur gRPC
	go startGRPCServer()

	// 3. Connexion au serveur principal et lancement du stream duplex
	startClientStream()
}
