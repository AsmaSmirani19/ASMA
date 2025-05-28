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
func receivePacket(conn *net.UDPConn) ([]byte, error) {
    conn.SetReadDeadline(time.Now().Add(2 * time.Second)) // timeout 2s
    buffer := make([]byte, AppConfig.Network.PacketSize)
    n, _, err := conn.ReadFromUDP(buffer)
    if err != nil {
        if ne, ok := err.(net.Error); ok && ne.Timeout() {
            // Timeout de lecture UDP : on peut retourner nil et gérer ça ailleurs
            return nil, fmt.Errorf("timeout lecture UDP")
        }
        return nil, fmt.Errorf("échec de la réception du paquet UDP: %v", err)
    }
    if n == 0 {
        return nil, fmt.Errorf("paquet reçu vide (0 octet)")
    }
    if n > len(buffer) {
        return nil, fmt.Errorf("paquet trop grand: %d octets", n)
    }
    return buffer[:n], nil
}

func StartTest(db *sql.DB, config TestConfig, ws *websocket.Conn) (*PacketStats, *QoSMetrics, error) {
    log.Printf("🚀 [Client] Lancement du test ID %d...", config.TestID)

	
    // Étape 1 : Parse la durée
    duration := time.Duration(config.Duration)

    // Étape 3 : Initialisation
    log.Println("⚙️ Étape 3 : Initialisation des structures de métriques...")


    stats := &PacketStats{
        StartTime:      time.Now(),
        TargetAddress:  config.TargetIP,
        TargetPort:     config.TargetPort,
        LatencySamples: make([]int64, 0),
		TestID:         config.TestID,
	
    }
    qos := &QoSMetrics{}

    localAddr := &net.UDPAddr{
        IP:   net.ParseIP(config.SourceIP),
        Port: config.SourcePort,
    }

    // Étape 4 : Création du socket UDP
    log.Println("🔌 Étape 4 : Création du socket UDP...")
    conn, err := net.ListenUDP("udp", localAddr)
    if err != nil {
        return nil, nil, fmt.Errorf("❌ Échec de l'ouverture du socket UDP (%s:%d) : %v",
            config.SourceIP, config.SourcePort, err)
    }
    defer conn.Close()
    log.Printf("✅ Socket bindé sur %s:%d", config.SourceIP, config.SourcePort)

    // Étape 5 : Exécution du test
    log.Println("🚀 Étape 5 : Lancement de la boucle d'envoi des paquets...")
    if ws != nil {
        log.Println("📤 Envoi du statut 'running' via WebSocket...")
        sendTestStatus(ws, config.TestID, "running")
		err := ws.WriteMessage(websocket.TextMessage, []byte("🟢 WS Test commencé"))
		if err != nil {
			log.Printf("❌ Impossible d'écrire sur WebSocket: %v", err)
		} else {
			log.Println("✅ Test de WebSocket : message envoyé")
		}

    }

    testEnd := stats.StartTime.Add(duration)

	if config.Profile == nil {
    log.Println("❌ Erreur : config.Profile est nil")
    return nil, nil, fmt.Errorf("config.Profile est nil")
}

	intervalMs := config.Profile.SendingInterval
	intervalDuration := time.Duration(intervalMs)

    for time.Now().Before(testEnd) {
        if err := handleSender(stats, qos, conn, ws); err != nil {
            log.Printf("❌ Erreur dans handleSender : %v", err)
            if ws != nil {
                sendTestStatus(ws, config.TestID, "failed")
            }
            return nil, nil, err
        }
        time.Sleep(intervalDuration)
    }

    // Étape 6 : Calcul des métriques QoS
    log.Println("📊 Étape 6 : Calcul des métriques QoS...")
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

    // === Intégration Kafka ===
    kafkaBrokers := []string{"localhost:9092"} // ou à prendre dans ta config
    kafkaTopic := "test-results"

    result := TestResult1{
        TestID:         config.TestID,
        LatencyMs:      qos.AvgLatencyMs,
        JitterMs:       qos.AvgJitterMs,
        ThroughputKbps: qos.AvgThroughputKbps,
    }

    if err := sendTestResultKafka(kafkaBrokers, kafkaTopic, result); err != nil {
        log.Printf("❌ Erreur lors de l'envoi Kafka du résultat : %v", err)
        // Optionnel : gérer l'erreur (stopper test, retry, etc.)
    } else {
        log.Printf("✅ Résultat Kafka envoyé pour TestID %d", config.TestID)
    }

    // Étape 7 : Fin du test, envoi statut "finished" via WS
    if ws != nil {
        log.Println("📤 Envoi du statut 'finished' via WebSocket...")
        sendTestStatus(ws, config.TestID, "finished")
    }

    log.Println("✅ Test terminé avec succès.")
    return stats, qos, nil
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
	TestID          int     `json:"test_id"` 
	LatencyMs       float64 `json:"latency_ms"`
	JitterMs        float64 `json:"jitter_ms"`
	ThroughputKbps  float64 `json:"throughput_kbps"`
}

func handleSender(stats *PacketStats, qos *QoSMetrics, conn *net.UDPConn, wsConn *websocket.Conn) error {
	log.Println("🚀 handleSender : début")

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

	// Construction du message WebSocket
	wsResult := WsTestResult{
		TestID:         stats.TestID,
		LatencyMs:      latencyMs,
		JitterMs:       jitterMs,
		ThroughputKbps: throughputKbps,
	}

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

	var receivedPacket TwampTestPacket
	if err := deserializeTwampTestPacket(data, &receivedPacket); err != nil {
		return fmt.Errorf("erreur de désérialisation: %v", err)
	}

	// Ne pas modifier le SenderTimestamp d'origine
	receivedPacket.ReceptionTimestamp = uint64(time.Now().UnixNano())

	serializedPacket, err := SerializeTwampTestPacket(&receivedPacket)
	if err != nil {
		log.Printf("❌ Sérialisation échouée: %v", err)
		return fmt.Errorf("erreur de sérialisation: %v", err)
	}

	if _, err := conn.WriteToUDP(serializedPacket, addr); err != nil {
		return fmt.Errorf("échec de l'envoi de la réponse: %v", err)
	}

	log.Printf("✅ Réponse envoyée à %s (%d octets), Seq #%d",
		addr.String(), len(serializedPacket), receivedPacket.SequenceNumber)
	return nil
}


func listenAsReflector() {
	addr := net.UDPAddr{
		Port: AppConfig.Reflector.Port,
		IP:   net.ParseIP(AppConfig.Reflector.IP),
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


func Start(db *sql.DB) {
	log.Println("🚀 [Agent] Démarrage de l'agent TWAMP...")

	// Chargement de la configuration
	LoadConfig("agent/config.yaml")

	// Lancement du serveur local (TWAMP listener)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🔥 [Agent] Panic dans le serveur : %v", r)
			}
		}()
		Serveur()
	}()
	log.Println("📡 [Agent] Serveur TCP lancé.")
	time.Sleep(1 * time.Second) // Laisser le temps au serveur de démarrer

	// Lancement du Reflector TWAMP EN PREMIER
	go func() {
		defer log.Println("❌ [Agent] Reflector a quitté.")
		log.Println("🔁 [Agent] Lancement du Reflector TWAMP...")
		listenAsReflector()
	}()
	log.Println("✅ [Agent] Reflector TWAMP lancé.")
	time.Sleep(1 * time.Second) 

	// ✅ Lancement du WebSocket Agent (avant testWorker)
	wsConn, err := StartWebSocketAgent()
	if err != nil {
		log.Fatalf("❌ Impossible d'établir la connexion WebSocket : %v", err)
	}
	defer wsConn.Close()
	log.Println("🔌 [Agent] Connexion WebSocket établie.")


	// **ICI : démarrer le worker qui consomme la file de tests**
	ctx := context.Background()
	go testWorker(ctx, db) 

	// Lancement du listener Kafka
	go func() {
		defer log.Println("❌ [Agent] Kafka Listener a quitté.")
		ListenToTestRequestsFromKafka(db)
	}()
	log.Println("📨 [Agent] Écoute Kafka lancée.")


	// Lancement du client gRPC stream
	go func() {
		defer log.Println("❌ [Agent] Client Stream arrêté.")
		startClientStream()
	}()
	log.Println("🌐 [Agent] gRPC Stream Client lancé.")


	select {}
}




