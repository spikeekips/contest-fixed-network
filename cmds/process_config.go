package cmds

import (
	"context"
	"fmt"

	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/util/logging"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/contest/config"
)

const (
	ProcessNameConfig     = "config"
	HookNameConfigStorage = "config_storage"
)

var ProcessorConfig pm.Process

func init() {
	if i, err := pm.NewProcess(ProcessNameConfig, nil, ProcessConfig); err != nil {
		panic(err)
	} else {
		ProcessorConfig = i
	}
}

func ProcessConfig(ctx context.Context) (context.Context, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var flags map[string]interface{}
	if err := config.LoadFlagsContextValue(ctx, &flags); err != nil {
		return ctx, err
	}

	configSource := flags["Design"].([]byte)

	var design config.Design
	var designYAML config.DesignYAML
	if err := yaml.Unmarshal(configSource, &designYAML); err != nil {
		return ctx, err
	} else if de, err := designYAML.Merge(); err != nil {
		return ctx, err
	} else if err := de.IsValid(nil); err != nil {
		return ctx, err
	} else {
		design = de
	}

	log.Log().Info().Interface("design", design).Msg("design loaded")

	return context.WithValue(ctx, config.ContextValueDesign, design), nil
}

func HookConfigStorage(ctx context.Context) (context.Context, error) {
	var design config.Design
	if err := config.LoadDesignContextValue(ctx, &design); err != nil {
		return ctx, err
	}

	var testName string
	if err := config.LoadTestNameContextValue(ctx, &testName); err != nil {
		return ctx, err
	}

	if err := design.SetDatabase(fmt.Sprintf("%s_%s", design.Storage.Database, testName)); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, config.ContextValueDesign, design), nil
}
