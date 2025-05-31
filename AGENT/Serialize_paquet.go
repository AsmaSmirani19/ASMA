package agent

import(
	"fmt"
	"bytes"
	"encoding/binary"

)



const (
	PacketTypeSessionRequest = 0x01
	PacketTypeSessionAccept  = 0x02
	PacketTypeStartSession   = 0x03
	PacketTypeStartAck       = 0x04
	PacketTypeStopSession    = 0x05
)


func identifyPacketType(data []byte) string {
	if len(data) < 1 {
		return "Unknown"
	}

	switch data[0] {
	case PacketTypeSessionRequest:
		return "SessionRequest"
	case PacketTypeSessionAccept:
		return "SessionAccept"
	case PacketTypeStartSession:
		return "StartSession"
	case PacketTypeStartAck:
		return "StartAck"
	case PacketTypeStopSession:
		return "StopSession"
	default:
		return "Unknown"
	}
}
// structure des paquets
type SendSessionRequestPacket struct {
	Type          byte
	SenderAddress [16]byte
	//ReceiverPort  uint16
	//SenderPort    uint16
	PaddingLength uint32
	StartTime     uint32
	Timeout       uint32
	TypeP         uint8
}

type SessionAcceptPacket struct {
	Accept         uint8
	MBZ            uint8
	Port           uint16
	ReflectedOctet [16]byte
	ServerOctets   [16]byte
	SID            uint32
	HMAC           [16]byte
}

type StartSessionPacket struct {
	Type byte
	MBZ  uint8
	HMAC [16]byte
}

type StartAckPacket struct {
	Accept uint8
	MBZ    uint8
	HMAC   [16]byte
}

type StopSessionPacket struct {
	Type             byte
	Accept           uint8
	MBZ              uint8
	NumberOfSessions uint8
	HMAC             [16]byte
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
// Sérialisation des paquets
func SerializePacket(packet *SendSessionRequestPacket) ([]byte, error) {
	buf := new(bytes.Buffer)

	// 1. Écrire le champ Type comme premier octet
	err := binary.Write(buf, binary.BigEndian, packet.Type)
	if err != nil {
		return nil, err
	}

	// 2. Puis le reste du paquet
	err = binary.Write(buf, binary.BigEndian, packet.SenderAddress)
	if err != nil {
		return nil, err
	}

	fields := []interface{}{
		//packet.ReceiverPort,
		//packet.SenderPort,
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

func SerializeAcceptPacket(packet *SessionAcceptPacket) ([]byte, error) {
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

func SerializeStartPacket(packet *StartSessionPacket) ([]byte, error) {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		packet.Type,
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

func SerializeStartACKtPacket(packet *StartAckPacket) ([]byte, error) {
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

func SerializeStopSession(packet *StopSessionPacket) ([]byte, error) {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		packet.Type,
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