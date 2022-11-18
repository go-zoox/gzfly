package tcp

import (
	"fmt"
	"net"

	"github.com/go-zoox/gzfly/network/utils"
	"github.com/go-zoox/logger"
)

type ServeConfig struct {
	Host   string
	Port   int
	OnConn func() (net.Conn, error)
}

func Serve(cfg *ServeConfig) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	logger.Info("listen tcp server at: %s", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	for {
		source, err := listener.Accept()
		if err != nil {
			continue
		}

		logger.Info("[tcp] client connected")

		go func() {
			target, err := cfg.OnConn()
			if err != nil {
				logger.Warn("[warning] failed to connect to server: %v", err)
				source.Close()
				return
			}

			logger.Info("[tcp] server connected")

			go utils.Copy(source, target)
			go utils.Copy(target, source)
		}()
	}
}
