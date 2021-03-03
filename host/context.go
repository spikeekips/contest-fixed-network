package host

import (
	"context"

	"github.com/spikeekips/mitum/util"
)

var (
	ContextValueHosts      util.ContextKey = "hosts"
	ContextValueMongodb    util.ContextKey = "mongodb"
	ContextValueLogSaver   util.ContextKey = "log_saver"
	ContextValueLogWatcher util.ContextKey = "log_watcher"
)

func LoadHostsContextValue(ctx context.Context, l **Hosts) error {
	return util.LoadFromContextValue(ctx, ContextValueHosts, l)
}

func LoadMongodbContextValue(ctx context.Context, l **Mongodb) error {
	return util.LoadFromContextValue(ctx, ContextValueMongodb, l)
}

func LoadLogSaverContextValue(ctx context.Context, l **LogSaver) error {
	return util.LoadFromContextValue(ctx, ContextValueLogSaver, l)
}

func LoadLogWatcherContextValue(ctx context.Context, l **LogWatcher) error {
	return util.LoadFromContextValue(ctx, ContextValueLogWatcher, l)
}
