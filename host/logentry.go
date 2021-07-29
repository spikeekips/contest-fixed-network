package host

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
)

type LogEntry interface {
	Msg() []byte
	IsError() bool
	Write(io.Writer) error
	Map() (map[string]interface{}, error)
}

type NodeLogEntry struct {
	node    string
	msg     []byte
	isError bool
	isJSON  bool
}

func NewNodeLogEntry(node string, msg []byte, isError bool) (NodeLogEntry, error) {
	if len(node) < 1 {
		return NodeLogEntry{}, xerrors.Errorf("empty node string")
	}

	var isJSON bool
	if m := strings.TrimSpace(string(msg)); strings.HasPrefix(m, "{") && strings.HasSuffix(m, "}") {
		isJSON = true
	}

	return NodeLogEntry{
		node:    node,
		msg:     msg,
		isError: isError,
		isJSON:  isJSON,
	}, nil
}

func NewNodeLogEntryWithInterface(node string, i interface{}, isError bool) (NodeLogEntry, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return NodeLogEntry{}, err
	}

	return NewNodeLogEntry(node, b, isError)
}

func (ls NodeLogEntry) Node() string {
	return ls.node
}

func (ls NodeLogEntry) Msg() []byte {
	return ls.msg
}

func (ls NodeLogEntry) IsError() bool {
	return ls.isError
}

func (ls NodeLogEntry) Write(w io.Writer) error {
	_, err := fmt.Fprintln(w, string(ls.msg))

	return err
}

func (ls NodeLogEntry) Map() (map[string]interface{}, error) {
	var msg interface{}
	if ls.isJSON {
		var m bson.M
		if err := json.Unmarshal(ls.msg, &m); err != nil {
			return nil, err
		}
		msg = m
	} else {
		msg = string(ls.msg)
	}

	return map[string]interface{}{
		"node":     ls.node,
		"x":        msg,
		"is_error": ls.isError,
	}, nil
}

type ContestLogEntry struct {
	msg     []byte
	isError bool
	isJSON  bool
}

func NewContestLogEntry(msg []byte, isError bool) ContestLogEntry {
	m := strings.TrimSpace(string(msg))
	return ContestLogEntry{
		msg:     []byte(m),
		isError: isError,
		isJSON:  strings.HasPrefix(m, "{") && strings.HasSuffix(m, "}"),
	}
}

func (ls ContestLogEntry) Msg() []byte {
	return ls.msg
}

func (ls ContestLogEntry) IsError() bool {
	return ls.isError
}

func (ls ContestLogEntry) Write(w io.Writer) error {
	_, err := fmt.Fprintln(w, string(ls.msg))

	return err
}

func (ls ContestLogEntry) Map() (map[string]interface{}, error) {
	var msg interface{}
	if ls.isJSON {
		var m bson.M
		if err := json.Unmarshal(ls.msg, &m); err != nil {
			return nil, err
		}
		msg = m
	} else {
		msg = string(ls.msg)
	}

	return map[string]interface{}{
		"m":        msg,
		"is_error": ls.isError,
	}, nil
}
