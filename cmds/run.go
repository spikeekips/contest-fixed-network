package cmds

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/xerrors"

	"github.com/spikeekips/contest/config"
	"github.com/spikeekips/contest/host"
)

type RunCommand struct {
	*logging.Logging
	*mitumcmds.LogFlags
	RunnerFile     string             `arg:"" name:"runner-file" type:"existingfile"`
	Design         mitumcmds.FileLoad `arg:"" name:"contest design file" help:"contest design file"`
	ContestLogDir  string             `name:"contest-log-dir" help:"contest logs directory"`
	Force          bool               `name:"force" help:"kill the still running node containers"`
	CleanAfter     bool               `name:"clean-after" help:"clean node containers after exit"`
	ExitAfter      time.Duration      `name:"exit-after" help:"exit contest"`
	version        util.Version
	runProcesses   *pm.Processes
	closeProcesses *pm.Processes
}

func NewRunCommand() (RunCommand, error) {
	cmd := RunCommand{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "command-run")
		}),
		LogFlags: &mitumcmds.LogFlags{},
	}

	cmd.processes()

	return cmd, nil
}

func (cmd *RunCommand) Run(version util.Version) error {
	_, _ = maxprocs.Set(maxprocs.Logger(func(f string, s ...interface{}) {
		cmd.Log().Debug().Msgf(f, s...)
	}))

	if i, err := mitumcmds.SetupLoggingFromFlags(cmd.LogFlags); err != nil {
		return err
	} else {
		_ = cmd.SetLogger(i)
	}

	if err := version.IsValid(nil); err != nil {
		return err
	}

	cmd.Log().Debug().Str("version", version.String()).Interface("flags", cmd).Msg("flags parsed")

	if err := version.IsValid(nil); err != nil {
		return err
	} else {
		cmd.version = version
	}

	var exitError error
	if err := cmd.run(); err != nil {
		cmd.Log().Error().Err(err).Msg("failed to run contest")

		exitError = err
	}

	if err := cmd.close(cmd.runProcesses.Context(), exitError); err != nil {
		if exitError == nil {
			exitError = err
		}
	}

	return exitError
}

func (cmd *RunCommand) run() error {
	sigChan := cmd.connectSig()

	ctx := context.Background()

	exitChan := make(chan error, 100)
	ctx = context.WithValue(ctx, ContextValueExitChan, exitChan)

	runChan := make(chan error)
	go func() {
		runChan <- cmd.processRun(ctx)
	}()

	select {
	case err := <-runChan:
		if err != nil {
			return err
		}
	case sig := <-sigChan:
		return xerrors.Errorf("signal, %v interrupted", sig)
	}

	select {
	case err := <-exitChan:
		var ne host.NodeStderrError
		if xerrors.As(err, &ne) {
			fmt.Fprintln(os.Stderr, ne.String())
		}

		return err
	case sig := <-sigChan:
		return xerrors.Errorf("signal, %v interrupted", sig)
	case <-func() <-chan time.Time {
		if cmd.ExitAfter < 1 {
			cmd.Log().Debug().Msg("will not be expired")

			return make(chan time.Time)
		}

		cmd.Log().Debug().Dur("exit_after", cmd.ExitAfter).Msg("will be exited after")

		return time.After(cmd.ExitAfter)
	}():
		return xerrors.Errorf("expired with exit-after %s", cmd.ExitAfter)
	}
}

func (cmd *RunCommand) close(ctx context.Context, exitError error) error {
	ctx = context.WithValue(ctx, ContextValueExitError, exitError)

	cmd.closeProcesses.SetContext(ctx)
	_ = cmd.closeProcesses.SetLogger(cmd.Log())

	return cmd.closeProcesses.Run()
}

func (cmd *RunCommand) processRun(ctx context.Context) error {
	ctx = context.WithValue(ctx, config.ContextValueLog, cmd.Log())
	ctx = context.WithValue(ctx, config.ContextValueFlags, map[string]interface{}{
		"Design":     []byte(cmd.Design),
		"LogDir":     cmd.ContestLogDir,
		"RunnerFile": cmd.RunnerFile,
		"Force":      cmd.Force,
		"CleanAfter": cmd.CleanAfter,
	})

	cmd.runProcesses.SetContext(ctx)
	_ = cmd.runProcesses.SetLogger(cmd.Log())

	cmd.Log().Info().Msg("trying to run contest")

	return cmd.runProcesses.Run()
}

func (cmd *RunCommand) connectSig() chan os.Signal {
	sigc := make(chan os.Signal, 10)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)

	return sigc
}

func (cmd *RunCommand) processes() {
	startProcesses := pm.NewProcesses()

	for _, p := range []pm.Process{
		ProcessorConfig,
		ProcessorMongodb,
		ProcessorLogSaver,
		ProcessorHosts,
		ProcessorNodes,
		ProcessorLogWatcher,
	} {
		if err := startProcesses.AddProcess(p, false); err != nil {
			panic(err)
		}
	}

	startHooks := []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, pm.INITProcess, HookNameBase, HookBase),
		pm.NewHook(pm.HookPrefixPost, ProcessNameConfig, HookNameVars, HookVars),
		pm.NewHook(pm.HookPrefixPost, ProcessNameConfig, HookNameConfigStorage, HookConfigStorage),
		pm.NewHook(pm.HookPrefixPost, ProcessNameHosts, HookNameCleanStoppedNodeContainers, HookCleanStoppedNodeContainers),
		pm.NewHook(pm.HookPrefixPost, ProcessNameNodes, HookNameContestReady, HookContestReady),
	}
	for i := range startHooks {
		hook := startHooks[i]
		if err := startProcesses.AddHook(hook.Prefix, hook.Process, hook.Name, hook.F, true); err != nil {
			panic(err)
		}
	}

	closeProcesses := pm.NewProcesses()

	closeHooks := []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, pm.INITProcess, HookNameStopLogHandlers, HookStopLogHandlers),
		pm.NewHook(pm.HookPrefixPost, pm.INITProcess, HookNameCloseHosts, HookCloseHosts),
		pm.NewHook(pm.HookPrefixPost, pm.INITProcess, HookNameCloseMongodb, HookCloseMongodb),
		pm.NewHook(pm.HookPrefixPost, pm.INITProcess, "exit_with_error", func(ctx context.Context) (context.Context, error) {
			var exitError error
			switch err := LoadExitErrorContextValue(ctx, &exitError); {
			case err != nil:
				return ctx, err
			case exitError != nil:
				return ctx, exitError
			default:
				return ctx, nil
			}
		}),
		pm.NewHook(pm.HookPrefixPost, pm.INITProcess, HookNameCleanContainers, HookCleanContainers),
	}
	for i := range closeHooks {
		hook := closeHooks[i]
		if err := closeProcesses.AddHook(hook.Prefix, hook.Process, hook.Name, hook.F, true); err != nil {
			panic(err)
		}
	}

	cmd.runProcesses = startProcesses
	cmd.closeProcesses = closeProcesses
}
