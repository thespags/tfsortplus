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

// DefaultConfigNames are the config file names to search for.

// Config represents the sorting configuration.
type Config struct {
	// Order defines the sorting priority. Supports regex patterns like "resource\.gitlab_.*".
	Order []string `yaml:"order"`

	// AlphabeticalTies sorts blocks with the same order alphabetically by name.
	AlphabeticalTies bool `yaml:"alphabeticalTies"`

	// UnknownFirst places unmatched blocks before known blocks when true, after when false (default).
	UnknownFirst bool `yaml:"unknownFirst"`

	// Ignore lists file/directory patterns to skip (regex patterns).
	Ignore []string `yaml:"ignore"`

	// compiled patterns for matching
	patterns       []*regexp.Regexp
	ignorePatterns []*regexp.Regexp
}

// Default returns a default config with no ordering (preserves original order).
func Default() *Config {
	return &Config{
		AlphabeticalTies: true,
	}
}

// GetConfig loads the config from the given directory.
func GetConfig(dir string) (*Config, error) {
	path := findPath(dir)

	if path == "" {
		return Default(), nil
	}

	return load(path)
}

// Compile compiles the configs Order and Ignore fields into compiled regex patterns.
func (c *Config) Compile() error {
	var err, errs error

	c.patterns, err = compile(c.Order)
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

	var cfg Config
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
// Returns -1 for unknown blocks if UnknownFirst is true, or len(patterns) if false.
func (c *Config) GetOrder(blockType, resourceType string) int {
	key := blockType + "." + resourceType

	for i, pattern := range c.patterns {
		if pattern.MatchString(key) {
			return i
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
