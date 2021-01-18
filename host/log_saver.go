package host

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

var ContestLogName = "contest"

type LogSaver struct {
	*logging.Logging
	*util.FunctionDaemon
	mg          *Mongodb
	exitChan    chan error
	exitOnError bool
	entryChan   chan LogEntry
	ctx         context.Context
	cancel      func()
	logFiles    map[string][2]io.WriteCloser // [stdout, stderr]
}

func NewLogSaver(
	mg *Mongodb,
	logDir string,
	nodes []string,
	exitChan chan error,
	exitOnError bool,
) (*LogSaver, error) {
	ctx, cancel := context.WithCancel(context.Background())

	ls := &LogSaver{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "log-saver")
		}),
		mg:          mg,
		exitChan:    exitChan,
		exitOnError: exitOnError,
		entryChan:   make(chan LogEntry, 100),
		ctx:         ctx,
		cancel:      cancel,
	}

	logFiles := map[string][2]io.WriteCloser{}
	nodes = append(nodes, ContestLogName)
	for _, alias := range nodes {
		var n [2]io.WriteCloser
		if i, err := ls.createLogFile(logDir, fmt.Sprintf("%s.stdout", alias)); err != nil {
			return nil, err
		} else {
			n[0] = i
		}

		if i, err := ls.createLogFile(logDir, fmt.Sprintf("%s.stderr", alias)); err != nil {
			return nil, err
		} else {
			n[1] = i
		}

		logFiles[alias] = n
	}

	ls.logFiles = logFiles
	ls.FunctionDaemon = util.NewFunctionDaemon(ls.start, false)

	return ls, nil
}

func (ls *LogSaver) Stop() error {
	if err := ls.FunctionDaemon.Stop(); err != nil {
		return err
	}

	for k := range ls.logFiles {
		for i := range ls.logFiles[k] {
			if err := ls.logFiles[k][i].Close(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ls *LogSaver) LogEntryChan() chan<- LogEntry {
	return ls.entryChan
}

func (ls *LogSaver) start(stopChan chan struct{}) error {
	defer ls.cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	updated := map[string]struct{}{}
	var entries []LogEntry

end:
	for {
		select {
		case <-stopChan:
			ls.cancel()

			break end
		case <-ticker.C:
			if err := ls.saveEntries(entries, updated); err != nil {
				ls.Log().Error().Err(err).Msg("failed to save log entries")

				continue
			} else if len(entries) > 0 {
				entries = nil
				updated = map[string]struct{}{}
			}
		case entry := <-ls.entryChan:
			if entry == nil {
				continue
			}

			if name, err := ls.saveToFile(entry); err != nil {
				ls.Log().Error().Err(err).Interface("entry", entry).Msg("failed to save log entry")

				continue
			} else {
				updated[name] = struct{}{}
			}

			entries = append(entries, entry)
		}
	}

	return ls.saveEntries(entries, updated)
}

func (ls *LogSaver) saveToFile(entry LogEntry) (string, error) {
	var name string
	if i, ok := entry.(NodeLogEntry); ok {
		name = i.Node()
	} else {
		name = ContestLogName
	}

	var w io.Writer
	switch n, found := ls.logFiles[name]; {
	case !found:
		return "", xerrors.Errorf("log file for %q not found", name)
	case entry.IsError():
		w = n[1]
	default:
		w = n[0]
	}

	if err := entry.Write(w); err != nil {
		return "", xerrors.Errorf("failed to write log file, %q(IsError=%v): %w", name, entry.IsError(), err)
	} else {
		return name, nil
	}
}

func (ls *LogSaver) createLogFile(logDir, name string) (io.WriteCloser, error) {
	if i, err := os.OpenFile(
		filepath.Clean(filepath.Join(logDir, fmt.Sprintf("%s.log", name))),
		os.O_RDWR|os.O_CREATE, 0o600,
	); err != nil {
		return nil, xerrors.Errorf("failed to create log file, %q: %w", name, err)
	} else {
		return i, nil
	}
}

func (ls *LogSaver) syncs(updated map[string]struct{}) error {
	for k := range updated {
		for i := range ls.logFiles[k] {
			if s, ok := ls.logFiles[k][i].(interface{ Sync() error }); !ok {
				continue
			} else if err := s.Sync(); err != nil {
				ls.Log().Error().Err(err).Str("name", k).Bool("is_error", i == 1).Msg("failed to sync log file")

				return err
			}
		}
	}

	return nil
}

func (ls *LogSaver) isLogEntriesStderr(entries []LogEntry) {
	for i := range entries {
		if !ls.isLogEntryStderr(entries[i]) {
			break
		}
	}
}

func (ls *LogSaver) isLogEntryStderr(entry LogEntry) bool {
	switch i, ok := entry.(NodeLogEntry); {
	case !ok:
		return true
	case !i.IsError():
		return true
	default:
		go func(ne NodeLogEntry) {
			ls.exitChan <- NewNodeStderrError(ne.Node(), ne.Msg())
		}(i)

		return false
	}
}

func (ls *LogSaver) saveEntries(entries []LogEntry, updated map[string]struct{}) error {
	if len(entries) < 1 {
		return nil
	}

	if err := ls.mg.AddLogEntries(context.Background(), entries); err != nil {
		ls.Log().Error().Err(err).Msg("failed to insert log entries")

		return err
	}
	ls.Log().Verbose().Int("entries", len(entries)).Msg("log entry inserted")

	if err := ls.syncs(updated); err != nil {
		return err
	}

	if ls.exitOnError {
		ls.isLogEntriesStderr(entries)
	}

	return nil
}
