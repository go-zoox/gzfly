package command

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-zoox/gzfly/core"
	"github.com/go-zoox/gzfly/user"
)

func parseAuth(auth string) (*user.User, error) {
	parts := strings.Split(auth, ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid auth")
	}

	clientID, clientSecret, pairKey := parts[0], parts[1], parts[2]

	return &user.User{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		PairKey:      pairKey,
	}, nil
}

func parseRelay(relayR string) (protocol string, host string, port int, path string, err error) {
	relay, err := url.Parse(relayR)
	if err != nil {
		err = fmt.Errorf("invalid relay: %v", err)
		return
	}

	port = 443
	if relay.Port() != "" {
		port, err = strconv.Atoi(relay.Port())
		if err != nil {
			err = fmt.Errorf("invalid relay port: %v", err)
			return
		}
	}

	protocol = relay.Scheme
	host = relay.Hostname()
	path = relay.Path

	return
}

func parseTarget(target string) (*core.Target, error) {
	parts := strings.Split(target, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid target")
	}

	return &core.Target{
		UserClientID: parts[0],
		UserPairKey:  parts[1],
	}, nil
}

func parseSocks5(address string) (*core.Socks5, error) {
	parts := strings.Split(address, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid socks5")
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid socks5 port")
	}

	return &core.Socks5{
		IP:   parts[0],
		Port: port,
	}, nil
}

func parseBind(bind string) (*core.Bind, error) {
	parts := strings.Split(bind, ":")
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid bind")
	}

	LocalPort, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse local port: %v", err)
	}

	RemotePort, err := strconv.Atoi(parts[4])
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote port: %v", err)
	}

	return &core.Bind{
		Network:    parts[0],
		LocalHost:  parts[1],
		LocalPort:  LocalPort,
		RemoteHost: parts[3],
		RemotePort: RemotePort,
	}, nil
}
