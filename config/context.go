package config

import (
	"context"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	ContextValueTestName util.ContextKey = "test_name"
	ContextValueLogDir   util.ContextKey = "log_dir"
	ContextValueLog      util.ContextKey = "log"
	ContextValueDesign   util.ContextKey = "design"
	ContextValueVars     util.ContextKey = "vars"
	ContextValueHosts    util.ContextKey = "hosts"
	ContextValueFlags    util.ContextKey = "flags"
)

func LoadLogDirContextValue(ctx context.Context, l *string) error {
	return util.LoadFromContextValue(ctx, ContextValueLogDir, l)
}

func LoadTestNameContextValue(ctx context.Context, l *string) error {
	return util.LoadFromContextValue(ctx, ContextValueTestName, l)
}

func LoadLogContextValue(ctx context.Context, l *logging.Logger) error {
	return util.LoadFromContextValue(ctx, ContextValueLog, l)
}

func LoadDesignContextValue(ctx context.Context, l *Design) error {
	return util.LoadFromContextValue(ctx, ContextValueDesign, l)
}

func LoadVarsContextValue(ctx context.Context, l **Vars) error {
	return util.LoadFromContextValue(ctx, ContextValueVars, l)
}

func LoadFlagsContextValue(ctx context.Context, l *map[string]interface{}) error {
	return util.LoadFromContextValue(ctx, ContextValueFlags, l)
}
