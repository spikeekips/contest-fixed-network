package host

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/contest/config"
	"github.com/spikeekips/mitum/util/logging"
	"go.mongodb.org/mongo-driver/bson"
)

type Condition struct {
	*logging.Logging
	queryString   string
	query         bson.M
	storageString string
	storage       *Mongodb
	col           string
}

func NewCondition(ctx context.Context, q, storageURI, col string) (*Condition, error) {
	if len(storageURI) < 1 {
		var design config.Design
		if err := config.LoadDesignContextValue(ctx, &design); err != nil {
			return nil, err
		}

		storageURI = design.Storage.String()
	}

	if len(col) < 1 {
		col = colLogEntry
	}

	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var hosts *Hosts
	var local Host
	if err := LoadHostsContextValue(ctx, &hosts); err != nil {
		return nil, err
	} else if err := hosts.TraverseHosts(func(h Host) (bool, error) {
		// NOTE at this time, only local host is allowed exec command
		if i, ok := h.(*LocalHost); ok {
			local = i

			return false, nil
		}

		return true, nil
	}); err != nil {
		return nil, err
	} else if local == nil {
		return nil, errors.Errorf("local host not found for HostCommandAction")
	}

	co := &Condition{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.
				Str("module", "condition").
				Str("query", q)
		}),
		queryString:   q,
		storageString: storageURI,
		col:           col,
	}

	_ = co.SetLogging(log)

	return co, nil
}

func (co *Condition) QueryString() string {
	return co.queryString
}

func (co *Condition) Query(vars *config.Vars) (bson.M, error) {
	if co.query != nil {
		return co.query, nil
	}

	if !config.IsTemplateCondition(co.queryString) {
		i, err := config.ParseConditionQuery(co.queryString)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid compiled condition query string, %q", co.queryString)
		}
		co.query = i
	}

	b, err := config.CompileTemplate(co.queryString, vars)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile condition query, %q", co.queryString)
	}

	i, err := config.ParseConditionQuery(string(b))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid compiled condition query string, %q", co.queryString)
	}
	co.query = i

	co.Log().Debug().Str("col", co.col).Interface("query", co.query).Msg("querying")

	return co.query, nil
}

func (co *Condition) Check(vars *config.Vars, getStorage func(string) (*Mongodb, error)) (interface{}, bool, error) {
	if co.storage == nil {
		uri := co.storageString
		if config.IsTemplateCondition(uri) {
			i, err := config.CompileTemplate(uri, vars)
			if err != nil {
				return nil, false, errors.Wrap(err, "failed to compile storage uri")
			}
			uri = string(i)
		}

		i, err := getStorage(uri)
		if err != nil {
			return nil, false, err
		}
		co.storage = i
	}

	query, err := co.Query(vars)
	if err != nil {
		return nil, false, err
	}

	switch i, found, err := co.storage.Find(context.Background(), co.col, query); {
	case err != nil:
		co.Log().Error().Err(err).Msg("failed to find condition")

		return nil, false, err
	default:
		return i, found, nil
	}
}
