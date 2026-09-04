package client

import (
	"bufio"
	"net"
	"os"
	"strconv"
	"time"
	"syscall"
	"os/signal"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
)

const CONNECTION_ATTEMPTS_MAX = 3
const CONNECTION_ATTEMPS_DELAY_MS = 2000

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile  string
	OutputFile string
}

type Client struct {
	conn   net.Conn
	config ClientConfig
	protocol *ClientProtocol
}

func NewClient(config ClientConfig) (*Client, error) {
	conn, err := connectToServer(config.ServerHost, config.ServerPort)
	if err != nil {
		logger.Warn("connect-to-server", logger.Fail)
		return nil, err
	}

	protocol := NewClientProtocol(conn, config.AgencyId)

	client := &Client{conn: conn, config: config, protocol: protocol}
	return client, nil
}

func connectToServer(host, port string) (net.Conn, error) {
	const action = "connect-to-server"
	var err error
	var conn net.Conn

	logger.Info(action, logger.InProgress)
	for i := range CONNECTION_ATTEMPTS_MAX {
		conn, err = net.Dial("tcp", host+":"+port)
		if err != nil {
			logger.Warn(action, logger.Fail, "attempt", i)
			time.Sleep(CONNECTION_ATTEMPS_DELAY_MS * time.Millisecond)
			continue
		}

		logger.Info(action, logger.Success)
		break
	}

	return conn, err
}

func (client *Client) Run() error {
	const mainAction = "read-and-send-bets"
	var isShuttingDown bool
	defer client.conn.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
        <-sigChan
        isShuttingDown = true
        logger.Info("client-shutdown", logger.Success, "reason", "SIGTERM received")
        client.conn.Close()
    }()

	batchSizeStr := os.Getenv("BATCH_SIZE")
	batchSize, err := strconv.Atoi(batchSizeStr)
	if err != nil || batchSize <= 0 {
		batchSize = 10 
	}

	inputFile, err := os.Open(client.config.InputFile)
	if err != nil {
		logger.Error("open-input-file", logger.Fail, "err", err)
		return err
	}
	defer inputFile.Close()

	outputFile, err := os.Create(client.config.OutputFile)
	if err != nil {
		logger.Error("create-output-file", logger.Fail, "err", err)
		return err
	}
	defer outputFile.Close()

	scanner := bufio.NewScanner(inputFile)
	batch := make([][]byte, 0, batchSize)

	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		batch = append(batch, line)

		if len(batch) == batchSize {
			if err := client.protocol.SendBatch(batch); err != nil {
				logger.Error("send-batch", logger.Fail, "err", err)
				return err
			}
			batch = batch[:0] 
		}
	}

	// enviamos si las lineas que quedaron en el batch
	if len(batch) > 0 {
		if err := client.protocol.SendBatch(batch); err != nil {
			logger.Error("send-batch", logger.Fail, "err", err)
			return err
		}
	}

	if err := client.protocol.SendEnd(); err != nil {
		logger.Error("send-end", logger.Fail, "err", err)
		return err
	}

	winners, err := client.protocol.ReceiveWinners()
	if err != nil {

		if isShuttingDown {
            // Si el error ocurrió debido a la cancelación por SIGTERM, es un apagado limpio
            return nil
        }
		
		logger.Error("receive-winners", logger.Fail, "err", err)
		return err
	}

	writer := bufio.NewWriter(outputFile)
	for _, winner := range winners {
		if _, err := writer.WriteString(winner + "\n"); err != nil {
			logger.Error("write-output-file", logger.Fail, "err", err)
			return err
		}
	}

	writer.Flush()

	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)
	return nil
}
