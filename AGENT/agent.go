package agent

import (
	"bytes"
	
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"


	_ "github.com/lib/pq"

	
	"mon-projet-go/core"
	"github.com/gorilla/websocket"
	

)


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
	if len(data) < 49 {
		return fmt.Errorf("paquet trop court: %d octets", len(data))
	}

	buf := bytes.NewReader(data)

	fields := []interface{}{
		&pkt.SequenceNumber,
		&pkt.Timestamp,
		&pkt.ErrorEstimation,
		&pkt.MBZ,
		&pkt.ReceptionTimestamp,
		&pkt.SenderSequenceNumber,
		&pkt.SenderTimestamp,
		&pkt.SenderErrorEstimation,
		&pkt.SenderTTL,
	}

	for _, field := range fields {
		if err := binary.Read(buf, binary.BigEndian, field); err != nil {
			return fmt.Errorf("erreur lecture champ: %w", err)
		}
	}

	// Lire les 20 octets de padding
	pkt.Padding = make([]byte, 20)
	if _, err := buf.Read(pkt.Padding); err != nil {
		return fmt.Errorf("erreur lecture padding: %w", err)
	}

	return nil
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
func receivePacket(conn *net.UDPConn) ([]byte, error) {
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, AppConfig.Network.PacketSize)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
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

func StartTest(db *sql.DB, testID int, ws *websocket.Conn) (*PacketStats, *QoSMetrics, error) {
	log.Printf("🚀 [Client] Lancement du test ID %d...", testID)

	// Étape 1 : Récupération de la configuration
	log.Println("📥 Étape 1 : Chargement de la configuration du test...")
	config, err := core.LoadFullTestConfiguration(db, testID)
	if err != nil {
		return nil, nil, fmt.Errorf("❌ Impossible de récupérer la config du test ID %d : %v", testID, err)
	}
	log.Printf("✅ Configuration chargée : %+v", config)

	// Étape 2 : Marquer le test comme "en attente"
	log.Println("📌 Étape 2 : Marquage du test comme en attente...")
	if err := core.UpdateTestStatus(db, testID, true, false, false, false); err != nil {
		log.Printf("⚠️ Erreur lors de la mise à jour du test en attente : %v", err)
	}
	if ws != nil {
		log.Println("📤 Envoi du statut 'pending' via WebSocket...")
		sendTestStatus(ws, testID, "pending")
	}

	// Étape 3 : Initialisation
	log.Println("⚙️ Étape 3 : Initialisation des structures de métriques...")
	duration := config.Duration
	interval := config.Profile.SendingInterval

	stats := &PacketStats{
		StartTime:      time.Now(),
		TargetAddress:  config.TargetIP,
		TargetPort:     config.TargetPort,
		LatencySamples: make([]int64, 0),
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
		sendTestStatus(ws, testID, "running")
	}

	testEnd := stats.StartTime.Add(duration)
	for time.Now().Before(testEnd) {
		if err := handleSender(stats, qos, conn, int64(testID)); err != nil {
			log.Printf("❌ Erreur dans handleSender : %v", err)
			_ = core.UpdateTestStatus(db, testID, false, true, false ,true )
			if ws != nil {
				sendTestStatus(ws, testID, "failed")
			}
			return nil, nil, err
		}
		time.Sleep(interval)
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

	// Étape 7 : Marquer le test comme terminé
	log.Println("🏁 Étape 7 : Marquage du test comme terminé...")
	if err := core.UpdateTestStatus(db, testID, false, false, true ,false); err != nil {
		log.Printf("⚠️ Erreur lors de la mise à jour du test terminé : %v", err)
	}
	if ws != nil {
		log.Println("📤 Envoi du statut 'finished' via WebSocket...")
		sendTestStatus(ws, testID, "finished")
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

func handleSender(Stats *PacketStats, qos *QoSMetrics, conn *net.UDPConn, testID int64) error {
	fmt.Println("🚀 handleSender : début")

	destAddr := &net.UDPAddr{
		IP:   net.ParseIP(Stats.TargetAddress),
		Port: Stats.TargetPort,
	}

	// 🏗️ Création du paquet TWAMP
	twampPacket := TwampTestPacket{
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

	// 🧵 Sérialisation
	serializedPacket, err := SerializeTwampTestPacket(&twampPacket)
	if err != nil {
		log.Printf("❌ Erreur de sérialisation: %v", err)
		return fmt.Errorf("erreur de sérialisation du paquet TWAMP: %w", err)
	}
	log.Printf("📦 Paquet sérialisé (%d octets), envoi vers %s:%d", len(serializedPacket), destAddr.IP, destAddr.Port)

	// 📤 Envoi du paquet
	_, err = conn.WriteToUDP(serializedPacket, destAddr)
	if err != nil {
		log.Printf("❌ Erreur d'envoi: %v", err)
		return fmt.Errorf("erreur d'envoi du paquet TWAMP: %w", err)
	}

	// 📥 Réception du paquet
	receivedData, err := receivePacket(conn)
	if err != nil {
		log.Printf("❌ Erreur de réception: %v", err)
		return fmt.Errorf("réception paquet échouée: %w", err)
	}
	Stats.TotalBytesReceived += int64(len(receivedData))

	var receivedPacket TwampTestPacket
	err = deserializeTwampTestPacket(receivedData, &receivedPacket)
	if err != nil {
		log.Printf("❌ Erreur de désérialisation: %v", err)
		return fmt.Errorf("erreur de désérialisation du paquet reçu: %w", err)
	}

	// 🕒 Timestamp de réception
	receivedPacket.ReceptionTimestamp = uint64(time.Now().UnixNano())

	// 📊 Calculs QoS
	latency := int64(receivedPacket.ReceptionTimestamp - receivedPacket.SenderTimestamp)
	Stats.LatencySamples = append(Stats.LatencySamples, latency)
	Stats.LastLatency = latency

	var jitterMs float64 = 0
	if len(Stats.LatencySamples) > 1 {
		prev := Stats.LatencySamples[len(Stats.LatencySamples)-2]
		jitter := abs(latency - prev)
		qos.TotalJitter += jitter
		jitterMs = float64(qos.TotalJitter) / float64(len(Stats.LatencySamples)-1) / 1e6
	}

	Stats.ReceivedPackets++
	Stats.SentPackets++

	latencyMs := float64(latency) / 1e6

	// 📡 Calcul du débit moyen en kbps
	elapsed := time.Since(Stats.StartTime).Seconds()
	throughputKbps := float64(Stats.TotalBytesReceived*8) / 1000 / elapsed

	// 🧾 Affichage des métriques
	fmt.Printf("✅ [Paquet %d] Latence: %.3f ms | Jitter: %.3f ms | Débit: %.3f kbps\n",
		Stats.SentPackets,
		latencyMs,
		jitterMs,
		throughputKbps)

	// 🛢️ Sauvegarde des résultats dans la base de données
	db, err := core.InitDB()
	if err != nil {
		log.Fatalf("❌ Impossible de se connecter à la base : %v", err)
	}
	defer db.Close()

	if err := core.SaveAttemptResult(db, testID, latencyMs, jitterMs, throughputKbps); err != nil {
		log.Printf("❌ Erreur insertion BDD: %v", err)
	}

	return nil
}

func handleReflector(conn *net.UDPConn, addr *net.UDPAddr, data []byte) error {
	var receivedPacket TwampTestPacket

	// 1. Désérialisation du paquet reçu
	err := deserializeTwampTestPacket(data, &receivedPacket)
	if err != nil {
		return fmt.Errorf("erreur de désérialisation: %v", err)
	}

	// 2. Ajout du timestamp de réception
	receivedPacket.ReceptionTimestamp = uint64(time.Now().UnixNano())

	// ✅ 3. Mise à jour du SenderTimestamp pour refléter l'instant du renvoi
	receivedPacket.SenderTimestamp = uint64(time.Now().UnixNano())

	// 3. Sérialisation du paquet modifié
	serializedPacket, err := SerializeTwampTestPacket(&receivedPacket)
	if err != nil {
		return fmt.Errorf("erreur de sérialisation: %v", err)
	}

	// ✅ Réponse avec la même connexion à l'adresse d'origine
	_, err = conn.WriteToUDP(serializedPacket, addr)
	if err != nil {
		return fmt.Errorf("échec de l'envoi de la réponse: %v", err)
	}
	log.Printf("✅ Paquet réponse envoyé à %s (%d octets)", addr.String(), len(serializedPacket))
	log.Printf("🎯 Paquet reçu: Sequence #%d", receivedPacket.SequenceNumber)
	log.Printf("📦 Renvoi du paquet vers %s", addr.String())

	return nil
}

// Fonction qui gère l'écoute sur le port Reflector (UDP)
func listenAsReflector() {
	// Adresse du serveur (Reflector)
	addr := net.UDPAddr{
		Port: AppConfig.Reflector.Port,
		IP:   net.ParseIP(AppConfig.Reflector.IP),
	}

	// Ouverture du socket UDP pour écouter les paquets
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Erreur écoute UDP: %v", err)
	}
	defer conn.Close()

	// Tampon pour recevoir les paquets
	buffer := make([]byte, 1500)

	log.Println("Reflector en écoute sur", addr.String())

	// Boucle pour écouter les paquets en continu
	for {
		// Lecture d'un paquet depuis la connexion UDP
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Erreur lecture paquet UDP: %v", err)
			continue // Si erreur, on passe à l'itération suivante
		}

		log.Printf("Reçu %d octets de %s", n, remoteAddr.String())

		// Lancer une goroutine pour gérer le paquet reçu (envoi de réponse ou traitement)
		go func(conn *net.UDPConn, data []byte, addr *net.UDPAddr) {
			err := handleReflector(conn, addr, data)
			if err != nil {
				log.Printf("Erreur traitement paquet dans handleReflector: %v", err)
			}
		}(conn, buffer[:n], remoteAddr)

	}
}



func Start(db *sql.DB) {

	log.Println("Démarrage de l'agent TWAMP...")

	LoadConfig("agent/config.yaml")

	// Mode Reflector TWAMP
	go listenAsReflector()

	// Serveur gRPC pour Quick Tests
	//go startGRPCServer()

	// Attente du démarrage des services
	time.Sleep(2 * time.Second)

	// WebSocket QoS
	go StartWebSocketAgent()


	// Connexion au backend gRPC (client stream)
	go startClientStream()

	// Blocage principal pour empêcher l'arrêt
	select {}
}
