package cmds

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"text/template"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/spikeekips/contest/config"
	"github.com/spikeekips/contest/host"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

func filterNodes(hosts *host.Hosts, aliases []string) ([]*host.Node, error) {
	allNodes := make([]*host.Node, hosts.LenNodes())

	var i int
	_ = hosts.TraverseNodes(func(node *host.Node) (bool, error) {
		allNodes[i] = node
		i++

		return true, nil
	})

	if l := len(aliases); l < 1 {
		return allNodes, nil
	} else if l > 1 {
		founds := map[string]struct{}{}
		for _, alias := range aliases {
			if _, found := founds[alias]; found {
				return nil, xerrors.Errorf("duplicated node, %q found", alias)
			} else {
				founds[alias] = struct{}{}
			}
		}
	}

	nodes := make([]*host.Node, len(aliases))

	var j int
	for _, alias := range aliases {
		var found bool
		for i := range allNodes {
			if node := allNodes[i]; node.Alias() == alias {
				nodes[j] = node
				j++

				found = true
				break
			}
		}

		if !found {
			return nil, xerrors.Errorf("node, %q not found", alias)
		}
	}

	return nodes, nil
}

func filterRunningContainers(nodes []*host.Node, running bool) (map[string]string, error) {
	ids := map[string]string{}
	for i := range nodes {
		ids[nodes[i].Alias()] = ""
	}

	traversed := map[string]struct{}{}
	for i := range nodes {
		h := nodes[i].Host()
		if _, found := traversed[h.Host()]; found {
			continue
		}

		if err := host.TraverseContainers(h.DockerClient(), func(c dockerTypes.Container) (bool, error) {
			if c.Labels[host.ContainerLabelNodeType] != host.ContainerLabelNodeRunType {
				return true, nil
			}

			alias := c.Labels[host.ContainerLabelNodeAlias]
			var found bool
			for j := range nodes {
				if nodes[j].Alias() == alias {
					found = true

					break
				}
			}

			switch {
			case !found:
			case running && c.State == "running": // running will be ignored
				delete(ids, alias)
			case !running && c.State != "running": // not running will be ignored
				delete(ids, alias)
			default:
				ids[alias] = c.ID
			}

			return true, nil
		}); err != nil {
			return nil, err
		}

		traversed[h.Host()] = struct{}{}
	}

	return ids, nil
}

// calcSpreadNodes spread number by it's weight. weights should be sorted.
func calcSpreadNodes(n uint /* total number of nodes */, weights []uint) []uint {
	if n < 2 {
		return []uint{n}
	}

	counts := make([]uint, len(weights))
	var assigned uint
	var sum uint
	for i, w := range weights {
		if w < 1 {
			continue
		}

		counts[i] = 1
		assigned++
		sum += w

		if assigned == n {
			return counts
		}
	}

	d := n - assigned
	if d < 1 {
		return counts
	} else if d == 1 {
		var top int /* index of top weight */
		for i, w := range weights {
			if w > weights[top] {
				top = i
			}
		}
		counts[top]++

		return counts
	}

	r := float64(n) / float64(sum)

end:
	for {
		for i, w := range weights {
			j := uint(math.Ceil(float64(w) * r))
			switch a := assigned + j; {
			case a == n:
				counts[i] += j

				break end
			case a > n:
				counts[i] += n - assigned

				break end
			default:
				counts[i] += j
				assigned += j
			}
		}
	}

	return counts
}

func newHost(ctx context.Context, de config.DesignHost, nodeDesigns map[string]string) (host.Host, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var vars *config.Vars
	if err := config.LoadVarsContextValue(ctx, &vars); err != nil {
		return nil, err
	}

	var flags map[string]interface{}
	if err := config.LoadFlagsContextValue(ctx, &flags); err != nil {
		return nil, err
	}

	var logDir string
	if err := config.LoadLogDirContextValue(ctx, &logDir); err != nil {
		return nil, err
	}

	runnerFile := flags["RunnerFile"].(string)

	var h host.Host
	if de.Local { //nolint // TODO implement RemoteHost
		h = host.NewLocalHost(de, vars, nodeDesigns, runnerFile, logDir)
	} else {
		h = host.NewLocalHost(de, vars, nodeDesigns, runnerFile, logDir)
	}

	if l, ok := h.(logging.SetLogger); ok {
		_ = l.SetLogger(log)
	}

	return h, h.Connect()
}

func parseSequence(ctx context.Context, design config.DesignSequence) (*host.Sequence, error) {
	var condition *host.Condition
	if i, err := host.NewCondition(
		ctx,
		design.Condition.Query,
		design.Condition.Storage,
		design.Condition.Col,
	); err != nil {
		return nil, err
	} else {
		condition = i
	}

	var action host.Action = host.NullAction{}
	if !design.Action.IsEmpty() {
		if i, err := parseSequenceAction(ctx, design.Action); err != nil {
			return nil, err
		} else {
			action = i
		}
	}

	return host.NewSequence(condition, action, design.Register)
}

func parseSequenceAction(ctx context.Context, design config.DesignAction) (host.Action, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	if i, found := ActionLoaders[design.Name]; !found {
		return nil, xerrors.Errorf("unknown action, %q found", design.Name)
	} else if action, err := i(ctx, design); err != nil {
		return nil, xerrors.Errorf("failed to load action, %q: %w", design.Name, err)
	} else {
		if l, ok := action.(logging.SetLogger); ok {
			_ = l.SetLogger(log)
		}

		return action, nil
	}
}

func generateNodesConfig(ctx context.Context, design config.Design, hosts *host.Hosts) (map[string][]byte, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var vars *config.Vars
	if err := config.LoadVarsContextValue(ctx, &vars); err != nil {
		return nil, err
	}

	shared := map[string]interface{}{}
	if err := hosts.TraverseHosts(func(h host.Host) (bool, error) {
		var sh map[string]interface{}
		if s, err := h.Prepare(design.CommonNodeConfig, vars); err != nil {
			return false, err
		} else if s != nil {
			sh = s
			for k := range s {
				shared[k] = s[k]
			}
		}

		log.Debug().Str("host", h.Host()).Interface("shared", sh).Msg("host prepared")

		return true, nil
	}); err != nil {
		return nil, err
	}

	configs := map[string][]byte{}
	if err := hosts.TraverseNodes(func(node *host.Node) (bool, error) {
		s := shared["nodes-config"].(map[string]interface{})
		ns := map[string]interface{}{}
		for k := range s {
			if k == node.Alias() {
				continue
			}

			ns[k] = s[k]
		}

		nodesVars := config.NewVars(map[string]interface{}{"NodesConfig": ns})

		var bf bytes.Buffer
		if t, err := template.New("nodes-config").Funcs(nodesVars.FuncMap()).Parse(design.NodesConfig); err != nil {
			return false, err
		} else if err := t.Execute(&bf, nodesVars.Map()); err != nil {
			return false, err
		}

		configs[node.Alias()] = bf.Bytes()

		return true, nil
	}); err != nil {
		return nil, err
	}

	return configs, nil
}

func saveNodeConfig(node, logDir string, configData, nodesConfig []byte) error {
	c := configData
	c = append(c, '\n')
	c = append(c, nodesConfig...)

	configFile := filepath.Join(logDir, fmt.Sprintf("%s.yml", node))

	return ioutil.WriteFile(configFile, c, 0o600)
}
