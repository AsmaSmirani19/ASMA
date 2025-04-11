package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
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
func startTest(ctx context.Context, testParams string) (*PacketStats, *QoSMetrics, error) {

	params, err := parseTestParameters(testParams)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid test parameters: %v", err)
	}

	// 2. Configurer le test avec ces paramètres
	Stats := &PacketStats{
		StartTime:      time.Now(),
		TargetAddress:  params.TargetIP,
		TargetPort:     params.TargetPort,
		LatencySamples: make([]int64, 0),
	}
	qos := &QoSMetrics{}

	// 3. Exécuter le test
	testEnd := Stats.StartTime.Add(params.Duration)
	for time.Now().Before(testEnd) {
		err := handleSender(Stats, qos)
		if err != nil {
			return nil, nil, fmt.Errorf("erreur lors de l'envoi du paquet: %v", err)
		}
		time.Sleep(params.PacketInterval)
	}
	// 4. Calculer les métriques finales
	if Stats.SentPackets > 0 {
		qos.PacketLossPercent = float64(Stats.SentPackets-Stats.ReceivedPackets) / float64(Stats.SentPackets) * 100
	}

	// Calcul de la latence moyenne
	if len(Stats.LatencySamples) > 0 {
		var totalLatency int64
		for _, lat := range Stats.LatencySamples {
			totalLatency += lat
		}
		qos.AvgLatencyMs = totalLatency / int64(len(Stats.LatencySamples)) / 1e6
	}
	// Calcul de la gigue
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

	// Calcul du débit moyen
	if params.Duration.Seconds() > 0 {
		qos.AvgThroughputKbps = float64(Stats.TotalBytesReceived*8) / params.Duration.Seconds() / 1024
	}
	// Enregistrer les dernières métriques
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

func listenForTestCommands() {
	const (
		initialReconnectDelay = 1 * time.Second
		maxReconnectDelay     = 30 * time.Second
		connectionTimeout     = 10 * time.Second
		receiveTimeout        = 30 * time.Second
	)
	const grpcServerAddr = "localhost:50051"

	var reconnectDelay time.Duration = initialReconnectDelay

	for {
		// Création d'un contexte avec timeout pour la connexion
		connCtx, connCancel := context.WithTimeout(context.Background(), connectionTimeout)

		conn, err := grpc.DialContext(
			connCtx,
			grpcServerAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		connCancel() // Important: libérer les ressources du contexte

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

		// Création d'un contexte avec timeout pour le stream
		streamCtx, streamCancel := context.WithCancel(context.Background())
		defer streamCancel()

		stream, err := client.StreamTestCommands(streamCtx, &testpb.TestCommand{
			TestId: "CLIENT_READY",
		})
		if err != nil {
			log.Printf("Échec de création du stream: %v", err)
			conn.Close()
			time.Sleep(reconnectDelay)
			continue
		}

		log.Println("Connecté au serveur, en attente de commandes...")

		// Boucle de réception avec timeout
	receiveLoop:
		for {
			// Configurer un timeout pour chaque opération de réception
			select {
			case <-time.After(receiveTimeout):
				log.Printf("Timeout: aucune commande reçue depuis %v", receiveTimeout)
				break receiveLoop

			default:
				res, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						log.Printf("Fin du stream")
					} else {
						log.Printf("Erreur de réception: %v", err)
					}
					break receiveLoop
				}

				log.Printf("Commande reçue: %s - %s", res.Status, res.Result)

				// Lancer le test dans une goroutine avec son propre contexte
				go func(testParams string) {
					testCtx, testCancel := context.WithTimeout(context.Background(), 5*time.Minute)
					defer testCancel()

					_, _, err := startTest(testCtx, testParams)
					if err != nil {
						log.Printf("Erreur lors du test: %v,", err)
					}
				}(res.Result)
			}
		}

		conn.Close()
	}
}

// Fonction utilitaire pour la valeur minimale
func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
func sendInitialTestRequest() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Connexion initiale impossible: %v", err)
	}
	defer conn.Close()

	client := testpb.NewTestServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &testpb.TestCommand{
		TestId:     "Latence",
		Parameters: "10 paquets ICMP vers 8.8.8.8",
	}

	res, err := client.StreamTestCommands(ctx, req)
	if err != nil {
		log.Printf("Erreur de demande initiale de test QoS: %v", err)
		return
	}

	log.Printf("Réponse initiale du serveur : %v", res)
}

func main() {
	// 1. Démarrage du WebSocket Agent pour les résultats en temps réel
	go StartWebSocketAgent()

	// 2. Écoute des commandes du serveur gRPC (streaming)
	go listenForTestCommands()

	// 3. Connexion ponctuelle au serveur pour envoyer une demande simple (optionnelle)
	sendInitialTestRequest()

	// 4. Bloquer le main thread (ou attendre via un signal OS pour arrêt propre)
	select {} // infinite block, or use signal.Notify for graceful shutdown
}
