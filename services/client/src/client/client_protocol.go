package client

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const (
	AGENCY_ID_SIZE        = 2
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

func (cp *ClientProtocol) SendBet(message []byte) error {
	parts := bytes.Split(message, []byte(","))
	if len(parts) < 5 {
		logger.Error("send-bet", logger.Fail, "error", "invalid bet format")
		return errors.New("invalid bet format")
	}

	firstName := parts[0]
	lastName := parts[1]
	document := parts[2]
	birthdate := parts[3]
	number := parts[4]

	if len(firstName) > NAME_MAX_LENGTH || len(lastName) > LAST_NAME_MAX_LENGTH {
		logger.Error("send-bet", logger.Fail, "error", "name too long")
		return errors.New("name too long")
	}

	docNum, err := strconv.Atoi(string(document))
	if err != nil {
		logger.Error("send-bet", logger.Fail, "error", "invalid document format")
		return err
	}
	betNum, err := strconv.Atoi(string(number))
	if err != nil {
		logger.Error("send-bet", logger.Fail, "error", "invalid bet number")
		return err
	}
	agencyIdNum, err := strconv.Atoi(cp.AgencyId)
	if err != nil {
		logger.Error("send-bet", logger.Fail, "error", "invalid agency ID")
		return err
	}

	// agency_id(2) + len_name(1) + name + len_last(1) + last + doc(4) + birth(10) + bet(4)
	totalSize := AGENCY_ID_SIZE + NAME_LENGTH_SIZE + len(firstName) + LAST_NAME_LENGTH_SIZE + len(lastName) + DOCUMENT_SIZE + BIRTHDATE_SIZE + NUMBER_SIZE
	buffer := make([]byte, totalSize)

	cursor := 0

	// agencia (2 bytes)
	binary.BigEndian.PutUint16(buffer[cursor:], uint16(agencyIdNum))
	cursor += AGENCY_ID_SIZE

	// nombre (1 byte len + payload)
	buffer[cursor] = byte(len(firstName))
	cursor += NAME_LENGTH_SIZE
	cursor += copy(buffer[cursor:], firstName)

	// apellido (1 byte len + payload)
	buffer[cursor] = byte(len(lastName))
	cursor += LAST_NAME_LENGTH_SIZE
	cursor += copy(buffer[cursor:], lastName)

	// documento (4 bytes binarios uint32)
	binary.BigEndian.PutUint32(buffer[cursor:], uint32(docNum))
	cursor += DOCUMENT_SIZE

	// fecha de nacimiento (10 bytes texto)
	cursor += copy(buffer[cursor:], birthdate)

	// apuesta (4 bytes binarios uint32)
	binary.BigEndian.PutUint32(buffer[cursor:], uint32(betNum))

	if err := safe_socket.SendAll(cp.conn, buffer); err != nil {
		logger.Error("send-bet", logger.Fail, "error", "failed to send bet")
		return err
	}

	return nil
}

// mando nombre de tamanio 0 para indicar que no hay mas
func (cp *ClientProtocol) SendEnd() error {
	agencyIdNum, err := strconv.Atoi(cp.AgencyId)
	if err != nil {
		logger.Error("send-end", logger.Fail, "error", "invalid agency ID")
		return err
	}

	buffer := make([]byte, AGENCY_ID_SIZE+1)
	binary.BigEndian.PutUint16(buffer[0:], uint16(agencyIdNum))
	buffer[2] = 0 // nombre de tamanio 0 para indicar que no hay mas

	if err := safe_socket.SendAll(cp.conn, buffer); err != nil {
		logger.Error("send-end", logger.Fail, "error", "failed to send end")
		return err
	}
	return nil
}
func (cp *ClientProtocol) ReceiveWinners() ([]string, error) {
	
	// cantidad de ganadores (4 bytes)
	amountBuffer, err := safe_socket.RecvAll(cp.conn, 4)
	if err != nil {
		logger.Error("receive-winners", logger.Fail, "error", "failed to receive winners count")
		return nil, err
	}

	winnersCount := binary.BigEndian.Uint32(amountBuffer)
	winners := make([]string, 0, winnersCount)

	for i := uint32(0); i < winnersCount; i++ {
		
		// nombre
		nameLenBuf, err := safe_socket.RecvAll(cp.conn, NAME_LENGTH_SIZE)
		if err != nil {
			logger.Error("receive-winners", logger.Fail, "error", "failed to receive winner name length")
			return nil, err
		}
		nameLen := int(nameLenBuf[0])

		nameBuf, err := safe_socket.RecvAll(cp.conn, nameLen)
		if err != nil {
			logger.Error("receive-winners", logger.Fail, "error", "failed to receive winner name")
			return nil, err
		}

		// apellido
		lastNameLenBuf, err := safe_socket.RecvAll(cp.conn, LAST_NAME_LENGTH_SIZE)
		if err != nil {
			logger.Error("receive-winners", logger.Fail, "error", "failed to receive winner last name length")
			return nil, err
		}
		lastNameLen := int(lastNameLenBuf[0])

		lastNameBuf, err := safe_socket.RecvAll(cp.conn, lastNameLen)
		if err != nil {
			logger.Error("receive-winners", logger.Fail, "error", "failed to receive winner last name")
			return nil, err
		}

		// documento (4)
		docBuf, err := safe_socket.RecvAll(cp.conn, DOCUMENT_SIZE)
		if err != nil {
			logger.Error("receive-winners", logger.Fail, "error", "failed to receive winner document")
			return nil, err
		}
		doc := binary.BigEndian.Uint32(docBuf)

		// fecha de nacimiento (10)
		birthBuf, err := safe_socket.RecvAll(cp.conn, BIRTHDATE_SIZE)
		if err != nil {
			logger.Error("receive-winners", logger.Fail, "error", "failed to receive winner birth date")
			return nil, err
		}

		// apuesta (4)
		numBuf, err := safe_socket.RecvAll(cp.conn, NUMBER_SIZE)
		if err != nil {
			logger.Error("receive-winners", logger.Fail, "error", "failed to receive winner number")
			return nil, err
		}
		num := binary.BigEndian.Uint32(numBuf)

		// formatear línea resultado para guardar en OUTPUT_FILE
		winnerLine := fmt.Sprintf("%s,%s,%d,%s,%d", string(nameBuf), string(lastNameBuf), doc, string(birthBuf), num)
		winners = append(winners, winnerLine)
	}

	return winners, nil
}