package client

import (
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
)

const (
	CONNECTION_ATTEMPTS_MAX      = 3
	CONNECTION_ATTEMPTS_DELAY_MS = 2000
	DEFAULT_BATCH_SIZE           = 10
)

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile  string
	OutputFile string
	BatchSize  int
}

type Client struct {
	conn     net.Conn
	config   ClientConfig
	protocol *ClientProtocol
}

func NewClient(config ClientConfig) (*Client, error) {
	if config.BatchSize <= 0 {
		config.BatchSize = DEFAULT_BATCH_SIZE
	}

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
			time.Sleep(CONNECTION_ATTEMPTS_DELAY_MS * time.Millisecond)
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

	batchReader, err := NewBetBatchReader(client.config.InputFile, client.config.BatchSize)
	if err != nil {
		logger.Error("open-input-file", logger.Fail, "err", err)
		return err
	}
	defer batchReader.Close()

	batch := make([]Bet, 0, client.config.BatchSize)

	for {
		batch, err := batchReader.NextBatch(batch)
		if err != nil {
			logger.Error("read-batch", logger.Fail, "err", err)
			return err
		}
		if len(batch) == 0 {
			break
		}

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
			// si el error ocurrió debido a la cancelación por SIGTERM, es un apagado limpio
			return nil
		}
		logger.Error("receive-winners", logger.Fail, "err", err)
		return err
	}

	if err := SaveWinners(client.config.OutputFile, winners); err != nil {
		logger.Error("save-winners", logger.Fail, "err", err)
		return err
	}

	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)
	return nil
}
