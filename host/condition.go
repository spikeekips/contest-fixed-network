package host

import (
	"context"

	"github.com/spikeekips/contest/config"
	"github.com/spikeekips/mitum/util/logging"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
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

	var log logging.Logger
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
		return nil, xerrors.Errorf("local host not found for HostCommandAction")
	}

	co := &Condition{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.
				Str("module", "condition").
				Str("query", q)
		}),
		queryString:   q,
		storageString: storageURI,
		col:           col,
	}

	_ = co.SetLogger(log)

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
		if i, err := config.ParseConditionQuery(co.queryString); err != nil {
			return nil, xerrors.Errorf("invalid compiled condition query string, %q: %w", co.queryString, err)
		} else {
			co.query = i
		}
	}

	if b, err := config.CompileTemplate(co.queryString, vars); err != nil {
		return nil, xerrors.Errorf("failed to compile condition query, %q: %w", co.queryString, err)
	} else {
		if i, err := config.ParseConditionQuery(string(b)); err != nil {
			return nil, xerrors.Errorf("invalid compiled condition query string, %q: %w", co.queryString, err)
		} else {
			co.query = i
		}
	}

	co.Log().Debug().Str("col", co.col).Interface("query", co.query).Msg("querying")

	return co.query, nil
}

func (co *Condition) Check(vars *config.Vars, getStorage func(string) (*Mongodb, error)) (interface{}, bool, error) {
	if co.storage == nil {
		uri := co.storageString
		if config.IsTemplateCondition(uri) {
			if i, err := config.CompileTemplate(uri, vars); err != nil {
				return nil, false, xerrors.Errorf("failed to compile storage uri: %w", err)
			} else {
				uri = string(i)
			}
		}

		if i, err := getStorage(uri); err != nil {
			return nil, false, err
		} else {
			co.storage = i
		}
	}

	var query bson.M
	if i, err := co.Query(vars); err != nil {
		return nil, false, err
	} else {
		query = i
	}

	switch i, found, err := co.storage.Find(context.Background(), co.col, query); {
	case err != nil:
		co.Log().Error().Err(err).Msg("failed to find condition")

		return nil, false, err
	default:
		return i, found, nil
	}
}
