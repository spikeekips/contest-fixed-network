package config

import (
	"context"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	ContextValueTestName config.ContextKey = "test_name"
	ContextValueLogDir   config.ContextKey = "log_dir"
	ContextValueLog      config.ContextKey = "log"
	ContextValueDesign   config.ContextKey = "design"
	ContextValueVars     config.ContextKey = "vars"
	ContextValueHosts    config.ContextKey = "hosts"
	ContextValueFlags    config.ContextKey = "flags"
)

func LoadLogDirContextValue(ctx context.Context, l *string) error {
	return config.LoadFromContextValue(ctx, ContextValueLogDir, l)
}

func LoadTestNameContextValue(ctx context.Context, l *string) error {
	return config.LoadFromContextValue(ctx, ContextValueTestName, l)
}

func LoadLogContextValue(ctx context.Context, l *logging.Logger) error {
	return config.LoadFromContextValue(ctx, ContextValueLog, l)
}

func LoadDesignContextValue(ctx context.Context, l *Design) error {
	return config.LoadFromContextValue(ctx, ContextValueDesign, l)
}

func LoadVarsContextValue(ctx context.Context, l **Vars) error {
	return config.LoadFromContextValue(ctx, ContextValueVars, l)
}

func LoadFlagsContextValue(ctx context.Context, l *map[string]interface{}) error {
	return config.LoadFromContextValue(ctx, ContextValueFlags, l)
}
