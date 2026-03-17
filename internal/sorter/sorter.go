package sorter

import (
	"bytes"
	"errors"
	"math"
	"path/filepath"
	"slices"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/thespags/tfsortplus/internal/config"
)

// Sorter handles sorting of HCL files.
type Sorter struct {
	cfg *config.Config
}

// NewSorter creates a new Sorter with the given config.
func NewSorter(cfg *config.Config) *Sorter {
	return &Sorter{cfg: cfg}
}

// Result contains the result of sorting a file.
type Result struct {
	Path     string
	Changed  bool
	Content  []byte
	Original []byte
	Error    error
}

// SortFile sorts a single file and returns the result.
func (s *Sorter) SortFile(path string, content []byte) *Result {
	result := &Result{
		Path:     path,
		Original: content,
	}

	blocks, err := s.parseBlocks(content, path)
	if err != nil {
		result.Error = err

		return result
	}

	if len(blocks) == 0 {
		result.Content = content

		return result
	}

	if IsSorted(blocks, s.cfg.ShouldSortAlphabetically()) {
		result.Content = content

		return result
	}

	result.Content = s.sort(content, blocks)
	result.Changed = true

	return result
}

// parseBlocks parses an HCL file and extracts all blocks.
func (s *Sorter) parseBlocks(content []byte, filename string) ([]*Block, error) {
	parser := hclparse.NewParser()

	hclFile, diags := parser.ParseHCL(content, filename)
	if diags.HasErrors() {
		return nil, errors.New(diags.Error())
	}

	body, ok := hclFile.Body.(*hclsyntax.Body)
	if !ok {
		return nil, errors.New("unexpected body type (JSON syntax not supported)")
	}

	blocks := make([]*Block, 0, len(body.Blocks))

	for i, b := range body.Blocks {
		blocks = append(blocks, s.toBlock(i, b))
	}

	// Calculate ContentStart for each block to capture leading comments
	s.attachLeadingComments(content, blocks)

	return blocks, nil
}

// toBlock converts an HCL block to our Block type with ordering.
func (s *Sorter) toBlock(position int, block *hclsyntax.Block) *Block {
	resourceType := ""
	if len(block.Labels) > 0 {
		resourceType = block.Labels[0]
	}

	// For modules, use the source attribute's base name as the resource type
	if block.Type == "module" {
		if attr, exists := block.Body.Attributes["source"]; exists {
			val, diags := attr.Expr.Value(nil)
			if !diags.HasErrors() {
				resourceType = filepath.Base(val.AsString())
			}
		}
	}

	order := s.cfg.GetOrder(block.Type, resourceType)

	return &Block{
		BlockType: block.Type,
		Labels:    block.Labels,
		Order:     order,
		Position:  position,
		Range:     block.Range(),
	}
}

// sort reorders blocks and reconstructs the file content.
func (s *Sorter) sort(content []byte, blocks []*Block) []byte {
	if len(blocks) == 0 {
		return content
	}

	var buf bytes.Buffer

	// Find the earliest ContentStart position (includes leading comments)
	minStart := math.MaxInt
	for _, b := range blocks {
		minStart = min(b.ContentStart, minStart)
	}

	// Write any content before the first block (header comments, etc.)
	if minStart > 0 {
		header := bytes.TrimRight(content[:minStart], "\n")
		_, _ = buf.Write(header)
		_, _ = buf.WriteString("\n\n")
	}

	// Sort blocks
	alphabetical := s.cfg.ShouldSortAlphabetically()

	slices.SortStableFunc(blocks, func(a, b *Block) int {
		return Compare(a, b, alphabetical)
	})

	// Write all blocks with spacing
	for i, b := range blocks {
		blockContent := b.Content(content)

		// Trim leading/trailing whitespace from block content, preserving internal structure
		blockContent = bytes.TrimLeft(blockContent, "\n")

		// Add spacing between blocks
		if i > 0 {
			_, _ = buf.WriteString("\n")
		}

		_, _ = buf.Write(blockContent)

		// Ensure block ends with newline
		if len(blockContent) > 0 && blockContent[len(blockContent)-1] != '\n' {
			_, _ = buf.WriteString("\n")
		}
	}

	return buf.Bytes()
}

// attachLeadingComments calculates ContentStart for each block.
// Leading comments are lines between the previous block's end and this block's start
// that contain # or // comments.
func (*Sorter) attachLeadingComments(content []byte, blocks []*Block) {
	for i, block := range blocks {
		var searchStart int
		if i == 0 {
			searchStart = 0
		} else {
			searchStart = blocks[i-1].Range.End.Byte
		}

		// Find where leading comments for this block start
		block.ContentStart = findLeadingCommentStart(content, searchStart, block.Range.Start.Byte)
	}
}

// findLeadingCommentStart finds the start position of leading comments for a block.
// It looks backward from blockStart to find comment lines that are attached to this block.
func findLeadingCommentStart(content []byte, searchStart, blockStart int) int {
	if searchStart >= blockStart {
		return blockStart
	}

	region := content[searchStart:blockStart]

	// Find the last blank line in the region - comments after that belong to this block
	lastBlankLine := 0

	for i := 0; i < len(region); {
		lineEnd := bytes.IndexByte(region[i:], '\n')
		if lineEnd == -1 {
			lineEnd = len(region)
		} else {
			lineEnd += i + 1
		}

		line := region[i:lineEnd]
		trimmed := bytes.TrimSpace(line)

		// A blank line marks the boundary
		if len(trimmed) == 0 && i > 0 {
			lastBlankLine = i
		}

		i = lineEnd
	}

	return searchStart + lastBlankLine
}
