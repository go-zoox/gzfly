package tcp

import (
	"fmt"
	"net"

	"github.com/go-zoox/logger"
)

type CreateTCPConnectionConfig struct {
	Host string
	Port int
	//
	ID   string
	Conn net.Conn
}

func CreateTCPConnection(cfg *CreateTCPConnectionConfig) error {
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	logger.Infof("[connection:tcp][%s] connect to: %s", cfg.ID, addr)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go Copy(cfg.Conn, conn)
	go Copy(conn, cfg.Conn)

	return nil
}

func CloseTCPConnection(conn net.Conn) error {
	return conn.Close()
}
