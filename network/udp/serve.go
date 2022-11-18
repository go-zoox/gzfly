package udp

import (
	"fmt"
	"net"

	"github.com/go-zoox/logger"

	"github.com/go-zoox/gzfly/network/utils"
)

type ServeConfig struct {
	Host   string
	Port   int
	OnConn func() (net.Conn, error)
}

func Serve(cfg *ServeConfig) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	logger.Info("listen udp server at: %s", addr)
	s, err := net.ResolveUDPAddr("udp4", addr)
	listener, err := net.ListenUDP("udp4", s)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		source, err := acceptUDP(listener) // listener.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		logger.Info("[udp] client connected")

		go func() {
			target, err := cfg.OnConn()
			if err != nil {
				logger.Warn("[warning] failed to connect to server: %v", err)
				source.Close()
				return
			}

			logger.Info("[udp] server connected")

			go utils.Copy(source, target)
			go utils.Copy(target, source)
		}()
	}
}

func acceptUDP(udp *net.UDPConn) (*UDPConn, error) {
	buffer := make([]byte, 1024)
	n, addr, err := udp.ReadFromUDP(buffer)
	if err != nil {
		return nil, err
	}

	return &UDPConn{
		Origin:     udp,
		OriginAddr: addr,
		Buffer:     buffer[:n],
	}, nil
}

type UDPConn struct {
	net.Conn

	Origin     *net.UDPConn
	OriginAddr *net.UDPAddr

	Buffer []byte
}

func (c *UDPConn) Read(b []byte) (n int, err error) {
	return copy(b, c.Buffer), err
}

func (c *UDPConn) Write(b []byte) (n int, err error) {
	return c.Origin.WriteToUDP(b, c.OriginAddr)
}

func (c *UDPConn) Close() error {
	return nil
}

func (c *UDPConn) LocalAddr() net.Addr {
	return c.Origin.LocalAddr()
}

func (c *UDPConn) RemoteAddr() net.Addr {
	return c.Origin.RemoteAddr()
}
