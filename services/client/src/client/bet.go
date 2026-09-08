package client

import (
	"bytes"
	"errors"
	"strconv"
)

const (
	NameMaxLength     = 255
	LastNameMaxLength = 255
)

type Bet struct {
	FirstName string
	LastName  string
	Document  uint32
	Birthdate string
	Number    uint32
}

func ParseBetCSV(line []byte) (Bet, error) {
	parts := bytes.SplitN(line, []byte(","), 5)
	if len(parts) < 5 {
		return Bet{}, errors.New("invalid bet format")
	}

	firstName := string(parts[0])
	lastName := string(parts[1])

	if len(firstName) > NameMaxLength || len(lastName) > LastNameMaxLength {
		return Bet{}, errors.New("name too long")
	}

	docNum, err := strconv.ParseUint(string(parts[2]), 10, 32)
	if err != nil {
		return Bet{}, err
	}

	birthdate := string(parts[3])

	betNum, err := strconv.ParseUint(string(parts[4]), 10, 32)
	if err != nil {
		return Bet{}, err
	}

	return Bet{
		FirstName: firstName,
		LastName:  lastName,
		Document:  uint32(docNum),
		Birthdate: birthdate,
		Number:    uint32(betNum),
	}, nil
}

func (b Bet) ToCSV() string {
	return b.FirstName + "," + b.LastName + "," + strconv.FormatUint(uint64(b.Document), 10) + "," + b.Birthdate + "," + strconv.FormatUint(uint64(b.Number), 10)
}
