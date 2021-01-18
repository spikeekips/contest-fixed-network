package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/util"

	"github.com/spikeekips/contest/cmds"
)

var (
	Version string = "v0.0.0"
	options        = []kong.Option{
		kong.Name("mitum-contest"),
		kong.Description("mitum contest"),
		mitumcmds.LogVars,
		kong.Vars{
			"enable_pprof":     "false",
			"mem_pprof_file":   "mitum-contest-mem.pprof",
			"cpu_pprof_file":   "mitum-contest-cpu.pprof",
			"trace_pprof_file": "mitum-contest-trace.pprof",
		},
	}
)

type mainflags struct {
	RunContest cmds.RunCommand `cmd:"" name:"run" help:"run contest"`
}

func main() {
	flags := mainflags{}
	if r, err := cmds.NewRunCommand(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)

		os.Exit(1)
	} else {
		flags.RunContest = r
	}

	ctx := kong.Parse(&flags, options...)

	version := util.Version(Version)
	if err := version.IsValid(nil); err != nil {
		ctx.FatalIfErrorf(err)
	}

	if ctx.Command() == "version" {
		_, _ = fmt.Fprintln(os.Stdout, version)

		os.Exit(0)
	}

	if err := ctx.Run(version); err != nil {
		ctx.FatalIfErrorf(err)
	}

	os.Exit(0)
}
