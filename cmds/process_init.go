package cmds

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"

	"github.com/spikeekips/contest/config"
)

const (
	HookNameBase = "base"
)

var defaultLogDir = filepath.Clean("./contest")

func HookBase(ctx context.Context) (context.Context, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var flags map[string]interface{}
	if err := config.LoadFlagsContextValue(ctx, &flags); err != nil {
		return ctx, err
	}

	testName := config.ULID().String()

	logDir := flags["LogDir"].(string)
	if len(logDir) < 1 {
		logDir = defaultLogDir

		log.Log().Debug().Str("log_dir", defaultLogDir).Msg("log directory is empty, default directory will be used")
	} else {
		logDir = filepath.Clean(logDir)
	}

	testDir, err := filepath.Abs(filepath.Join(logDir, testName))
	if err != nil {
		return ctx, err
	}

	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testDir, 0o700); err != nil {
			return ctx, xerrors.Errorf("failed to create log directory, %q", testDir)
		}

		log.Log().Debug().Str("test_dir", logDir).Msg("test log directory created")
	}

	log.Log().Info().Str("test_dir", logDir).Str("test_name", testName).Msg("base prepared")

	ctx = context.WithValue(ctx, config.ContextValueTestName, testName)
	ctx = context.WithValue(ctx, config.ContextValueLogDir, testDir)

	return ctx, nil
}
