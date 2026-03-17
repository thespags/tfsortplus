package sorter

import (
	"cmp"

	"github.com/hashicorp/hcl/v2"
)

// Block represents a terraform block with its metadata.
type Block struct {
	BlockType string
	Labels    []string
	Order     int
	Position  int
	Range     hcl.Range

	// ContentStart is the byte offset where block content starts,
	// including any leading comments attached to this block.
	ContentStart int
}

// Content extracts the block content from the source, including leading comments.
func (b *Block) Content(content []byte) []byte {
	return content[b.ContentStart:b.Range.End.Byte]
}

// Name returns the block's identifying name (first label or block type).
func (b *Block) Name() string {
	if len(b.Labels) > 0 {
		return b.Labels[0]
	}

	return b.BlockType
}

// ResourceName returns the resource name (second label for resources/data).
func (b *Block) ResourceName() string {
	if len(b.Labels) > 1 {
		return b.Labels[1]
	}

	return ""
}

// Compare compares two blocks for sorting.
// Returns negative if left < right, positive if left > right, zero if equal.
//
//nolint:revive // alphabetical flag is clear and readable
func Compare(left, right *Block, alphabetical bool) int {
	if left.Order != right.Order {
		return cmp.Compare(left.Order, right.Order)
	}

	if !alphabetical {
		return cmp.Compare(left.Position, right.Position)
	}

	// Sort alphabetically by labels
	for i := range left.Labels {
		if i >= len(right.Labels) {
			return 1
		}

		if c := cmp.Compare(left.Labels[i], right.Labels[i]); c != 0 {
			return c
		}
	}

	if len(left.Labels) < len(right.Labels) {
		return -1
	}

	return 0
}

// IsSorted checks if blocks are already in sorted order.
//
//nolint:revive // alphabetical flag is clear and readable
func IsSorted(blocks []*Block, alphabetical bool) bool {
	for i := range len(blocks) - 1 {
		if Compare(blocks[i], blocks[i+1], alphabetical) > 0 {
			return false
		}
	}

	return true
}
