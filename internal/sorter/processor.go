package sorter

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/thespags/tfsortplus/internal/config"
)

// Processor handles file discovery and sorting.
type Processor struct {
	Recursive bool
	Check     bool
	Diff      bool
	Cfg       *config.Config

	extensions   []string
	excludedDirs []string
	results      []*Result
}

// NewProcessor creates a new Processor with the given config.
func NewProcessor() *Processor {
	return &Processor{
		extensions:   []string{".tf", ".hcl", ".tofu"},
		excludedDirs: []string{".git", ".terraform"},
	}
}

// Process processes the given directory.
func (p *Processor) Process(root string) ([]*Result, error) {
	p.results = nil

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return p.walkDir(root, path, entry)
		}

		if !p.isValidExtension(entry.Name()) {
			return nil
		}

		if p.Cfg.ShouldIgnore(entry.Name()) {
			return nil
		}

		return p.processFile(path)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to process files: %w", err)
	}

	return p.results, nil
}

// Print processes the results into a string.
func (p *Processor) Print() (string, error) {
	var sb strings.Builder

	changed, unchanged, errored := p.printResults(&sb)

	if p.Check {
		fmt.Fprintf(&sb, "\nCheck: %d not sorted, %d sorted", changed, unchanged)
	} else {
		fmt.Fprintf(&sb, "\n%d changed, %d unchanged", changed, unchanged)
	}

	if errored > 0 {
		fmt.Fprintf(&sb, ", %d errors", errored)

		return sb.String(), fmt.Errorf("%d files had errors", errored)
	}

	if p.Check && changed > 0 {
		return sb.String(), fmt.Errorf("%d files are not sorted", changed)
	}

	return sb.String(), nil
}

func (p *Processor) walkDir(root, path string, entry fs.DirEntry) error {
	// Always process the root directory
	if path == root {
		return nil
	}

	// For non-recursive, skip all subdirectories
	if !p.Recursive {
		return fs.SkipDir
	}

	if p.isExcludedDir(entry.Name()) {
		return fs.SkipDir
	}

	if p.Cfg.ShouldIgnore(entry.Name()) {
		return fs.SkipDir
	}

	return nil
}

// processFile processes a single file.
func (p *Processor) processFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	sorter := NewSorter(p.Cfg)
	result := sorter.SortFile(path, content)

	p.results = append(p.results, result)

	if result.Error != nil {
		return nil //nolint:nilerr // Parse errors are recorded in result, not returned
	}

	if result.Changed && !p.Check {
		if err := os.WriteFile(path, result.Content, 0o600); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	return nil
}

// isValidExtension checks if a filename has a valid extension.
func (p *Processor) isValidExtension(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))

	return slices.Contains(p.extensions, ext)
}

// isExcludedDir checks if a directory should be skipped.
func (p *Processor) isExcludedDir(name string) bool {
	return slices.Contains(p.excludedDirs, name)
}

func (p *Processor) printResults(sb *strings.Builder) (int, int, int) {
	var changed, unchanged, errored int

	for _, result := range p.results {
		switch {
		case result.Error != nil:
			fmt.Fprintf(sb, "error: %s: %v\n", result.Path, result.Error)

			errored++
		case result.Changed:
			if p.Check {
				fmt.Fprintf(sb, "not sorted: %s\n", result.Path)
			} else {
				fmt.Fprintf(sb, "changed: %s\n", result.Path)
			}

			if p.Diff {
				printDiff(sb, result)
			}

			changed++
		default:
			unchanged++
		}
	}

	return changed, unchanged, errored
}

func printDiff(sb *strings.Builder, result *Result) {
	fmt.Fprintf(sb, "--- %s (original)\n", result.Path)
	fmt.Fprintf(sb, "+++ %s (sorted)\n", result.Path)
	fmt.Fprintln(sb, "@@ changes @@")

	fmt.Fprintln(sb, "Original:")
	fmt.Fprintln(sb, string(result.Original))
	fmt.Fprintln(sb, "Sorted:")
	fmt.Fprintln(sb, string(result.Content))
}
