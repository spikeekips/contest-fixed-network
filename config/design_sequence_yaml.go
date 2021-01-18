package config

import (
	"strings"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

type DesignSequenceYAML struct {
	Condition interface{}
	Action    *DesignActionYAML   `yaml:",omitempty"`
	Register  *DesignRegisterYAML `yaml:"register,omitempty"`
}

func (de DesignSequenceYAML) Merge() (DesignSequence, error) {
	design := DesignSequence{}

	if de.Condition != nil {
		if i, err := parseCondition(de.Condition); err != nil {
			return design, err
		} else if d, err := i.Merge(); err != nil {
			return design, err
		} else {
			design.Condition = d
		}
	}

	if de.Action != nil {
		if i, err := de.Action.Merge(); err != nil {
			return design, err
		} else {
			design.Action = i
		}
	}

	if de.Register != nil {
		if i, err := de.Register.Merge(); err != nil {
			return design, err
		} else {
			design.Register = i
		}
	}

	return design, nil
}

type DesignActionYAML struct {
	Name *string
	Args *[]string
}

func (de DesignActionYAML) Merge() (DesignAction, error) {
	if de.Name == nil {
		if de.Args == nil {
			return DesignAction{}, nil
		}

		return DesignAction{}, xerrors.Errorf("empty action name")
	}

	var args []string
	if de.Args != nil && len(*de.Args) > 0 {
		args = make([]string, len(*de.Args))
		for i := range *de.Args {
			args[i] = strings.TrimSpace((*de.Args)[i])
		}
	}

	return DesignAction{Name: *de.Name, Args: args}, nil
}

type DesignRegisterYAML struct {
	Type *string `yaml:"type"`
	To   *string `yaml:"to"`
}

func (de DesignRegisterYAML) Merge() (DesignRegister, error) {
	design := DesignRegister{}

	if de.Type != nil {
		if s := strings.TrimSpace(*de.Type); len(s) < 1 {
			return design, xerrors.Errorf("empty type")
		} else {
			design.Type = DesignRegisterType(s)
		}
	}

	if de.To != nil {
		if s := strings.TrimSpace(*de.To); len(s) < 1 {
			return design, xerrors.Errorf("empty to")
		} else {
			design.To = s
		}
	}

	return design, nil
}

type DesignConditionYAML struct {
	Query   *string `yaml:"query"`
	Storage *string `yaml:"storage,omitempty"`
	Col     *string `yaml:"col,omitempty"`
}

func (de DesignConditionYAML) Merge() (DesignCondition, error) {
	design := DesignCondition{}

	if de.Query != nil {
		design.Query = strings.TrimSpace(*de.Query)
	}

	if de.Storage != nil {
		design.Storage = strings.TrimSpace(*de.Storage)
	}

	if de.Col != nil {
		design.Col = strings.TrimSpace(*de.Col)
	}

	return design, nil
}

func parseCondition(v interface{}) (DesignConditionYAML, error) {
	design := DesignConditionYAML{}

	switch t := v.(type) {
	case string:
		design.Query = &t

		return design, nil
	case map[string]interface{}:
		if b, err := yaml.Marshal(t); err != nil {
			return design, xerrors.Errorf("invalid yaml for condition: %w", err)
		} else if err := yaml.Unmarshal(b, &design); err != nil {
			return design, xerrors.Errorf("invalid DesignConditionYAML: %w", err)
		} else {
			return design, nil
		}
	default:
		return design, xerrors.Errorf("wrong type for DesignCondition, %T", v)
	}
}
