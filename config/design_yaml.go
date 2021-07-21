package config

import (
	"strings"

	"golang.org/x/xerrors"
)

type DesignYAML struct {
	Storage     *string
	Hosts       []*DesignHostYAML
	NodeConfig  map[ /* node alias */ string]interface{} `yaml:"node-config"`
	NodesConfig *string                                  `yaml:"nodes-config"`
	Sequences   []*DesignSequenceYAML
	ExitOnError *bool `yaml:"exit-on-error"`
}

func (de DesignYAML) Merge() (Design, error) {
	design := Design{}

	if de.Storage != nil {
		design.StorageString = strings.TrimSpace(*de.Storage)
	}

	if i, err := de.mergeHosts(); err != nil {
		return design, err
	} else {
		design.Hosts = i
	}

	if i, j, err := de.mergeNodeConfigs(); err != nil {
		return design, err
	} else {
		design.NodeConfig = i
		design.CommonNodeConfig = j
	}

	if de.NodesConfig != nil {
		design.NodesConfig = *de.NodesConfig
	}

	if i, err := de.mergeSequences(); err != nil {
		return design, err
	} else {
		design.Sequences = i
	}

	if i, err := de.mergeEtc(design); err != nil {
		return design, err
	} else {
		design = i
	}

	return design, nil
}

func (de DesignYAML) mergeHosts() ([]DesignHost, error) {
	if de.Hosts == nil {
		return nil, nil
	}

	hosts := make([]DesignHost, len(de.Hosts))
	for i := range de.Hosts {
		h := de.Hosts[i]
		if d, err := h.Merge(); err != nil {
			return nil, err
		} else {
			hosts[i] = d
		}
	}

	return hosts, nil
}

func (de DesignYAML) mergeNodeConfigs() (map[string]string, string, error) {
	nodeConfig := map[string]string{}

	var commonNodeConfig string
	if c, found := de.NodeConfig["common"]; found {
		if c != nil {
			commonNodeConfig = c.(string)
		}
	}

	for k := range de.NodeConfig {
		if k == "common" {
			continue
		}

		if v := de.NodeConfig[k]; v != nil {
			if s, ok := v.(string); !ok {
				return nil, "", xerrors.Errorf("node config should be string, not %T", v)
			} else {
				nodeConfig[k] = s
			}
		} else {
			nodeConfig[k] = ""
		}
	}

	return nodeConfig, commonNodeConfig, nil
}

func (de DesignYAML) mergeSequences() ([]DesignSequence, error) {
	ss := make([]DesignSequence, len(de.Sequences))
	for i := range de.Sequences {
		if d, err := de.Sequences[i].Merge(); err != nil {
			return nil, err
		} else {
			ss[i] = d
		}
	}

	return ss, nil
}

func (de DesignYAML) mergeEtc(design Design) (Design, error) { // nolint:unparam
	if de.ExitOnError == nil {
		design.ExitOnError = true
	} else {
		design.ExitOnError = *de.ExitOnError
	}

	return design, nil
}

type DesignHostYAML struct {
	Weight *uint
	Local  *bool
	Host   *string
	SSH    *DesignHostSSHYAML
}

func (de DesignHostYAML) Merge() (DesignHost, error) {
	design := DesignHost{}

	if de.Weight != nil {
		design.Weight = *de.Weight
	}

	if de.Local != nil {
		design.Local = *de.Local
	}

	if de.Host != nil {
		design.Host = *de.Host
	}

	if de.SSH != nil {
		if d, err := de.SSH.Merge(); err != nil {
			return design, err
		} else {
			design.SSH = d
		}
	}

	return design, nil
}

type DesignHostSSHYAML struct {
	Host *string
	User *string
	Key  *string
}

func (de DesignHostSSHYAML) Merge() (DesignHostSSH, error) {
	design := DesignHostSSH{}

	if de.Host != nil {
		design.Host = strings.TrimSpace(*de.Host)
	}

	if de.User != nil {
		design.User = strings.TrimSpace(*de.User)
	}

	if de.Key != nil {
		design.Key = strings.TrimSpace(*de.Key)
	}

	return design, nil
}
