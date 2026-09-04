package client

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"errors"
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

	NAME_MAX_LENGTH      = 255
	LAST_NAME_MAX_LENGTH = 255
)

type ClientProtocol struct {
	conn     net.Conn
	AgencyId string
}

func NewClientProtocol(conn net.Conn, agencyId string) *ClientProtocol {
	return &ClientProtocol{conn: conn, AgencyId: agencyId}
}

func (cp *ClientProtocol) SendBatch(lines [][]byte) error {
	 
	type parsedBet struct {
        firstName, lastName, birthdate []byte
        docNum, betNum                 uint32
    }

	agencyIdNum, err := strconv.Atoi(cp.AgencyId)
	if err != nil {
		logger.Warn("send-batch", logger.Fail, "error", "invalid agency ID")
		return err
	}
	
    bets := make([]parsedBet, 0, len(lines))
    totalSize := AGENCY_ID_SIZE + BATCH_COUNT_SIZE
    for _, line := range lines {
        parts := bytes.SplitN(line, []byte(","), 5)
        if len(parts) < 5 {
            return errors.New("invalid bet format")
        }
        docNum, err := strconv.Atoi(string(parts[2]))
        if err != nil { return err }
        betNum, err := strconv.Atoi(string(parts[4]))
        if err != nil { return err }
        bet := parsedBet{
            firstName: parts[0], lastName: parts[1],
            birthdate: parts[3],
            docNum: uint32(docNum), betNum: uint32(betNum),
        }
        if len(bet.firstName) > NAME_MAX_LENGTH || len(bet.lastName) > LAST_NAME_MAX_LENGTH {
            return errors.New("name too long")
        }
        totalSize += NAME_LENGTH_SIZE + len(bet.firstName) +
                     LAST_NAME_LENGTH_SIZE + len(bet.lastName) +
                     DOCUMENT_SIZE + BIRTHDATE_SIZE + NUMBER_SIZE
        bets = append(bets, bet)
    }

	
    payload := make([]byte, AGENCY_ID_SIZE+BATCH_COUNT_SIZE, totalSize)
    binary.BigEndian.PutUint16(payload[0:], uint16(agencyIdNum))
    binary.BigEndian.PutUint16(payload[2:], uint16(len(bets)))
    for _, bet := range bets {
        payload = append(payload, byte(len(bet.firstName)))
        payload = append(payload, bet.firstName...)
        payload = append(payload, byte(len(bet.lastName)))
        payload = append(payload, bet.lastName...)
        payload = payload[:len(payload)+DOCUMENT_SIZE]
        binary.BigEndian.PutUint32(payload[len(payload)-DOCUMENT_SIZE:], bet.docNum)
        payload = append(payload, bet.birthdate...)
        payload = payload[:len(payload)+NUMBER_SIZE]
        binary.BigEndian.PutUint32(payload[len(payload)-NUMBER_SIZE:], bet.betNum)
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

func (cp *ClientProtocol) ReceiveWinners() ([]string, error) {
	
	// cantidad de ganadores (4 bytes)
	amountBuffer, err := safe_socket.RecvAll(cp.conn, 4)
	if err != nil {
		logger.Warn("receive-winners", logger.Fail, "error", "failed to receive winners count")
		return nil, err
	}

	winnersCount := binary.BigEndian.Uint32(amountBuffer)
	winners := make([]string, 0, winnersCount)

	for i := uint32(0); i < winnersCount; i++ {
		
		// nombre
		nameLenBuf, err := safe_socket.RecvAll(cp.conn, NAME_LENGTH_SIZE)
		if err != nil {
			logger.Warn("receive-winners", logger.Fail, "error", "failed to receive winner name length")
			return nil, err
		}
		nameLen := int(nameLenBuf[0])

		nameBuf, err := safe_socket.RecvAll(cp.conn, nameLen)
		if err != nil {
			logger.Warn("receive-winners", logger.Fail, "error", "failed to receive winner name")
			return nil, err
		}

		// apellido
		lastNameLenBuf, err := safe_socket.RecvAll(cp.conn, LAST_NAME_LENGTH_SIZE)
		if err != nil {
			logger.Warn("receive-winners", logger.Fail, "error", "failed to receive winner last name length")
			return nil, err
		}
		lastNameLen := int(lastNameLenBuf[0])

		lastNameBuf, err := safe_socket.RecvAll(cp.conn, lastNameLen)
		if err != nil {
			logger.Warn("receive-winners", logger.Fail, "error", "failed to receive winner last name")
			return nil, err
		}

		// documento (4)
		docBuf, err := safe_socket.RecvAll(cp.conn, DOCUMENT_SIZE)
		if err != nil {
			logger.Warn("receive-winners", logger.Fail, "error", "failed to receive winner document")
			return nil, err
		}
		doc := binary.BigEndian.Uint32(docBuf)

		// fecha de nacimiento (10)
		birthBuf, err := safe_socket.RecvAll(cp.conn, BIRTHDATE_SIZE)
		if err != nil {
			logger.Warn("receive-winners", logger.Fail, "error", "failed to receive winner birth date")
			return nil, err
		}

		// apuesta (4)
		numBuf, err := safe_socket.RecvAll(cp.conn, NUMBER_SIZE)
		if err != nil {
			logger.Warn("receive-winners", logger.Fail, "error", "failed to receive winner number")
			return nil, err
		}
		num := binary.BigEndian.Uint32(numBuf)

		// formatear línea resultado para guardar en OUTPUT_FILE
		winnerLine := string(nameBuf) + "," + string(lastNameBuf) + "," + strconv.FormatUint(uint64(doc), 10) + "," + string(birthBuf) + "," + strconv.FormatUint(uint64(num), 10)
		winners = append(winners, winnerLine)
	}

	return winners, nil
}