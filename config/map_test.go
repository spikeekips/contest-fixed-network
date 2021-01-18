package config

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testMap struct {
	suite.Suite
}

func (t *testMap) TestMerge() {
	cases := []struct {
		name     string
		target   map[string]interface{}
		source   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil",
			expected: map[string]interface{}{},
		},
		{
			name:     "simple",
			target:   map[string]interface{}{"a": 1},
			source:   map[string]interface{}{"b": 2},
			expected: map[string]interface{}{"a": 1, "b": 2},
		},
		{
			name:     "missing source",
			target:   map[string]interface{}{"a": 1},
			source:   map[string]interface{}{"b": 2},
			expected: map[string]interface{}{"a": 1, "b": 2},
		},
		{
			name:     "override value",
			target:   map[string]interface{}{"a": 1},
			source:   map[string]interface{}{"a": 2},
			expected: map[string]interface{}{"a": 2},
		},
		{
			name:     "sub branch",
			target:   map[string]interface{}{"a": map[string]interface{}{"b": 1}},
			source:   map[string]interface{}{"a": map[string]interface{}{"c": 2}},
			expected: map[string]interface{}{"a": map[string]interface{}{"b": 1, "c": 2}},
		},
		{
			name:     "sub branch: interface{} -> int",
			target:   map[string]interface{}{"a": map[string]int{"b": 1}},
			source:   map[string]interface{}{"a": map[string]int{"c": 2}},
			expected: map[string]interface{}{"a": map[string]interface{}{"b": 1, "c": 2}},
		},
		{
			name:     "sub branch: interface{} -> string",
			target:   map[string]interface{}{"a": map[string]string{"b": "1"}},
			source:   map[string]interface{}{"a": map[string]string{"c": "2"}},
			expected: map[string]interface{}{"a": map[string]interface{}{"b": "1", "c": "2"}},
		},
		{
			name:     "repalce slice",
			target:   map[string]interface{}{"a": []int{0, 1, 2}},
			source:   map[string]interface{}{"a": []string{"0", "1", "2"}},
			expected: map[string]interface{}{"a": []string{"0", "1", "2"}},
		},
		{
			name:     "new slice",
			target:   map[string]interface{}{"a": []int{0, 1, 2}},
			source:   map[string]interface{}{"b": []string{"0", "1", "2"}},
			expected: map[string]interface{}{"a": []int{0, 1, 2}, "b": []string{"0", "1", "2"}},
		},
		{
			name:     "nil",
			target:   map[string]interface{}{"a": 1, "b": nil},
			source:   map[string]interface{}{"b": map[string]interface{}{"c": 1}},
			expected: map[string]interface{}{"a": 1, "b": map[string]interface{}{"c": 1}},
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				m, _ := MergeItem(c.target, c.source)
				t.Equal(c.expected, m, "%d: %s: %v != %v", i, c.name, c.expected, m)
			},
		)
	}
}

func TestMap(t *testing.T) {
	suite.Run(t, new(testMap))
}
