package sorter

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
)

func TestBlock_Name(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		block            *Block
		expectedName     string
		expectedResource string
	}{
		{
			name: "resource with labels",
			block: &Block{
				BlockType: "resource",
				Labels:    []string{"aws_instance", "example"},
			},
			expectedName:     "aws_instance",
			expectedResource: "example",
		},
		{
			name: "data with labels",
			block: &Block{
				BlockType: "data",
				Labels:    []string{"aws_ami", "ubuntu"},
			},
			expectedName:     "aws_ami",
			expectedResource: "ubuntu",
		},
		{
			name: "locals no labels",
			block: &Block{
				BlockType: "locals",
				Labels:    nil,
			},
			expectedName:     "locals",
			expectedResource: "",
		},
		{
			name: "empty labels",
			block: &Block{
				BlockType: "terraform",
				Labels:    []string{},
			},
			expectedName:     "terraform",
			expectedResource: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedName, tt.block.Name())
			assert.Equal(t, tt.expectedResource, tt.block.ResourceName())
		})
	}
}

func TestBlock_Content(t *testing.T) {
	t.Parallel()

	content := []byte(`# Comment
resource "aws_instance" "example" {
  ami = "ami-123"
}
`)

	block := &Block{
		ContentStart: 0,
		Range: hcl.Range{
			End: hcl.Pos{Byte: len(content) - 1},
		},
	}

	got := block.Content(content)
	assert.Equal(t, content[:len(content)-1], got)
}

func TestCompare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		a            *Block
		b            *Block
		alphabetical bool
		expected     int
	}{
		{
			name:         "different order - a first",
			a:            &Block{Order: 0, Position: 0, Labels: []string{"a"}},
			b:            &Block{Order: 1, Position: 1, Labels: []string{"b"}},
			alphabetical: true,
			expected:     -1,
		},
		{
			name:         "different order - b first",
			a:            &Block{Order: 2, Position: 0, Labels: []string{"a"}},
			b:            &Block{Order: 1, Position: 1, Labels: []string{"b"}},
			alphabetical: true,
			expected:     1,
		},
		{
			name:         "same order - alphabetical - a first",
			a:            &Block{Order: 0, Position: 1, Labels: []string{"alpha"}},
			b:            &Block{Order: 0, Position: 0, Labels: []string{"beta"}},
			alphabetical: true,
			expected:     -1,
		},
		{
			name:         "same order - alphabetical - b first",
			a:            &Block{Order: 0, Position: 0, Labels: []string{"zebra"}},
			b:            &Block{Order: 0, Position: 1, Labels: []string{"alpha"}},
			alphabetical: true,
			expected:     1,
		},
		{
			name:         "same order - no alphabetical - position order",
			a:            &Block{Order: 0, Position: 0, Labels: []string{"zebra"}},
			b:            &Block{Order: 0, Position: 1, Labels: []string{"alpha"}},
			alphabetical: false,
			expected:     -1,
		},
		{
			name:         "same order - no alphabetical - position order reversed",
			a:            &Block{Order: 0, Position: 1, Labels: []string{"alpha"}},
			b:            &Block{Order: 0, Position: 0, Labels: []string{"zebra"}},
			alphabetical: false,
			expected:     1,
		},
		{
			name:         "same order - alphabetical - same labels",
			a:            &Block{Order: 0, Position: 0, Labels: []string{"foo", "bar"}},
			b:            &Block{Order: 0, Position: 1, Labels: []string{"foo", "bar"}},
			alphabetical: true,
			expected:     0,
		},
		{
			name:         "same order - alphabetical - a has more labels",
			a:            &Block{Order: 0, Position: 0, Labels: []string{"foo", "bar"}},
			b:            &Block{Order: 0, Position: 1, Labels: []string{"foo"}},
			alphabetical: true,
			expected:     1,
		},
		{
			name:         "same order - alphabetical - b has more labels",
			a:            &Block{Order: 0, Position: 0, Labels: []string{"foo"}},
			b:            &Block{Order: 0, Position: 1, Labels: []string{"foo", "bar"}},
			alphabetical: true,
			expected:     -1,
		},
		{
			name:         "same order - alphabetical - compare second label",
			a:            &Block{Order: 0, Position: 0, Labels: []string{"aws_instance", "alpha"}},
			b:            &Block{Order: 0, Position: 1, Labels: []string{"aws_instance", "beta"}},
			alphabetical: true,
			expected:     -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Compare(tt.a, tt.b, tt.alphabetical)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestIsSorted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		blocks       []*Block
		alphabetical bool
		expected     bool
	}{
		{
			name:         "empty",
			blocks:       []*Block{},
			alphabetical: true,
			expected:     true,
		},
		{
			name:         "single block",
			blocks:       []*Block{{Order: 0, Position: 0}},
			alphabetical: true,
			expected:     true,
		},
		{
			name: "sorted by order",
			blocks: []*Block{
				{Order: 0, Position: 0, Labels: []string{"a"}},
				{Order: 1, Position: 1, Labels: []string{"b"}},
				{Order: 2, Position: 2, Labels: []string{"c"}},
			},
			alphabetical: true,
			expected:     true,
		},
		{
			name: "not sorted by order",
			blocks: []*Block{
				{Order: 1, Position: 0, Labels: []string{"a"}},
				{Order: 0, Position: 1, Labels: []string{"b"}},
			},
			alphabetical: true,
			expected:     false,
		},
		{
			name: "sorted alphabetically within same order",
			blocks: []*Block{
				{Order: 0, Position: 0, Labels: []string{"alpha"}},
				{Order: 0, Position: 1, Labels: []string{"beta"}},
				{Order: 0, Position: 2, Labels: []string{"gamma"}},
			},
			alphabetical: true,
			expected:     true,
		},
		{
			name: "not sorted alphabetically within same order",
			blocks: []*Block{
				{Order: 0, Position: 0, Labels: []string{"beta"}},
				{Order: 0, Position: 1, Labels: []string{"alpha"}},
			},
			alphabetical: true,
			expected:     false,
		},
		{
			name: "not sorted alphabetically but position preserved",
			blocks: []*Block{
				{Order: 0, Position: 0, Labels: []string{"beta"}},
				{Order: 0, Position: 1, Labels: []string{"alpha"}},
			},
			alphabetical: false,
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsSorted(tt.blocks, tt.alphabetical)
			assert.Equal(t, tt.expected, got)
		})
	}
}
