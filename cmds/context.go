package cmds

import (
	"context"

	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

var (
	ContextValueExitError util.ContextKey = "exit_error"
	ContextValueExitChan  util.ContextKey = "exit_chan"
)

func LoadExitErrorContextValue(ctx context.Context, l *error) error {
	if err := util.LoadFromContextValue(ctx, ContextValueExitError, l); err != nil {
		if xerrors.Is(err, util.ContextValueNotFoundError) {
			return nil
		}

		return err
	} else {
		return nil
	}
}

func LoadExitChanContextValue(ctx context.Context, l *chan error) error {
	return util.LoadFromContextValue(ctx, ContextValueExitChan, l)
}
