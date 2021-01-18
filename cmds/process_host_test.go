package cmds

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testSpreadContainers struct {
	suite.Suite
}

func (t *testSpreadContainers) TestSanitizeMap() {
	cases := []struct {
		name     string
		n        uint
		weights  []uint
		expected []uint
	}{
		{
			name:     "nil",
			n:        3,
			weights:  []uint{1, 1, 1},
			expected: []uint{1, 1, 1},
		},
		{
			name:     "empty",
			n:        0,
			weights:  []uint{1, 1, 1},
			expected: []uint{0},
		},
		{
			name:     "one",
			n:        1,
			weights:  []uint{1, 1, 1},
			expected: []uint{1},
		},
		{
			name:     "2",
			n:        2,
			weights:  []uint{1, 1, 1},
			expected: []uint{1, 1, 0},
		},
		{
			name:     "10",
			n:        10,
			weights:  []uint{2, 1, 1},
			expected: []uint{6, 3, 1},
		},
		{
			name:     "9",
			n:        9,
			weights:  []uint{2, 1, 1, 1},
			expected: []uint{5, 2, 1, 1},
		},
		{
			name:     "12",
			n:        12,
			weights:  []uint{2, 1, 1, 1},
			expected: []uint{6, 4, 1, 1},
		},
		{
			name:     "14",
			n:        14,
			weights:  []uint{2, 1, 1, 1},
			expected: []uint{7, 4, 2, 1},
		},
		{
			name:     "14",
			n:        14,
			weights:  []uint{2, 1, 1, 0},
			expected: []uint{8, 5, 1, 0},
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				r := calcSpreadNodes(c.n, c.weights)
				t.Equal(c.expected, r, "%d: %s: %v != %v", i, c.name, c.expected, r)
			},
		)
	}
}

func TestSpreadContainers(t *testing.T) {
	suite.Run(t, new(testSpreadContainers))
}
