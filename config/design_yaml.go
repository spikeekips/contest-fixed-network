package config

import (
	"strings"

	"github.com/pkg/errors"
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

	i, err := de.mergeHosts()
	if err != nil {
		return design, err
	}
	design.Hosts = i

	j, k, err := de.mergeNodeConfigs()
	if err != nil {
		return design, err
	}
	design.NodeConfig = j
	design.CommonNodeConfig = k

	if de.NodesConfig != nil {
		design.NodesConfig = *de.NodesConfig
	}

	m, err := de.mergeSequences()
	if err != nil {
		return design, err
	}
	design.Sequences = m

	n, err := de.mergeEtc(design)
	if err != nil {
		return design, err
	}
	design = n

	return design, nil
}

func (de DesignYAML) mergeHosts() ([]DesignHost, error) {
	if de.Hosts == nil {
		return nil, nil
	}

	hosts := make([]DesignHost, len(de.Hosts))
	for i := range de.Hosts {
		h := de.Hosts[i]
		d, err := h.Merge()
		if err != nil {
			return nil, err
		}
		hosts[i] = d
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
			s, ok := v.(string)
			if !ok {
				return nil, "", errors.Errorf("node config should be string, not %T", v)
			}
			nodeConfig[k] = s
		} else {
			nodeConfig[k] = ""
		}
	}

	return nodeConfig, commonNodeConfig, nil
}

func (de DesignYAML) mergeSequences() ([]DesignSequence, error) {
	ss := make([]DesignSequence, len(de.Sequences))
	for i := range de.Sequences {
		d, err := de.Sequences[i].Merge()
		if err != nil {
			return nil, err
		}
		ss[i] = d
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
		d, err := de.SSH.Merge()
		if err != nil {
			return design, err
		}
		design.SSH = d
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
