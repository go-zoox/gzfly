package network

import (
	"fmt"
	"net"

	"github.com/go-zoox/gzfly/network/tcp"
	"github.com/go-zoox/gzfly/network/udp"
)

type ConnectTarget struct {
	Type string
	Host string
	Port int
	//
	ID string
}

func Connect(source net.Conn, cfg *ConnectTarget) error {
	switch cfg.Type {
	case "tcp":
		return tcp.Connect(source, &tcp.ConnectTarget{
			Host: cfg.Host,
			Port: cfg.Port,
			ID:   cfg.ID,
		})
	case "udp":
		return udp.Connect(source, &udp.ConnectTarget{
			Host: cfg.Host,
			Port: cfg.Port,
			ID:   cfg.ID,
		})
	default:
		return fmt.Errorf("network type(%s) not supported", cfg.Type)
	}
}
