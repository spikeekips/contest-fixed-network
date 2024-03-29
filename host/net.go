package host

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"sync"

	"github.com/pkg/errors"
)

var (
	bindPortRange = [2]int64{1025, 32767}
	bindPortLock  sync.RWMutex
)

func AvailablePort(network string, exclude []string) (string, error) {
	bindPortLock.Lock()
	defer bindPortLock.Unlock()

	switch network {
	case "tcp":
		return availableTCPPortWithExcludes(exclude)
	case "udp":
		return availableUDPPortWithExcludes(exclude)
	default:
		return "", errors.Errorf("unknown network, %q", network)
	}
}

func availableTCPPortWithExcludes(excludes []string) (string, error) {
	for {
		port, err := availableTCPPort()
		if err != nil {
			return port, err
		}
		var found bool
		for _, p := range excludes {
			if port == p {
				found = true

				break
			}
		}

		if !found {
			return port, nil
		}
	}
}

func availableTCPPort() (string, error) {
	if addr, err := net.ResolveTCPAddr("tcp", "localhost:0"); err != nil {
		return "", err
	} else if l, err := net.ListenTCP("tcp", addr); err != nil {
		return "", err
	} else {
		defer func() {
			_ = l.Close()
		}()

		return fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port), nil
	}
}

func randPorts() string {
	n, err := rand.Int(rand.Reader, big.NewInt(bindPortRange[1]-bindPortRange[0]))
	if err != nil {
		panic(err)
	}

	i := n.Int64() + bindPortRange[0]

	return fmt.Sprintf("%d", i)
}

func availableUDPPortWithExcludes(excludes []string) (string, error) {
	var port string
	for {
		port = randPorts()
		var found bool
		for _, p := range excludes {
			if port == p {
				found = true

				break
			}
		}

		if found {
			continue
		}

		if err := availableUDPPort(port); err == nil {
			break
		}
	}

	return port, nil
}

func availableUDPPort(port string) error {
	if addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("localhost:%s", port)); err != nil {
		return err
	} else if l, err := net.ListenUDP("udp", addr); err != nil {
		return err
	} else {
		defer func() {
			_ = l.Close()
		}()

		return nil
	}
}
