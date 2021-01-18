package host

import (
	"context"
	"sync"

	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type Hosts struct {
	sync.RWMutex
	*logging.Logging
	lo    *LogSaver
	hosts map[ /* Host.Host() */ string]Host
}

func NewHosts(lo *LogSaver) *Hosts {
	return &Hosts{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "hosts")
		}),
		lo:    lo,
		hosts: map[string]Host{},
	}
}

func (hs *Hosts) LenHosts() int {
	return len(hs.hosts)
}

func (hs *Hosts) LenNodes() int {
	var i int
	_ = hs.TraverseHosts(func(h Host) (bool, error) {
		i += len(h.Nodes())

		return true, nil
	})

	return i
}

func (hs *Hosts) TraverseHosts(callback func(h Host) (bool, error)) error {
	for i := range hs.hosts {
		if keep, err := callback(hs.hosts[i]); err != nil {
			return err
		} else if !keep {
			return nil
		}
	}

	return nil
}

func (hs *Hosts) TraverseNodes(callback func(node *Node) (bool, error)) error {
	for i := range hs.hosts {
		nodes := hs.hosts[i].Nodes()
		for node := range nodes {
			if keep, err := callback(nodes[node]); err != nil {
				return err
			} else if !keep {
				return nil
			}
		}
	}

	return nil
}

func (hs *Hosts) NodeExists(alias string) bool {
	var found bool
	_ = hs.TraverseNodes(func(node *Node) (bool, error) {
		if node.Alias() == alias {
			found = true

			return false, nil
		}

		return true, nil
	})

	return found
}

func (hs *Hosts) AddHost(h Host) error {
	hs.Lock()
	defer hs.Unlock()

	if _, found := hs.hosts[h.Host()]; found {
		return xerrors.Errorf("host, %q already added", h.Host())
	}

	hs.hosts[h.Host()] = h

	nodes := make([]string, len(h.Nodes()))
	var i int
	for node := range h.Nodes() {
		nodes[i] = node
		i++
	}

	hs.Log().Debug().Str("host", h.Host()).Strs("nodes", nodes).Msg("host added with nodes")

	return nil
}

func (hs *Hosts) Close() error {
	hs.Lock()
	defer hs.Unlock()

	hosts := make([]Host, len(hs.hosts))
	var i int
	for h := range hs.hosts {
		hosts[i] = hs.hosts[h]
		i++
	}

	return RunWaitGroup(len(hosts), func(i int) error {
		h := hosts[i]

		l := hs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
			return ctx.Str("host", h.Host())
		})
		if err := h.Close(context.Background()); err != nil {
			l.Error().Err(err).Msg("failed to close host")

			return err
		} else {
			l.Debug().Msg("host closed")

			return nil
		}
	})
}
