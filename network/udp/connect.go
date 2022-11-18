package udp

import (
	"fmt"
	"net"

	"github.com/go-zoox/gzfly/network/utils"
	"github.com/go-zoox/logger"
)

type ConnectTarget struct {
	Host string
	Port int
	//
	ID string
}

func Connect(source net.Conn, cfg *ConnectTarget) error {
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	logger.Infof("[connection:udp][%s] connect to: %s", cfg.ID, addr)

	conn, err := net.Dial("udp", addr)
	if err != nil {
		return err
	}

	go utils.Copy(source, conn)
	go utils.Copy(conn, source)

	return nil
}
