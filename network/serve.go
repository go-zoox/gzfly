package network

import (
	"fmt"
	"net"

	"github.com/go-zoox/gzfly/network/tcp"
	"github.com/go-zoox/gzfly/network/udp"
)

type ServeConfig struct {
	Type   string
	Host   string
	Port   int
	OnConn func() (net.Conn, error)
}

func Serve(cfg *ServeConfig) error {
	switch cfg.Type {
	case "tcp":
		return tcp.Serve(&tcp.ServeConfig{
			Host:   cfg.Host,
			Port:   cfg.Port,
			OnConn: cfg.OnConn,
		})
	case "udp":
		return udp.Serve(&udp.ServeConfig{
			Host:   cfg.Host,
			Port:   cfg.Port,
			OnConn: cfg.OnConn,
		})
	default:
		return fmt.Errorf("network type(%s) not supported", cfg.Type)
	}
}
