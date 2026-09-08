package client

import (
	"encoding/binary"
	"errors"
	"net"
	"strconv"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const (
	AGENCY_ID_SIZE        = 2
	BATCH_COUNT_SIZE      = 2
	NAME_LENGTH_SIZE      = 1
	LAST_NAME_LENGTH_SIZE = 1
	DOCUMENT_SIZE         = 4
	BIRTHDATE_SIZE        = 10
	NUMBER_SIZE           = 4
)

type ClientProtocol struct {
	conn     net.Conn
	AgencyId string
}

func NewClientProtocol(conn net.Conn, agencyId string) *ClientProtocol {
	return &ClientProtocol{conn: conn, AgencyId: agencyId}
}

func (cp *ClientProtocol) SendBatch(bets []Bet) error {
	agencyIdNum, err := strconv.Atoi(cp.AgencyId)
	if err != nil {
		logger.Warn("send-batch", logger.Fail, "error", "invalid agency ID")
		return err
	}

	totalSize := AGENCY_ID_SIZE + BATCH_COUNT_SIZE
	for _, bet := range bets {
		totalSize += NAME_LENGTH_SIZE + len(bet.FirstName) +
			LAST_NAME_LENGTH_SIZE + len(bet.LastName) +
			DOCUMENT_SIZE + BIRTHDATE_SIZE + NUMBER_SIZE
	}

	payload := make([]byte, AGENCY_ID_SIZE+BATCH_COUNT_SIZE, totalSize)
	binary.BigEndian.PutUint16(payload[0:], uint16(agencyIdNum))
	binary.BigEndian.PutUint16(payload[2:], uint16(len(bets)))

	for _, bet := range bets {
		payload = encodeBet(payload, bet)
	}

	// enviamos el batch completo por la red
	if err := safe_socket.SendAll(cp.conn, payload); err != nil {
		logger.Warn("send-batch", logger.Fail, "error", "failed to send batch")
		return err
	}

	// espero el ACK
	ackBuf, err := safe_socket.RecvAll(cp.conn, 1)
	if err != nil || ackBuf[0] != 0 {
		errAck := errors.New("batch submission not acknowledged")
		logger.Warn("send-batch", logger.Fail, "error", errAck.Error())
		return errAck
	}

	return nil
}

// mando un batch con cantidad 0 de apuestas para señalar el fin de la transmisión de apuestas
func (cp *ClientProtocol) SendEnd() error {
	agencyIdNum, err := strconv.Atoi(cp.AgencyId)
	if err != nil {
		return err
	}

	buffer := make([]byte, AGENCY_ID_SIZE+BATCH_COUNT_SIZE)
	binary.BigEndian.PutUint16(buffer[0:], uint16(agencyIdNum))
	binary.BigEndian.PutUint16(buffer[2:], 0) // cantidad 0 = FIN

	if err := safe_socket.SendAll(cp.conn, buffer); err != nil {
		logger.Warn("send-end", logger.Fail, "error", "failed to send end")
		return err
	}

	// espero el ACK del fin
	_, err = safe_socket.RecvAll(cp.conn, 1)
	return err
}

func (cp *ClientProtocol) ReceiveWinners() ([]Bet, error) {
	// cantidad de ganadores (4 bytes)
	amountBuffer, err := safe_socket.RecvAll(cp.conn, 4)
	if err != nil {
		logger.Warn("receive-winners", logger.Fail, "error", "failed to receive winners count")
		return nil, err
	}

	winnersCount := binary.BigEndian.Uint32(amountBuffer)
	winners := make([]Bet, 0, winnersCount)

	for i := uint32(0); i < winnersCount; i++ {
		winner, err := decodeBet(cp.conn)
		if err != nil {
			logger.Warn("receive-winners", logger.Fail, "error", "failed to receive winner")
			return nil, err
		}
		winners = append(winners, winner)
	}

	return winners, nil
}

func encodeBet(payload []byte, bet Bet) []byte {
	firstNameBytes := []byte(bet.FirstName)
	lastNameBytes := []byte(bet.LastName)
	birthdateBytes := []byte(bet.Birthdate)

	payload = append(payload, byte(len(firstNameBytes)))
	payload = append(payload, firstNameBytes...)
	payload = append(payload, byte(len(lastNameBytes)))
	payload = append(payload, lastNameBytes...)
	payload = payload[:len(payload)+DOCUMENT_SIZE]
	binary.BigEndian.PutUint32(payload[len(payload)-DOCUMENT_SIZE:], bet.Document)
	payload = append(payload, birthdateBytes...)
	payload = payload[:len(payload)+NUMBER_SIZE]
	binary.BigEndian.PutUint32(payload[len(payload)-NUMBER_SIZE:], bet.Number)

	return payload
}

func decodeBet(conn net.Conn) (Bet, error) {
	// nombre
	nameLenBuf, err := safe_socket.RecvAll(conn, NAME_LENGTH_SIZE)
	if err != nil {
		return Bet{}, err
	}
	nameBuf, err := safe_socket.RecvAll(conn, int(nameLenBuf[0]))
	if err != nil {
		return Bet{}, err
	}

	// apellido
	lastNameLenBuf, err := safe_socket.RecvAll(conn, LAST_NAME_LENGTH_SIZE)
	if err != nil {
		return Bet{}, err
	}
	lastNameBuf, err := safe_socket.RecvAll(conn, int(lastNameLenBuf[0]))
	if err != nil {
		return Bet{}, err
	}

	// campos fijos: documento (4) + fecha nacimiento (10) + numero (4) = 18 bytes
	fixedBuf, err := safe_socket.RecvAll(conn, DOCUMENT_SIZE+BIRTHDATE_SIZE+NUMBER_SIZE)
	if err != nil {
		return Bet{}, err
	}
	doc := binary.BigEndian.Uint32(fixedBuf[:DOCUMENT_SIZE])
	birthBuf := fixedBuf[DOCUMENT_SIZE : DOCUMENT_SIZE+BIRTHDATE_SIZE]
	num := binary.BigEndian.Uint32(fixedBuf[DOCUMENT_SIZE+BIRTHDATE_SIZE:])

	return Bet{
		FirstName: string(nameBuf),
		LastName:  string(lastNameBuf),
		Document:  doc,
		Birthdate: string(birthBuf),
		Number:    num,
	}, nil
}
