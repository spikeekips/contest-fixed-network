package host

import "fmt"

type NodeStderrError struct {
	Err  []byte
	Node string
}

func NewNodeStderrError(node string, b []byte) NodeStderrError {
	return NodeStderrError{Node: node, Err: b}
}

func (e NodeStderrError) Error() string {
	s := string(e.Err)
	if len(s) > 50 {
		s = s[:50] + " ..."
	}

	return fmt.Sprintf("node, %q occurred stderr: %s...", e.Node, s)
}

func (e NodeStderrError) String() string {
	return fmt.Sprintf(`node, %q occurred stderr:
================================================================================
%s
================================================================================`, e.Node, string(e.Err))
}
