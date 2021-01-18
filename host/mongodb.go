package host

import (
	"context"
	"strings"

	"github.com/spikeekips/contest/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"golang.org/x/xerrors"
)

var colLogEntry = "log"

var logEntryIndexModel = []mongo.IndexModel{
	{
		Keys:    bson.D{bson.E{Key: "node", Value: 1}},
		Options: options.Index().SetName("contest_logentry_node"),
	},
	{
		Keys:    bson.D{bson.E{Key: "is_stderr", Value: 1}},
		Options: options.Index().SetName("contest_logentry_is_stderr"),
	},
}

type Mongodb struct {
	cs     connstring.ConnString
	client *mongo.Client
	db     *mongo.Database
}

func NewMongodb(cs connstring.ConnString) *Mongodb {
	return &Mongodb{cs: cs}
}

func NewMongodbFromString(uri string) (*Mongodb, error) {
	if cs, err := config.CheckMongodbURI(uri); err != nil {
		return nil, err
	} else {
		return &Mongodb{cs: cs}, nil
	}
}

func (mg *Mongodb) Connect(ctx context.Context) error {
	clientOpts := options.Client().ApplyURI(mg.cs.String())
	if err := clientOpts.Validate(); err != nil {
		return err
	}

	var client *mongo.Client
	if c, err := mongo.Connect(ctx, clientOpts); err != nil {
		return err
	} else {
		client = c
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return err
	} else {
		mg.client = client
		mg.db = client.Database(mg.cs.Database)

		return nil
	}
}

func (mg *Mongodb) Close(ctx context.Context) error {
	return mg.client.Disconnect(ctx)
}

func (mg *Mongodb) Initialize(context.Context) error {
	return mg.createIndices(colLogEntry, logEntryIndexModel, "contest_")
}

func (mg *Mongodb) AddLogEntries(ctx context.Context, entries []LogEntry) error {
	if mg.client == nil || mg.db == nil {
		return xerrors.Errorf("not yet connected")
	}

	models := make([]mongo.WriteModel, len(entries))
	for i := range entries {
		models[i] = mongo.NewInsertOneModel().SetDocument(NewLogEntryBSON(entries[i]))
	}

	opts := options.BulkWrite().SetOrdered(true)
	if _, err := mg.db.Collection(colLogEntry).BulkWrite(ctx, models, opts); err != nil {
		return err
	} else {
		return nil
	}
}

func (mg *Mongodb) Find(ctx context.Context, col string, query bson.M) (map[string]interface{}, bool, error) {
	option := options.FindOne()
	option = option.SetSort(bson.D{{Key: "_id", Value: -1}})

	var record map[string]interface{}
	if r := mg.db.Collection(col).FindOne(ctx, query, option); r.Err() != nil {
		if xerrors.Is(r.Err(), mongo.ErrNoDocuments) {
			return nil, false, nil
		}

		return nil, false, r.Err()
	} else if err := r.Decode(&record); err != nil {
		return nil, true, err
	} else {
		return record, true, nil
	}
}

func (mg *Mongodb) createIndices(col string, models []mongo.IndexModel, prefix string) error {
	iv := mg.db.Collection(col).Indexes()

	cursor, err := iv.List(context.TODO())
	if err != nil {
		return err
	}

	var existings []string
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		return err
	} else {
		for _, r := range results {
			name := r["name"].(string)
			if !strings.HasPrefix(name, prefix) {
				continue
			}

			existings = append(existings, name)
		}
	}

	if len(existings) > 0 {
		for _, name := range existings {
			if _, err := iv.DropOne(context.TODO(), name); err != nil {
				return err
			}
		}
	}

	if len(models) < 1 {
		return nil
	}

	if _, err := iv.CreateMany(context.TODO(), models); err != nil {
		return err
	} else {
		return nil
	}
}

type LogEntryBSON struct {
	l LogEntry
}

func NewLogEntryBSON(l LogEntry) LogEntryBSON {
	return LogEntryBSON{l: l}
}

func (lo LogEntryBSON) MarshalBSON() ([]byte, error) {
	if m, err := lo.l.Map(); err != nil {
		return nil, err
	} else {
		m["_id"] = config.ULID().String()

		return bson.Marshal(m)
	}
}
