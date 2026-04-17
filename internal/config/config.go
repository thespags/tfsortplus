package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

// Config represents the sorting configuration.
type Config struct {
	// Order defines the sorting priority. Supports regex patterns like "resource\.gitlab_.*".
	Order []string `yaml:"order"`

	// AlphabeticalTies sorts blocks with the same order alphabetically by name.
	AlphabeticalTies bool `yaml:"alphabeticalTies"`

	// UnknownFirst places unmatched blocks before known blocks when true, after when false (default).
	UnknownFirst bool `yaml:"unknownFirst"`

	// Last defines sorting priority after order and unknowns.
	// e.g., output will appear after unknown resources by default.
	Last []string `yaml:"last"`

	// LocalsFirst places locals blocks before all other blocks. Default true.
	LocalsFirst bool `yaml:"localsFirst"`

	// OutputLast places output blocks after all other blocks (including unknowns). Default true.
	OutputLast bool `yaml:"outputLast"`

	// Ignore lists file/directory patterns to skip (regex patterns).
	Ignore []string `yaml:"ignore"`

	// compiled patterns for matching
	patterns       []*regexp.Regexp
	lastPatterns   []*regexp.Regexp
	ignorePatterns []*regexp.Regexp
}

// Default returns a default config with no ordering (preserves original order).
func Default() *Config {
	return &Config{
		AlphabeticalTies: true,
		LocalsFirst:      true,
		OutputLast:       true,
	}
}

// GetConfig loads the config from the given directory.
func GetConfig(dir string) (*Config, error) {
	path := findPath(dir)

	if path == "" {
		cfg := Default()

		return cfg, cfg.Compile()
	}

	return load(path)
}

// Compile compiles the configs Order and Ignore fields into compiled regex patterns.
func (c *Config) Compile() error {
	order := c.Order
	if c.LocalsFirst {
		order = append([]string{"locals"}, order...)
	}

	last := c.Last
	if c.OutputLast {
		last = append(last, `output\..*`)
	}

	var err, errs error

	c.patterns, err = compile(order)
	errs = errors.Join(errs, err)

	c.lastPatterns, err = compile(last)
	errs = errors.Join(errs, err)

	c.ignorePatterns, err = compile(c.Ignore)
	errs = errors.Join(errs, err)

	return errs
}

// findPath searches for a config file starting from the given directory.
func findPath(dir string) string {
	configFileNames := []string{".tfsortplus.yaml", ".tfsortplus.yml", "tfsortplus.yaml", "tfsortplus.yml"}

	for _, name := range configFileNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// load reads a config file from the given path.
func load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := Config{
		AlphabeticalTies: true,
		LocalsFirst:      true,
		OutputLast:       true,
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	if err := cfg.Compile(); err != nil {
		return nil, fmt.Errorf("failed to compile patterns: %w", err)
	}

	return &cfg, nil
}

func compile(rawPattern []string) ([]*regexp.Regexp, error) {
	patterns := make([]*regexp.Regexp, 0, len(rawPattern))

	for _, pattern := range rawPattern {
		pattern = anchorPattern(pattern)

		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}

		patterns = append(patterns, regex)
	}

	return patterns, nil
}

// anchorPattern adds ^ and $ anchors if not already present.
func anchorPattern(pattern string) string {
	if !strings.HasPrefix(pattern, "^") {
		pattern = "^" + pattern
	}

	if !strings.HasSuffix(pattern, "$") {
		pattern += "$"
	}

	return pattern
}

// ShouldIgnore checks if a path should be ignored.
func (c *Config) ShouldIgnore(path string) bool {
	for _, re := range c.ignorePatterns {
		if re.MatchString(path) {
			return true
		}
	}

	return false
}

// GetOrder returns the order for a block type and resource type.
// Order priority: patterns (0...n-1), unknowns (n or -1), lastPatterns (n+1...).
func (c *Config) GetOrder(blockType, resourceType string) int {
	key := blockType
	if resourceType != "" {
		key += "." + resourceType
	}

	for i, pattern := range c.patterns {
		if pattern.MatchString(key) {
			return i
		}
	}

	// Check last patterns (should sort after unknowns)
	for i, pattern := range c.lastPatterns {
		if pattern.MatchString(key) {
			return len(c.patterns) + 1 + i
		}
	}

	// Unknown block
	if c.UnknownFirst {
		return -1
	}

	return len(c.patterns)
}

// ShouldSortAlphabetically returns whether ties should be sorted alphabetically.
func (c *Config) ShouldSortAlphabetically() bool {
	return c.AlphabeticalTies
}
