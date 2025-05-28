package agent


import (
	"fmt"
	"io"
	"log"
	"net"
	"time"	
)


type TestResult struct {
	AgentID    string
	Timestamp  time.Time
	Latency    float64
	Loss       float64
	Throughput float64
}

func Serveur() {
	log.Println("🟢 [Serveur] Démarrage du serveur...")

	address := fmt.Sprintf(":%d", AppConfig.Network.ServerPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("❌ [Serveur] Échec du démarrage sur %s : %v", address, err)
	}
	defer func() {
		log.Println("🔒 [Serveur] Fermeture du listener TCP.")
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("❌ [Serveur] Erreur lors de l'acceptation d'une connexion : %v", err)
			continue
		}

		log.Printf("🔗 [Serveur] Connexion acceptée depuis %s", conn.RemoteAddr())

		go func(conn net.Conn) {
			defer func() {
				log.Printf("🔌 [Serveur] Fermeture de la connexion avec %s", conn.RemoteAddr())
				conn.Close()
			}()

			buf := make([]byte, 1024)

			for {
				n, err := conn.Read(buf)
				if err != nil {
					if err == io.EOF {
						log.Printf("📴 [Serveur] Connexion fermée proprement par %s", conn.RemoteAddr())
					} else {
						log.Printf("❌ [Serveur] Erreur de lecture sur %s : %v", conn.RemoteAddr(), err)
					}
					return
				}

				if n == 0 {
					log.Printf("⚠️ [Serveur] Paquet vide reçu depuis %s", conn.RemoteAddr())
					continue
				}

				data := buf[:n]
				log.Printf("📦 [Serveur] Paquet brut reçu (%d octets) : %x", n, data)

				packetType := identifyPacketType(data)
				log.Printf("🔍 [Serveur] Type de paquet identifié : %s", packetType)

				switch packetType {
				case "SessionRequest":
					log.Println("✅ [Serveur] Traitement du paquet SessionRequest...")
					acceptSessionPacket := SessionAcceptPacket{
						Accept: 0,
						MBZ:    0,
						HMAC:   [16]byte{},
					}
					serializedPacket, err := SerializeAcceptPacket(&acceptSessionPacket)
					if err != nil {
						log.Printf("❌ [Serveur] Erreur de sérialisation du paquet Accept-Session : %v", err)
						return
					}
					_, err = conn.Write(serializedPacket)
					if err != nil {
						log.Printf("❌ [Serveur] Erreur lors de l'envoi d'Accept-Session : %v", err)
						return
					}
					log.Printf("📤 [Serveur] Accept-Session envoyé à %s : %x", conn.RemoteAddr(), serializedPacket)

				case "StartSession":
					log.Println("✅ [Serveur] Traitement du paquet StartSession...")
					startAckPacket := StartAckPacket{
						Accept: 0,
						MBZ:    0,
						HMAC:   [16]byte{},
					}
					serializedStartAckPacket, err := SerializeStartACKtPacket(&startAckPacket)
					if err != nil {
						log.Printf("❌ [Serveur] Erreur de sérialisation du paquet Start-Ack : %v", err)
						return
					}
					_, err = conn.Write(serializedStartAckPacket)
					if err != nil {
						log.Printf("❌ [Serveur] Erreur lors de l'envoi de Start-Ack : %v", err)
						return
					}
					log.Printf("📤 [Serveur] Start-Ack envoyé à %s : %x", conn.RemoteAddr(), serializedStartAckPacket)

				case "StopSession":
					log.Println("✅ [Serveur] Paquet StopSession reçu, fermeture de la session.")
					return

				default:
					log.Printf("⚠️ [Serveur] Type de paquet inconnu ou non pris en charge. Données brutes : %x", data)
				}
			}
		}(conn)
	}
}
