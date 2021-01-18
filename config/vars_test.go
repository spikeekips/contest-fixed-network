package config

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testVars struct {
	suite.Suite
}

func (t *testVars) TestGetItem() {
	cases := []struct {
		name     string
		v        interface{}
		key      string
		expected interface{}
		found    bool
	}{
		{
			name:     "found",
			v:        map[string]interface{}{"a": 1},
			key:      "a",
			expected: 1,
			found:    true,
		},
		{
			name:  "not found",
			v:     map[string]interface{}{"a": 1},
			key:   "b",
			found: false,
		},
		{
			name:  "not map",
			v:     "k",
			key:   "a",
			found: false,
		},
		{
			name:     "sub map",
			v:        map[string]interface{}{"a": map[string]interface{}{"b": 2}},
			key:      "a.b",
			expected: 2,
			found:    true,
		},
		{
			name:  "sub map: not found #0",
			v:     map[string]interface{}{"a": map[string]interface{}{"b": 2}},
			key:   "a.c",
			found: false,
		},
		{
			name:  "sub map: not found #1",
			v:     map[string]interface{}{"a": map[string]interface{}{"b": 2}},
			key:   "d.c",
			found: false,
		},
		{
			name:  "sub map: error",
			v:     map[string]interface{}{"a": map[string]interface{}{"b": 2}},
			key:   "a.b.c",
			found: false,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				r, found := getVar(c.v, c.key)
				t.Equal(c.found, found, "%d: %s: %v != %v", i, c.name, c.found, found)
				t.Equal(c.expected, r, "%d: %s: %v != %v", i, c.name, c.expected, r)
			},
		)
	}
}

func (t *testVars) TestSetItem() {
	cases := []struct {
		name     string
		m        map[string]interface{}
		v        interface{}
		key      string
		err      string
		expected map[string]interface{}
	}{
		{
			name: "simple",
			v:    1,
			key:  "a",
		},
		{
			name:     "nested: deep",
			v:        1,
			key:      "a.b.c.d",
			expected: map[string]interface{}{"a": map[string]interface{}{"b": map[string]interface{}{"c": map[string]interface{}{"d": 1}}}},
		},
		{
			name: "nested: not exists",
			v:    1,
			key:  "a.b",
		},
		{
			name:     "nested:  exists",
			m:        map[string]interface{}{"a": map[string]interface{}{"c": 3}},
			v:        1,
			key:      "a.b",
			expected: map[string]interface{}{"a": map[string]interface{}{"b": 1, "c": 3}},
		},
		{
			name: "nested:  override",
			m:    map[string]interface{}{"a": map[string]interface{}{"b": []int{0}}},
			v:    "showme",
			key:  "a.b",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				var m map[string]interface{}
				if c.m == nil {
					m = map[string]interface{}{}
				} else {
					m = c.m
				}

				if err := setVar(m, c.key, c.v); err != nil {
					if len(c.err) > 0 {
						t.Contains(err.Error(), c.err, "%d: %s: err=(%+v) expected=(%v)", i, c.name, err, c.err)
					} else {
						t.NoError(err)
					}

					return
				}

				r, found := getVar(m, c.key)
				t.True(found)
				t.Equal(c.v, r, "%d: %s: %v != %v", i, c.name, c.v, r)

				if c.expected != nil {
					t.Equal(c.expected, m, "%d: %s: %v != %v", i, c.name, c.expected, m)
				}
			},
		)
	}
}

func (t *testVars) TestSanitizeMap() {
	cases := []struct {
		name     string
		m        map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil",
			expected: map[string]interface{}{},
		},
		{
			name:     "single char",
			m:        map[string]interface{}{"a": 1, "b": 2},
			expected: map[string]interface{}{"A": 1, "B": 2},
		},
		{
			name:     "long",
			m:        map[string]interface{}{"spike": 1, "ekips": 2},
			expected: map[string]interface{}{"Spike": 1, "Ekips": 2},
		},
		{
			name:     "has space",
			m:        map[string]interface{}{"spike ekips": 1, "ekips ": 2},
			expected: map[string]interface{}{"SpikeEkips": 1, "Ekips": 2},
		},
		{
			name:     "has hyphen",
			m:        map[string]interface{}{"spike-ekips": 1, "ekips ": 2},
			expected: map[string]interface{}{"SpikeEkips": 1, "Ekips": 2},
		},
		{
			name:     "has underscore",
			m:        map[string]interface{}{"spike_ekips": 1, "ekips ": 2},
			expected: map[string]interface{}{"SpikeEkips": 1, "Ekips": 2},
		},
		{
			name:     "has underscore with sub",
			m:        map[string]interface{}{"spike_ekips": 1, "ekips ": map[string]interface{}{"srothan": 3}},
			expected: map[string]interface{}{"SpikeEkips": 1, "Ekips": map[string]interface{}{"Srothan": 3}},
		},
		{
			name:     "underscore suffix",
			m:        map[string]interface{}{"spike_ekips_": 1, "ekips ": 2},
			expected: map[string]interface{}{"SpikeEkips": 1, "Ekips": 2},
		},
		{
			name:     "id",
			m:        map[string]interface{}{"network-id": 1, "ekips ": 2},
			expected: map[string]interface{}{"NetworkID": 1, "Ekips": 2},
		},
		{
			name:     "id prefix",
			m:        map[string]interface{}{"id-network": 1, "ekips ": 2},
			expected: map[string]interface{}{"IDNetwork": 1, "Ekips": 2},
		},
		{
			name:     "sub slice",
			m:        map[string]interface{}{"a": []map[string]interface{}{{"b": 2, "c": 3}}},
			expected: map[string]interface{}{"A": []interface{}{map[string]interface{}{"B": 2, "C": 3}}},
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				m := SanitizeVarsMap(c.m)
				t.Equal(c.expected, m, "%d: %s: %v != %v", i, c.name, c.expected, m)
			},
		)
	}
}

func TestVars(t *testing.T) {
	suite.Run(t, new(testVars))
}
