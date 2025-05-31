package agent

import(
	"log"
	"time"
	"net"
	"fmt"
	//"database/sql"
	"strconv"
	
	//"github.com/gorilla/websocket"

)


func SendTCPPacket(packet []byte, addr string, port int) error {

	timeout := 5 * time.Second

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("[%s]:%d", addr, port), timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	if err != nil {
		return fmt.Errorf("échec d'envoi du paquet: %w", err)
	}
	return nil
}


func Client(config TestConfig) error {
	log.Println("🔵 [Client] Début d'exécution du client...")

	serverAddress := AppConfig.Network.ServerAddress
	serverPort := AppConfig.Network.ServerPort
	//senderPort := AppConfig.Network.SenderPort
	//receiverPort := AppConfig.Network.ReceiverPort
	timeout := AppConfig.Network.Timeout

	connStr := net.JoinHostPort(serverAddress, strconv.Itoa(serverPort))
	log.Printf("🔌 [Client] Connexion à %s ...", connStr)
	conn, err := net.DialTimeout("tcp", connStr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("[Client] Erreur de connexion au serveur TCP : %w", err)
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			log.Printf("[Client] Erreur fermeture connexion TCP : %v", cerr)
		}
	}()

	log.Println("✅ [Client] Connexion TCP établie.")

	// 1. Envoi Session-Request
	log.Println("⚙️ [Client] Construction du paquet Session-Request...")
	ip := net.ParseIP(AppConfig.Sender.IP).To16()
	if ip == nil {
		return fmt.Errorf("[Client] IP source invalide : %s", AppConfig.Sender.IP)
	}
	var senderIP [16]byte
	copy(senderIP[:], ip)

	packet := SendSessionRequestPacket{
		Type:          PacketTypeSessionRequest,
		SenderAddress: senderIP,
		//ReceiverPort:  uint16(receiverPort),
		//SenderPort:    uint16(senderPort),
		PaddingLength: 0,
		StartTime:     uint32(time.Now().Unix()),
		Timeout:       uint32(timeout.Seconds()),
		TypeP:         0x05,
	}

	log.Println("📤 [Client] Envoi Session-Request...")
	serializedPacket, err := SerializePacket(&packet)
	if err != nil {
		return fmt.Errorf("[Client] Erreur de sérialisation Session-Request : %w", err)
	}

	if _, err := conn.Write(serializedPacket); err != nil {
		return fmt.Errorf("[Client] Erreur d'envoi du Session-Request : %w", err)
	}

	// 2. Lire Accept-Session
	log.Println("📥 [Client] Attente de l'Accept-Session...")
	acceptSize := 32 
	acceptBuffer, err := readFullPacket(conn, acceptSize)
	if err != nil {
		return fmt.Errorf("[Client] Erreur de lecture (Accept-Session) : %w", err)
	}
	log.Printf("📥 [Client] Accept-Session reçu (%d octets) : %x", len(acceptBuffer), acceptBuffer)

	// 3. Envoi Start-Session
	startSessionPacket := StartSessionPacket{
		Type: PacketTypeStartSession,
		MBZ:  0,
		HMAC: [16]byte{},
	}

	log.Println("📤 [Client] Envoi Start-Session...")
	serializedStart, err := SerializeStartPacket(&startSessionPacket)
	if err != nil {
		return fmt.Errorf("[Client] Erreur de sérialisation Start-Session : %w", err)
	}
	log.Printf("📦 [Client] Paquet Start-Session (hex) : %x", serializedStart)

	if _, err := conn.Write(serializedStart); err != nil {
		return fmt.Errorf("[Client] Erreur d'envoi Start-Session : %w", err)
	}

	// 4. Lire Start-Ack
	log.Println("📥 [Client] Attente du Start-Ack...")
	startAckSize := 32 // taille supposée du Start-Ack, adapte selon protocole
	startAckBuffer, err := readFullPacket(conn, startAckSize)
	if err != nil {
		return fmt.Errorf("[Client] Erreur de lecture (Start-Ack) : %w", err)
	}
	log.Printf("📥 [Client] Start-Ack reçu (%d octets) : %x", len(startAckBuffer), startAckBuffer)

	// 5. Connexion WebSocket (avant de démarrer le test)
	log.Println("🌐 Tentative de connexion WebSocket pour le test...")
	wsConn, err := StartWebSocketAgent()
	if err != nil {
		log.Printf("❌ [Client] Échec de la connexion WebSocket : %v", err)
	} else {
		log.Println("✅ [Client] Connexion WebSocket établie.")
	}
	
	defer func() {
	if wsConn != nil {
		if err := wsConn.Close(); err != nil {
			log.Printf("⚠️ Erreur à la fermeture de WebSocket : %v", err)
		} else {
			log.Println("🔌 Connexion WebSocket fermée.")
		}
	}
}()

// 6. Démarrage du test QoS avec WebSocket active
	log.Printf("🔍 DEBUG TestConfig utilisé : %+v", config)
	log.Println("🚀 [Client] Lancement du test via StartTest()...")
	stats, qos, err := StartTest( config, wsConn)
	if err != nil {
		log.Printf("❌ [Client] Erreur lors du test : %v", err)
	} else {
		log.Printf("✅ [Client] Test terminé. Stats : %+v | QoS : %+v", stats, qos)
	}


	// 6. Envoi Stop-Session
	log.Println("📤 [Client] Envoi Stop-Session...")
	stopSessionPacket := StopSessionPacket{
		Type:             PacketTypeStopSession,
		Accept:           0,
		MBZ:              0,
		NumberOfSessions: 1,
		HMAC:             [16]byte{},
	}
	serializedStop, err := SerializeStopSession(&stopSessionPacket)
	if err != nil {
		return fmt.Errorf("[Client] Erreur de sérialisation Stop-Session : %w", err)
	}
	log.Printf("📦 [Client] Paquet Stop-Session (hex) : %x", serializedStop)

	if _, err := conn.Write(serializedStop); err != nil {
		return fmt.Errorf("[Client] Erreur d'envoi Stop-Session : %w", err)
	}


	log.Println("🏁 [Client] Exécution terminée avec succès.")
	return nil
}

// readFullPacket lit exactement 'size' octets depuis la connexion TCP, ou retourne une erreur.
func readFullPacket(conn net.Conn, size int) ([]byte, error) {
	buffer := make([]byte, size)
	totalRead := 0
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	for totalRead < size {
		n, err := conn.Read(buffer[totalRead:])
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, fmt.Errorf("connexion fermée avant la lecture complète")
		}
		totalRead += n
	}
	return buffer, nil
}
