package host

import (
	"context"

	"github.com/spikeekips/mitum/launch/config"
)

var (
	ContextValueHosts      config.ContextKey = "hosts"
	ContextValueMongodb    config.ContextKey = "mongodb"
	ContextValueLogSaver   config.ContextKey = "log_saver"
	ContextValueLogWatcher config.ContextKey = "log_watcher"
)

func LoadHostsContextValue(ctx context.Context, l **Hosts) error {
	return config.LoadFromContextValue(ctx, ContextValueHosts, l)
}

func LoadMongodbContextValue(ctx context.Context, l **Mongodb) error {
	return config.LoadFromContextValue(ctx, ContextValueMongodb, l)
}

func LoadLogSaverContextValue(ctx context.Context, l **LogSaver) error {
	return config.LoadFromContextValue(ctx, ContextValueLogSaver, l)
}

func LoadLogWatcherContextValue(ctx context.Context, l **LogWatcher) error {
	return config.LoadFromContextValue(ctx, ContextValueLogWatcher, l)
}
