package config

import (
	"crypto/rsa"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

type testDesign struct {
	suite.Suite
}

func (t *testDesign) TestYAMLStorage() {
	y := `
storage: mongodb://3.3.3.3/c
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)
	t.NoError(design.IsValid(nil))

	t.Equal("mongodb://3.3.3.3/c", design.StorageString)
	t.Equal("c", design.Storage.Database)
	t.Equal([]string{"3.3.3.3"}, design.Storage.Hosts)
}

func (t *testDesign) TestYAMLEmptyStorage() {
	// NOTE if 'storage' is empty, default storage uri will be set.
	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(""), &dy))

	design, err := dy.Merge()
	t.NoError(err)
	t.NoError(design.IsValid(nil))

	t.Equal(defaultStorage, design.Storage)
}

func (t *testDesign) TestYAMLEmptyHosts() {
	// NOTE if 'hosts' is empty, local host design will be added.
	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(""), &dy))

	design, err := dy.Merge()
	t.NoError(err)
	t.NoError(design.IsValid(nil))

	t.Equal(1, len(design.Hosts))
	t.True(design.Hosts[0].Local)
}

func (t *testDesign) TestYAMLHostsWithoutSSH() {
	y := `
hosts:
  - weight: 10
  - weight: 3
  - weight: 4
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)

	t.Equal(3, len(design.Hosts))
	t.Equal(uint(10), design.Hosts[0].Weight)
	t.Equal(uint(3), design.Hosts[1].Weight)
	t.Equal(uint(4), design.Hosts[2].Weight)
}

func (t *testDesign) TestYAMLHostsWithSSHButEmptyHostString() {
	y := `
hosts:
  - weight: 10
    ssh:
      user: spike
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)

	err = design.IsValid(nil)
	t.Contains(err.Error(), "empty host for remote")
}

func (t *testDesign) TestYAMLHostsWithSSHButEmptyUserString() {
	y := `
hosts:
  - weight: 10
    ssh:
      host: 4.4.4.4:22
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)

	err = design.IsValid(nil)
	t.Contains(err.Error(), "empty user for remote")
}

func (t *testDesign) TestYAMLHostsWithSSH() {
	y := `
hosts:
  - weight: 10
    ssh:
      host: 4.4.4.4:22
      user: spike
  - weight: 3
    ssh:
      host: 5.5.5.5
      user: ekips
  - weight: 4
    ssh:
      host: 6.6.6.6
      user: cong
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)
	t.NoError(design.IsValid(nil))

	t.Equal(3, len(design.Hosts))
	t.Equal(uint(10), design.Hosts[0].Weight)
	t.Equal("4.4.4.4:22", design.Hosts[0].SSH.Host)
	t.Equal("4.4.4.4", design.Hosts[0].Host)
	t.Equal("spike", design.Hosts[0].SSH.User)
	t.Equal(uint(3), design.Hosts[1].Weight)
	t.Equal("5.5.5.5", design.Hosts[1].SSH.Host)
	t.Equal("5.5.5.5", design.Hosts[1].Host)
	t.Equal("ekips", design.Hosts[1].SSH.User)
	t.Equal(uint(4), design.Hosts[2].Weight)
	t.Equal("6.6.6.6", design.Hosts[2].SSH.Host)
	t.Equal("6.6.6.6", design.Hosts[2].Host)
	t.Equal("cong", design.Hosts[2].SSH.User)
}

func (t *testDesign) TestYAMLNodes() {
	y := `
node-config:
  showme:
  findme:
  killme:
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)

	t.Equal(3, len(design.NodeConfig))
	_, found := design.NodeConfig["showme"]
	t.True(found)
	_, found = design.NodeConfig["findme"]
	t.True(found)
	_, found = design.NodeConfig["killme"]
	t.True(found)
}

func (t *testDesign) TestYAMLNodesWithCommon() {
	y := `
node-config:
  common:
  showme:
  findme:
  killme:
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)

	t.Equal(3, len(design.NodeConfig)) // common not counted
	_, found := design.NodeConfig["showme"]
	t.True(found)
	_, found = design.NodeConfig["findme"]
	t.True(found)
	_, found = design.NodeConfig["killme"]
	t.True(found)
}

func (t *testDesign) TestYAMLEmptySequence() {
	y := `
sequences:
  - condition:
  - condition: findme
  - condition: killme
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)

	err = design.IsValid(nil)
	t.Contains(err.Error(), "empty condition query")
}

func (t *testDesign) TestYAMLSequencesBadCondition() {
	y := `
sequences:
  - condition: >
          killme{"a": 1}
  - condition: >
        {"b": 1}
  - condition: >
          {"c": 1}
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)

	err = design.IsValid(nil)
	t.Contains(err.Error(), "bad condition query")
	t.Contains(err.Error(), "invalid JSON input")
}

func (t *testDesign) TestYAMLSequences() {
	y := `
sequences:
  - condition: >
          {"a": 1}
  - condition: >
        {"b": 1}
  - condition: >
          {"c": 1}
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)
	t.NoError(design.IsValid(nil))

	t.Equal(3, len(design.Sequences)) // common not counted
	t.Equal(`{"a": 1}`, design.Sequences[0].Condition.Query)
	t.Equal(`{"b": 1}`, design.Sequences[1].Condition.Query)
	t.Equal(`{"c": 1}`, design.Sequences[2].Condition.Query)
}

func (t *testDesign) TestYAMLSequenceAction() {
	y := `
sequences:
  - condition: >
          {"a": 1}
    action:
      name: showme
      args:
        - killme
        - eatme
  - condition: >
          {"b": 1}
    action:
      name: findme
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)
	t.NoError(design.IsValid(nil))

	t.Equal(2, len(design.Sequences)) // common not counted

	t.Equal(`{"a": 1}`, design.Sequences[0].Condition.Query)
	t.Equal("showme", design.Sequences[0].Action.Name)
	t.Equal([]string{"killme", "eatme"}, design.Sequences[0].Action.Args)

	t.Equal(`{"b": 1}`, design.Sequences[1].Condition.Query)
	t.Equal("findme", design.Sequences[1].Action.Name)
	t.Empty(design.Sequences[1].Action.Args)
}

func (t *testDesign) TestYAMLSequenceRegister() {
	y := `
sequences:
  - condition: >
          {"a": 1}
    register:
        type: last_match
        to: showme
  - condition: >
          {"b": 2}
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)
	t.NoError(design.IsValid(nil))

	t.Equal(2, len(design.Sequences)) // common not counted

	t.Equal(`{"a": 1}`, design.Sequences[0].Condition.Query)
	t.Equal(RegisterLastMatchType, design.Sequences[0].Register.Type)
	t.Equal("showme", design.Sequences[0].Register.To)

	t.Equal(`{"b": 2}`, design.Sequences[1].Condition.Query)
	t.True(design.Sequences[1].Register.IsEmpty())
}

func (t *testDesign) TestYAMLSequenceRegisterWrongType() {
	y := `
sequences:
  - condition: >
          {"a": 1}
    register:
        type: killme
        to: showme
  - condition: >
          {"b": 2}
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)
	err = design.IsValid(nil)
	t.Contains(err.Error(), "unknown register type")
}

func (t *testDesign) TestYAMLSequenceMapCondition() {
	y := `
sequences:
  - condition:
        query: >
           {"a": 1}
        storage: mongodb://127.0.0.1:27017/showme
	`

	var dy DesignYAML
	t.NoError(yaml.Unmarshal([]byte(strings.TrimSpace(y)), &dy))

	design, err := dy.Merge()
	t.NoError(err)
	t.NoError(design.IsValid(nil))

	t.Equal(1, len(design.Sequences)) // common not counted
	t.Equal(`{"a": 1}`, design.Sequences[0].Condition.Query)
}

func (t *testDesign) TestYAMLLoadStorage() {
	b, err := ioutil.ReadFile(filepath.Clean("./test_simple.yml"))
	t.NoError(err)

	var dy DesignYAML
	t.NoError(yaml.Unmarshal(b, &dy))

	design, err := dy.Merge()
	t.NoError(err)
	t.NoError(design.IsValid(nil))

	// storage
	t.Equal("mongodb://127.0.0.1:27017/contest", design.StorageString)
	t.Equal("contest", design.Storage.Database)
	t.Equal([]string{"127.0.0.1:27017"}, design.Storage.Hosts)
}

func (t *testDesign) TestYAMLLoadHosts() {
	b, err := ioutil.ReadFile(filepath.Clean("./test_simple.yml"))
	t.NoError(err)

	var dy DesignYAML
	t.NoError(yaml.Unmarshal(b, &dy))

	design, err := dy.Merge()
	t.NoError(err)

	// hosts
	t.Equal(1, len(design.Hosts))

	t.Equal(uint(1), design.Hosts[0].Weight)
	t.Equal("localhost:22", design.Hosts[0].SSH.Host)
	t.Equal("ubuntu", design.Hosts[0].SSH.User)

	key, err := ssh.ParseRawPrivateKey([]byte(design.Hosts[0].SSH.Key))
	t.NoError(err)

	pub, err := ssh.NewPublicKey(key.(*rsa.PrivateKey).Public())
	t.NoError(err)

	t.Equal("SHA256:nEf5luBbtptoq+aPVD3QyrAnLD8Oies3VqFfEZtAHkQ", ssh.FingerprintSHA256(pub))

	// nodes
	t.Equal(3, len(design.NodeConfig))
}

func (t *testDesign) TestYAMLLoadNodes() {
	b, err := ioutil.ReadFile(filepath.Clean("./test_simple.yml"))
	t.NoError(err)

	var dy DesignYAML
	t.NoError(yaml.Unmarshal(b, &dy))

	design, err := dy.Merge()
	t.NoError(err)

	// nodes
	t.Equal(3, len(design.NodeConfig))
	_, found := design.NodeConfig["n0"]
	t.True(found)
	_, found = design.NodeConfig["n1"]
	t.True(found)
	_, found = design.NodeConfig["n2"]
	t.True(found)
}

func TestDesign(t *testing.T) {
	suite.Run(t, new(testDesign))
}
