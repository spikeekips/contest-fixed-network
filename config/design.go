package config

import (
	"net"
	"net/url"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"golang.org/x/xerrors"
)

var (
	defaultStorageURI = "mongodb://127.0.0.1:27017/contest"
	defaultStorage    connstring.ConnString
)

var (
	reConditionStringFormat = `\{\{[\s]*[a-zA-Z0-9_\.][a-zA-Z0-9_\.]*[\s]*\}\}`
	reConditionString       = regexp.MustCompile(reConditionStringFormat)
)

func init() {
	if cs, err := CheckMongodbURI(defaultStorageURI); err != nil {
		panic(err)
	} else {
		defaultStorage = cs
	}
}

type Design struct {
	StorageString    string
	Storage          connstring.ConnString `json:"-"`
	Hosts            []DesignHost
	NodeConfig       map[ /* node alias */ string]string
	CommonNodeConfig string
	NodesConfig      string
	Sequences        []DesignSequence
	ExitOnError      bool
}

func (de *Design) IsValid([]byte) error {
	if len(de.StorageString) < 1 {
		de.Storage = defaultStorage
	} else if cs, err := CheckMongodbURI(de.StorageString); err != nil {
		return err
	} else {
		de.Storage = cs
	}

	if len(de.Hosts) < 1 {
		de.Hosts = []DesignHost{defaultLocalDesignHost()}
	} else {
		for i := range de.Hosts {
			if err := de.Hosts[i].IsValid(nil); err != nil {
				return err
			}
		}
	}

	for i := range de.Sequences {
		if err := de.Sequences[i].IsValid(nil); err != nil {
			return err
		}
	}

	return nil
}

func (de *Design) SetDatabase(s string) error {
	if i, err := url.Parse(de.Storage.String()); err != nil {
		return err
	} else {
		i.Path = s

		if cs, err := CheckMongodbURI(i.String()); err != nil {
			return err
		} else {
			de.Storage = cs

			return nil
		}
	}
}

type DesignHost struct {
	Weight uint // if 0 weight, this host will be ignored.
	Host   string
	Local  bool
	SSH    DesignHostSSH
}

func defaultLocalDesignHost() DesignHost {
	return DesignHost{Weight: 1, Local: true}
}

func (de *DesignHost) IsValid([]byte) error {
	if !de.Local {
		if err := de.SSH.IsValid(nil); err != nil {
			return err
		}

		if len(de.Host) < 1 {
			if !strings.Contains(de.SSH.Host, ":") {
				de.Host = de.SSH.Host
			} else if h, _, err := net.SplitHostPort(de.SSH.Host); err != nil {
				return err
			} else {
				de.Host = h
			}
		}
	}

	if len(de.Host) < 1 {
		return xerrors.Errorf("host is missing")
	}

	return nil
}

type DesignHostSSH struct {
	Host string
	User string
	Key  string
}

func (de *DesignHostSSH) IsValid([]byte) error {
	if len(de.Host) < 1 {
		return xerrors.Errorf("empty host for remote")
	} else if strings.Contains(de.Host, ":") {
		if _, _, err := net.SplitHostPort(de.Host); err != nil {
			return err
		}
	}

	if len(de.User) < 1 {
		return xerrors.Errorf("empty user for remote")
	}

	return nil
}
