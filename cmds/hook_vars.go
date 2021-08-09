package cmds

import (
	"bytes"
	"context"
	"os"
	"text/template"

	"github.com/pkg/errors"
	"github.com/spikeekips/contest/config"
	"gopkg.in/yaml.v3"
)

const HookNameVars = "vars"

func HookVars(ctx context.Context) (context.Context, error) {
	var flags map[string]interface{}
	if err := config.LoadFlagsContextValue(ctx, &flags); err != nil {
		return ctx, err
	}

	configSource := flags["Design"].([]byte)

	// vars
	vars := config.NewVars(nil)
	vars.Set("Runtime", map[string]interface{}{
		"Args":  os.Args,
		"Flags": flags,
		"Node":  map[string]interface{}{},
	})

	var m map[string]interface{}
	if err := yaml.Unmarshal(configSource, &m); err != nil {
		return ctx, err
	}
	vars.Set("Design.Contest", config.SanitizeVarsMap(m))

	if i, found := m["vars"]; found {
		varsString, ok := i.(string)
		if !ok {
			return ctx, errors.Errorf("vars not string, %T", i)
		}

		var bf bytes.Buffer
		if t, err := template.New("design-vars").Funcs(vars.FuncMap()).Parse(varsString); err != nil {
			return ctx, errors.Wrap(err, "failed to compile vars string")
		} else if err := t.Execute(&bf, vars.Map()); err != nil {
			return ctx, errors.Wrap(err, "failed to compile vars string")
		}
	}

	return context.WithValue(ctx, config.ContextValueVars, vars), nil
}
