package agent

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
	"context"
	"encoding/json"


	_ "github.com/lib/pq"
	"github.com/gorilla/websocket"
)

// Réception d'un paquet UDP
func receivePacket(conn *net.UDPConn) ([]byte, error) {
    conn.SetReadDeadline(time.Now().Add(5 * time.Second)) // timeout 2s
    buffer := make([]byte, AppConfig.Network.PacketSize)
    n, _, err := conn.ReadFromUDP(buffer)
    if err != nil {
        if ne, ok := err.(net.Error); ok && ne.Timeout() {
            log.Printf("❌ Timeout lecture UDP après 2 secondes")
            return nil, fmt.Errorf("timeout lecture UDP")
        }
        log.Printf("❌ Erreur de réception UDP: %v", err)
        return nil, fmt.Errorf("échec de la réception du paquet UDP: %v", err)
    }
	
    if n == 0 {
        log.Printf("❌ Paquet reçu vide (0 octet)")
        return nil, fmt.Errorf("paquet reçu vide (0 octet)")
    }
    if n > len(buffer) {
        log.Printf("❌ Paquet trop grand: %d octets", n)
        return nil, fmt.Errorf("paquet trop grand: %d octets", n)
    }
    log.Printf("✅ Paquet UDP reçu (%d octets)", n)
    return buffer[:n], nil
}

func StartTest(config TestConfig, ws *websocket.Conn) (*PacketStats, *QoSMetrics, error) {
	log.Printf("🚀 [Client] Lancement du test ID %d...", config.TestID)

	// Étape 0 bis : Vérification des paramètres
	if config.Duration <= 0 {
		log.Println("❌ ERREUR : Durée de test invalide ou manquante.")
		if ws != nil {
			_ = sendTestStatus(ws, config.TestID, "failed")
		}
		return nil, nil, fmt.Errorf("Durée de test invalide : %d ms", config.Duration)
	}

	if config.Profile == nil || config.Profile.SendingInterval <= 0 {
		log.Println("❌ ERREUR : Intervalle d'envoi invalide ou config.Profile manquant.")
		if ws != nil {
			_ = sendTestStatus(ws, config.TestID, "failed")
		}
		return nil, nil, fmt.Errorf("Intervalle d'envoi invalide : %d ms", config.Profile.SendingInterval)
	}

	// Étape 1 : Parsing durée
	log.Printf("Durée brute (ms) : %d", config.Duration)
	duration := time.Duration(config.Duration) * time.Millisecond
	log.Printf("Durée convertie : %v", duration)

	// Initialisation du contexte avec timeout
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	// Étape 2 : Initialisation
	stats := &PacketStats{
		StartTime:      time.Now(),
		TargetAddress:  config.TargetIP,
		TargetPort:     config.TargetPort,
		LatencySamples: make([]int64, 0),
		TestID:         config.TestID,
		TargetID:       config.TargetID, 
	}
	qos := &QoSMetrics{}

	// Étape 3 : Socket UDP
	localAddr := &net.UDPAddr{
		IP:   net.ParseIP(config.SourceIP),
		Port: config.SourcePort,
	}
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("❌ Socket UDP erreur (%s:%d) : %v",
			config.SourceIP, config.SourcePort, err)
	}
	defer conn.Close()

	log.Printf("✅ Socket bindé sur %s:%d", config.SourceIP, config.SourcePort)

	// Étape 4 : WebSocket statut "In progress"
	if ws != nil {
		if err := sendTestStatus(ws, config.TestID, "In progress"); err != nil {
			log.Printf("❌ Erreur statut WebSocket : %v", err)
		}
		_ = ws.WriteMessage(websocket.TextMessage, []byte("🟢 WS Test commencé"))
	}

	// Étape 5 : Boucle d'envoi
	intervalDuration := time.Duration(config.Profile.SendingInterval)
	log.Printf("Intervalle entre paquets : %v", intervalDuration)

	log.Println("🚀 Début de la boucle d'envoi...")

	for {
		select {
		case <-ctx.Done():
			log.Println("⏱️ Durée du test atteinte.")
			goto finDuTest
		default:
			log.Println("🔄 Envoi paquet UDP...")
			if err := handleSender(stats, qos, conn, ws); err != nil {
				log.Printf("❌ Erreur handleSender : %v", err)
				if ws != nil {
					_ = sendTestStatus(ws, config.TestID, "failed")
				}
				return nil, nil, err
			}
			time.Sleep(intervalDuration)
		}
	}

finDuTest:
	// Étape 6 : Calcul métriques
	log.Println("📊 Calcul des métriques QoS...")

	if stats.SentPackets > 0 {
		qos.PacketLossPercent = float64(stats.SentPackets-stats.ReceivedPackets) / float64(stats.SentPackets) * 100
	}
	if len(stats.LatencySamples) > 0 {
		var totalLatency int64
		for _, lat := range stats.LatencySamples {
			totalLatency += lat
		}
		qos.AvgLatencyMs = float64(totalLatency) / float64(len(stats.LatencySamples)) / 1e6
	}
	if len(stats.LatencySamples) > 1 {
		var totalJitter int64
		for i := 1; i < len(stats.LatencySamples); i++ {
			totalJitter += abs(stats.LatencySamples[i] - stats.LatencySamples[i-1])
		}
		qos.TotalJitter = totalJitter
		qos.AvgJitterMs = float64(totalJitter) / float64(len(stats.LatencySamples)-1) / 1e6
	}
	if duration.Seconds() >= 1.0 && stats.TotalBytesReceived > 0 {
		qos.AvgThroughputKbps = float64(stats.TotalBytesReceived*8) / duration.Seconds() / 1000
	}

	SetLatestMetrics(qos)
	log.Printf("✅ Métriques calculées : %+v", qos)

		if ws != nil {
		qosMsg := WsQoSMetrics{
			Type:              "qos_metrics",
			TestID:            config.TestID,
			TargetID:          config.TargetID,
			AvgLatencyMs:      qos.AvgLatencyMs,
			AvgJitterMs:       qos.AvgJitterMs,
			AvgThroughputKbps: qos.AvgThroughputKbps,
			PacketLossPercent: qos.PacketLossPercent,
		}

		data, err := json.Marshal(qosMsg)
		if err != nil {
			log.Printf("❌ Erreur JSON WebSocket QoS : %v", err)
		} else if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("❌ Erreur envoi QoS WebSocket : %v", err)
		} else {
			log.Println("📤 Métriques QoS envoyées via WebSocket.")
		}
	}


	// Étape 7 : Envoi Kafka  
	log.Printf("📤 Envoi Kafka : agent_id(TargetID) = %d", config.TargetID)

	kafkaBrokers := []string{"localhost:9092"}
	kafkaTopic := "test-results"
	result := TestResult1{
		TestID:         config.TestID,
		TargetID:       config.TargetID,
		LatencyMs:      qos.AvgLatencyMs,
		JitterMs:       qos.AvgJitterMs,
		ThroughputKbps: qos.AvgThroughputKbps,
	}
	if err := sendTestResultKafka(kafkaBrokers, kafkaTopic, result); err != nil {
		log.Printf("❌ Erreur Kafka : %v", err)
	} else {
		log.Printf("✅ Résultat Kafka envoyé (TestID %d)", config.TestID)
	}

	// Étape 8 : WebSocket "completed"
	if ws != nil {
		if err := sendTestStatus(ws, config.TestID, "completed"); err != nil {
			log.Printf("❌ Erreur WebSocket fin: %v", err)
		}
	}
	log.Println("✅ Test terminé avec succès.")
	return stats, qos, nil
}


// Fonction utilitaire abs pour int64
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
	metricsMutex.Lock()
	defer metricsMutex.Unlock()
	latestMetrics = metrics
	fmt.Println("Les métriques ont été mises à jour :", latestMetrics)
}

func GetLatestMetrics() *QoSMetrics {
	metricsMutex.RLock()
	defer metricsMutex.RUnlock()
	return latestMetrics
}

// Structure à envoyer via WebSocket
type WsTestResult struct {
    TestID         int     `json:"test_id"` 
    TargetID       int64   `json:"target_id"`
    ReflectorID    int     `json:"reflector_id"`
    LatencyMs      float64 `json:"latency_ms"`
    JitterMs       float64 `json:"jitter_ms"`
    ThroughputKbps float64 `json:"throughput_kbps"`
}


func handleSender(stats *PacketStats, qos *QoSMetrics, conn *net.UDPConn, wsConn *websocket.Conn) error {
	log.Println("🚀 handleSender : début")
	
	// ✅ Vérification que l'adresse cible est bien définie
	if stats.TargetAddress == "" || net.ParseIP(stats.TargetAddress) == nil || stats.TargetPort == 0 {
		log.Printf("❌ destination UDP invalide : IP=%q, Port=%d", stats.TargetAddress, stats.TargetPort)
		return fmt.Errorf("destination UDP invalide : IP=%q, Port=%d", stats.TargetAddress, stats.TargetPort)
	}

	// Création de l'adresse de destination
	destAddr := &net.UDPAddr{
		IP:   net.ParseIP(stats.TargetAddress),
		Port: stats.TargetPort,
	}
	log.Printf("➡️ Destination UDP: %s:%d", destAddr.IP.String(), destAddr.Port)

	log.Printf("🔵 Socket locale liée à: %s", conn.LocalAddr().String())

	// Incrément du compteur de paquets à envoyer
	stats.SentPackets++

	// Création du paquet TWAMP
	timestamp := uint64(time.Now().UnixNano())
	twampPacket := TwampTestPacket{
		SequenceNumber:        uint32(stats.SentPackets),
		Timestamp:             timestamp,
		ErrorEstimation:       0,
		MBZ:                   0,
		ReceptionTimestamp:    0,
		SenderSequenceNumber:  uint64(stats.SentPackets),
		SenderTimestamp:       timestamp,
		SenderErrorEstimation: 0,
		SenderTTL:             255,
		Padding:               make([]byte, 20),
	}

	serializedPacket, err := SerializeTwampTestPacket(&twampPacket)
	if err != nil {
		log.Printf("❌ Erreur de sérialisation TWAMP: %v", err)
		return err
	}

	if _, err := conn.WriteToUDP(serializedPacket, destAddr); err != nil {
		log.Printf("❌ Erreur d'envoi UDP: %v", err)
		return err
	}
	log.Printf("📨 Attente de réponse UDP sur %s", conn.LocalAddr().String())

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	receivedData, err := receivePacket(conn)
	if err != nil {
		log.Printf("❌ Erreur de réception UDP: %v", err)
		return err
	}
	log.Printf("✅ Paquet reçu (%d octets)", len(receivedData))
	stats.TotalBytesReceived += int64(len(receivedData))
	stats.ReceivedPackets++

	var receivedPacket TwampTestPacket
	if err := deserializeTwampTestPacket(receivedData, &receivedPacket); err != nil {
		log.Printf("❌ Erreur de désérialisation TWAMP: %v", err)
		return err
	}
	receivedPacket.ReceptionTimestamp = uint64(time.Now().UnixNano())

	// Calcul de la latence
	latency := int64(receivedPacket.ReceptionTimestamp - receivedPacket.SenderTimestamp)
	stats.LatencySamples = append(stats.LatencySamples, latency)
	stats.LastLatency = latency

	// Calcul du jitter
	var jitterMs float64
	if len(stats.LatencySamples) > 1 {
		prev := stats.LatencySamples[len(stats.LatencySamples)-2]
		jitter := abs(latency - prev)
		qos.TotalJitter += jitter
		jitterMs = float64(qos.TotalJitter) / float64(len(stats.LatencySamples)-1) / 1e6
	}

	// Calcul du débit
	elapsed := time.Since(stats.StartTime).Seconds()
	var throughputKbps float64
	if elapsed > 0 {
		throughputKbps = float64(stats.TotalBytesReceived*8) / 1000 / elapsed
	}

	latencyMs := float64(latency) / 1e6
	log.Printf("🧪 DEBUG TargetID stats: %d", stats.TargetID)

	// Construction du message WebSocket
		wsResult := WsTestResult{
		TestID:    stats.TestID,
		TargetID: int64(stats.TargetID),
		LatencyMs: latencyMs,
		JitterMs:  jitterMs,
		ThroughputKbps: throughputKbps,
	}

	log.Printf("📊 Résultat WebSocket à envoyer: %+v", wsResult)

	// Envoi WebSocket
	if wsConn != nil {
		log.Printf("📡 Envoi WebSocket actif : TestID=%d", stats.TestID)
		data, err := json.Marshal(wsResult)
		if err != nil {
			log.Printf("❌ Erreur JSON WebSocket : %v", err)
		} else if err := wsConn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("❌ Erreur envoi WebSocket : %v", err)
		} else {
			log.Println("📤 Résultat envoyé via WebSocket.")
		}
	}

	// Affichage console
	fmt.Printf("✅ [Paquet %d] Latence: %.3f ms | Jitter: %.3f ms | Débit: %.3f kbps\n",
		stats.SentPackets, latencyMs, jitterMs, throughputKbps)

	return nil
}

func handleReflector(conn *net.UDPConn, addr *net.UDPAddr, data []byte) error {
	log.Println("🟢 Reflector lancé...")
	log.Printf("🟢 Reflector en écoute sur %s", conn.LocalAddr().String())

	log.Printf("🛠️ handleReflector appelé pour %s (taille: %d)", addr.String(), len(data))

	// ✅ Log pour vérifier réception
	log.Printf("📥 Paquet brut reçu de %s (%d octets)", addr.String(), len(data))
	log.Printf("📦 Contenu brut (hex) : %x", data)

	var receivedPacket TwampTestPacket
	if err := deserializeTwampTestPacket(data, &receivedPacket); err != nil {
		log.Printf("❌ Erreur de désérialisation TWAMP : %v", err)
		return fmt.Errorf("erreur de désérialisation: %v", err)
	}
	log.Printf("🔍 Paquet désérialisé : Sequence #%d, Timestamp=%d", receivedPacket.SequenceNumber, receivedPacket.Timestamp)

	// ✅ Log contenu du paquet reçu
	log.Printf("📊 TWAMP reçu ➤ Seq: %d, SenderTS: %d", receivedPacket.SequenceNumber, receivedPacket.SenderTimestamp)

	// Ajout du timestamp de réception
	receivedPacket.ReceptionTimestamp = uint64(time.Now().UnixNano())

	// ✅ Log sur le timestamp de réception
	log.Printf("⏱️ Ajout ReceptionTimestamp: %d", receivedPacket.ReceptionTimestamp)

	serializedPacket, err := SerializeTwampTestPacket(&receivedPacket)
	if err != nil {
		log.Printf("❌ Erreur de sérialisation TWAMP : %v", err)
		return fmt.Errorf("erreur de sérialisation: %v", err)
	}
	// ✅ Log paquet à renvoyer
	log.Printf("📤 Paquet TWAMP prêt à renvoyer (%d octets) à %s", len(serializedPacket), addr.String())

	// Envoi
	if _, err := conn.WriteToUDP(serializedPacket, addr); err != nil {
		log.Printf("❌ Échec envoi UDP : %v", err)
		return fmt.Errorf("échec de l'envoi de la réponse: %v", err)
	}
	log.Printf("📤 Réponse envoyée à %s (%d octets)", addr.String(), len(serializedPacket))

	// ✅ Log final succès
	log.Printf("✅ Réponse envoyée à %s ➤ Seq: %d", addr.String(), receivedPacket.SequenceNumber)
	return nil
}

func listenAsReflector(ip string, port int) {
	log.Println("🎧 [Reflector] Initialisation du réflecteur UDP...")

	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP(ip),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("❌ Erreur écoute UDP: %v", err)
	}
	defer conn.Close()

	buffer := make([]byte, 1500)
	log.Printf("🟢 Reflector en écoute sur %s", addr.String())

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("⚠️ Erreur lecture paquet UDP: %v", err)
			continue
		}
		log.Printf("📥 Reflector a reçu un paquet de %s (%d octets)", remoteAddr.String(), n)

		// Copier le buffer pour éviter les conflits entre goroutines
		dataCopy := make([]byte, n)
		copy(dataCopy, buffer[:n])

		go func(data []byte, addr *net.UDPAddr) {
			if err := handleReflector(conn, addr, data); err != nil {
				log.Printf("❌ Erreur handleReflector: %v", err)
			}
		}(dataCopy, remoteAddr)
	}
}


type TestStatusMessage struct {
	Type    string      `json:"type"`
	Payload TestStatus  `json:"payload"`
}


func Start(db *sql.DB) {
	log.Println("🚀 [Agent] Démarrage de l'agent TWAMP...")

	// Chargement de la configuration
	LoadConfig("agent/config.yaml")

	go Serveur()
	log.Println("📡 [Agent] Serveur TCP lancé.")

	//go listenAsReflector("127.0.0.1", 8081)//7
	go listenAsReflector("127.0.0.1", 8080)//8
	//go listenAsReflector("127.0.0.1", 50051)//10
	//go listenAsReflector("127.0.0.1", 8082)//11
	//go listenAsReflector(AppConfig.Reflector.IP, AppConfig.Reflector.Port)


	//go listenAsReflector()

	// ✅ Lancement du WebSocket Agent (avant testWorker)
	wsConn, err := StartWebSocketAgent()
	if err != nil {
		log.Fatalf("❌ Impossible d'établir la connexion WebSocket : %v", err)
	}
	defer wsConn.Close()
	log.Println("🔌 [Agent] Connexion WebSocket établie.")

	// ✅ Démarrage du testWorker (tests TWAMP)
	ctx := context.Background()
	brokers := []string{"localhost:9092"}
	go testWorker(ctx, brokers)

	// Lancement du listener Kafka
	go func() {
		defer log.Println("❌ [Agent] Kafka Listener a quitté.")
		ListenToTestRequestsFromKafka(db)
	}()
	log.Println("📨 [Agent] Écoute Kafka lancée.")

	// 🛰️ Démarrage du serveur gRPC Agent
	go func() {
		defer log.Println("❌ [Agent] Serveur gRPC arrêté.")
		startAgentServer()
	}()
	log.Println("🛰️ [Agent] Serveur gRPC lancé.")

	// Bloquer pour maintenir l’agent actif
	select {}
}