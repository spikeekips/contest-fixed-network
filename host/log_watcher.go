package host

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/contest/config"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type LogWatcher struct {
	sync.RWMutex
	storagePoolLock sync.RWMutex
	*logging.Logging
	*util.ContextDaemon
	mg          *Mongodb
	sqs         []*Sequence
	exitChan    chan error
	vars        *config.Vars
	cl          int
	storagePool map[string]*Mongodb
}

func NewLogWatcher(mg *Mongodb, sqs []*Sequence, exitChan chan error, vars *config.Vars) (*LogWatcher, error) {
	if len(sqs) < 1 {
		return nil, errors.Errorf("empty conditions")
	}

	lw := &LogWatcher{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "log-watcher")
		}),
		mg:          mg,
		sqs:         sqs,
		exitChan:    exitChan,
		vars:        vars,
		storagePool: map[string]*Mongodb{},
	}

	lw.ContextDaemon = util.NewContextDaemon("log-watcher", lw.start)

	return lw, nil
}

func (lw *LogWatcher) start(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	current, _ := lw.Current()
	lw.Log().Debug().Str("condition", current.Condition().QueryString()).Msg("starts with sequence")

	var stopError error
end:
	for {
		select {
		case <-ctx.Done():
			break end
		case <-ticker.C:
			sq, found := lw.Current()
			if !found {
				continue
			}

			if finished, err := lw.evaluate(sq); err != nil {
				stopError = err

				break end
			} else if finished {
				lw.Log().Info().Msg("all conditions are matched")

				break end
			}
		}
	}

	go func() {
		lw.exitChan <- stopError
	}()

	return nil
}

func (lw *LogWatcher) Stop() error {
	lw.Lock()
	defer lw.Unlock()

	for uri := range lw.storagePool {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		if err := lw.storagePool[uri].Close(ctx); err != nil {
			lw.Log().Error().Err(err).Msg("failed to close mongodb")
		}
		cancel()
	}

	if !lw.ContextDaemon.IsStarted() {
		return nil
	}

	return lw.ContextDaemon.Stop()
}

func (lw *LogWatcher) Current() (*Sequence, bool) {
	lw.RLock()
	defer lw.RUnlock()

	return lw.current()
}

func (lw *LogWatcher) current() (*Sequence, bool) {
	if lw.cl == len(lw.sqs) {
		return nil, false
	}

	return lw.sqs[lw.cl], true
}

func (lw *LogWatcher) evaluate(sq *Sequence) (bool, error) {
	lw.Lock()
	defer lw.Unlock()

	l := lw.Log().With().Interface("condition", sq.Condition().QueryString()).Logger()

	var record interface{}
	switch i, matched, err := sq.Condition().Check(lw.vars, lw.getStorage); {
	case err != nil:
		return false, err
	case !matched:
		return false, nil
	default:
		record = i

		lw.vars.Set("Register.last_match", record)
	}

	sq.SetRegister(lw.vars, record)

	l.Info().Interface("matched", record).Msg("codition matched")

	lw.cl++

	finished := lw.cl == len(lw.sqs)

	if _, ok := sq.Action().(NullAction); !ok {
		l.Debug().Interface("action", sq.Action()).Msg("trying to run action")
		if err := sq.Action().Run(context.Background()); err != nil {
			l.Error().Err(err).Msg("failed to run action")

			return finished, err
		}
	}

	if nsq, found := lw.current(); found {
		_, err := nsq.Condition().Query(lw.vars)
		if err != nil {
			return false, err
		}

		l.Debug().Interface("next_condition", nsq.Condition().QueryString()).Msg("will wait next sequence")
	}

	return finished, nil
}

func (lw *LogWatcher) getStorage(uri string) (*Mongodb, error) {
	lw.storagePoolLock.Lock()
	defer lw.storagePoolLock.Unlock()

	if i, found := lw.storagePool[uri]; found {
		return i, nil
	}

	l := lw.Log().With().Str("uri", uri).Logger()

	timeout := time.Second * 3
	l.Debug().Dur("timeout", timeout).Msg("connecting storage")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if i, err := NewMongodbFromString(uri); err != nil {
		return nil, errors.Wrap(err, "failed to ready storage")
	} else if err := i.Connect(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to connect storage")
	} else {
		l.Debug().Msg("storage connected")

		lw.storagePool[uri] = i

		return i, nil
	}
}
