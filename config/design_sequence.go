package config

import (
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

type DesignSequence struct {
	Condition DesignCondition
	Action    DesignAction
	Register  DesignRegister
}

func (de *DesignSequence) IsValid([]byte) error {
	if err := de.Condition.IsValid(nil); err != nil {
		return err
	} else if err := de.Action.IsValid(nil); err != nil {
		return err
	} else if err := de.Register.IsValid(nil); err != nil {
		return err
	}

	return nil
}

type DesignAction struct {
	Name  string
	Args  []string
	Extra map[string]interface{}
}

func (de DesignAction) IsEmpty() bool {
	return len(de.Name) < 1
}

func (de *DesignAction) IsValid([]byte) error {
	if len(de.Name) < 1 {
		if len(de.Args) < 1 {
			return nil
		}

		return errors.Errorf("empty action name")
	}

	return nil
}

func CheckMongodbURI(uri string) (connstring.ConnString, error) {
	cs, err := connstring.Parse(uri)
	if err != nil {
		return connstring.ConnString{}, err
	}

	if len(cs.Database) < 1 {
		return connstring.ConnString{}, errors.Errorf("empty database name in mongodb uri: '%v'", uri)
	}

	return cs, nil
}

type DesignRegisterType string

const (
	RegisterLastMatchType DesignRegisterType = "last_match"
)

func (t DesignRegisterType) IsValid([]byte) error {
	if t != RegisterLastMatchType {
		return errors.Errorf("unknown register type, %q", t)
	}

	return nil
}

type DesignRegister struct {
	Type DesignRegisterType
	To   string
}

func (de DesignRegister) IsEmpty() bool {
	return len(de.Type) < 1 || len(de.To) < 1
}

func (de *DesignRegister) IsValid([]byte) error {
	if de.IsEmpty() {
		de.Type = ""
		de.To = ""

		return nil
	}

	if len(de.Type) < 1 || len(de.To) < 1 {
		return errors.Errorf("type and to empty")
	}

	return de.Type.IsValid(nil)
}

func IsTemplateCondition(s string) bool {
	return len(reConditionString.Find([]byte(s))) > 0
}

func ParseConditionQuery(s string) (bson.M, error) {
	if len(s) < 1 {
		return nil, errors.Errorf("empty condition query")
	}

	var q bson.M

	b := []byte(s)
	if IsTemplateCondition(s) {
		b = reConditionString.ReplaceAll(b, []byte("1"))
	}

	if err := bson.UnmarshalExtJSON(b, false, &q); err != nil {
		return nil, errors.Wrap(err, "bad condition query string")
	}

	return q, nil
}

type DesignCondition struct {
	Query   string
	Storage string
	Col     string
}

func (de *DesignCondition) IsValid([]byte) error {
	if len(de.Query) < 1 {
		return errors.Errorf("empty condition query")
	} else if _, err := ParseConditionQuery(de.Query); err != nil {
		return err
	}

	if len(de.Storage) > 0 {
		if !IsTemplateCondition(de.Storage) {
			if _, err := CheckMongodbURI(de.Storage); err != nil {
				return err
			}
		}
	}

	return nil
}
