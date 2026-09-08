package client

import (
	"bufio"
	"os"
)

type BetBatchReader struct {
	file      *os.File
	scanner   *bufio.Scanner
	batchSize int
}

func NewBetBatchReader(filePath string, batchSize int) (*BetBatchReader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	return &BetBatchReader{
		file:      file,
		scanner:   bufio.NewScanner(file),
		batchSize: batchSize,
	}, nil
}

func (r *BetBatchReader) NextBatch(batch []Bet) ([]Bet, error) {
	batch = batch[:0]

	for len(batch) < r.batchSize && r.scanner.Scan() {
		bet, err := ParseBetCSV(r.scanner.Bytes())
		if err != nil {
			return nil, err
		}
		batch = append(batch, bet)
	}

	if err := r.scanner.Err(); err != nil {
		return nil, err
	}

	return batch, nil
}

func (r *BetBatchReader) Close() error {
	return r.file.Close()
}

func SaveWinners(filePath string, winners []Bet) error {
	outputFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	for _, winner := range winners {
		if _, err := writer.WriteString(winner.ToCSV() + "\n"); err != nil {
			return err
		}
	}

	return writer.Flush()
}
