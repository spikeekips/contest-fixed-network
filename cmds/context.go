package cmds

import (
	"context"

	"github.com/spikeekips/mitum/launch/config"
	"golang.org/x/xerrors"
)

var (
	ContextValueExitError config.ContextKey = "exit_error"
	ContextValueExitChan  config.ContextKey = "exit_chan"
)

func LoadExitErrorContextValue(ctx context.Context, l *error) error {
	if err := config.LoadFromContextValue(ctx, ContextValueExitError, l); err != nil {
		if xerrors.Is(err, config.ContextValueNotFoundError) {
			return nil
		}

		return err
	} else {
		return nil
	}
}

func LoadExitChanContextValue(ctx context.Context, l *chan error) error {
	return config.LoadFromContextValue(ctx, ContextValueExitChan, l)
}
