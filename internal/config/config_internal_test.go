package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrder(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Order: []string{
			`module\.group`,   // exact match (auto-anchored)
			`data\.gitlab_.*`, // prefix match
			`resource\.gitlab_.*`,
			`resource\..*`,
			`module\..*`,
		},
		UnknownFirst: false,
	}
	require.NoError(t, cfg.Compile())

	tests := []struct {
		blockType    string
		resourceType string
		expected     int
	}{
		{"module", "group", 0},
		{"data", "gitlab_group", 1},
		{"data", "gitlab_project", 1},
		{"resource", "gitlab_project", 2},
		{"resource", "gitlab_branch_protection", 2},
		{"resource", "aws_instance", 3},
		{"resource", "google_compute_instance", 3},
		{"module", "vpc", 4},
		{"module", "network", 4},
		{"locals", "", 5},    // unknown, goes after (len=5)
		{"terraform", "", 5}, // unknown
	}

	for _, tt := range tests {
		t.Run(tt.blockType+"."+tt.resourceType, func(t *testing.T) {
			t.Parallel()

			got := cfg.GetOrder(tt.blockType, tt.resourceType)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGetOrderUnknownBefore(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Order: []string{
			`resource\.gitlab_.*`,
		},
		UnknownFirst: true,
	}
	require.NoError(t, cfg.Compile())

	// Known pattern
	assert.Equal(t, 0, cfg.GetOrder("resource", "gitlab_project"))

	// Unknown - should be -1 (before)
	assert.Equal(t, -1, cfg.GetOrder("resource", "aws_instance"))
}

func TestLoad(t *testing.T) {
	t.Parallel()

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		configPath := filepath.Join(dir, ".tfsortplus.yaml")

		content := `
order:
  - module\.group
  - data\.gitlab_.*
  - resource\.gitlab_.*

ignore:
  - .*\.generated\.tf
  - vendor/.*

alphabeticalTies: true
unknownFirst: true
`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))

		cfg, err := load(configPath)
		require.NoError(t, err)

		assert.Len(t, cfg.Order, 3)
		assert.Len(t, cfg.Ignore, 2)
		assert.True(t, cfg.AlphabeticalTies)
		assert.True(t, cfg.UnknownFirst)
		assert.Equal(t, 2, cfg.GetOrder("resource", "gitlab_project"))
	})
	t.Run("missing config", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		configPath := filepath.Join(dir, ".tfsortplus.yaml")

		_, err := load(configPath)

		require.ErrorContains(t, err, "failed to read config file:")
	})
	t.Run("invalid yaml", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		configPath := filepath.Join(dir, ".tfsortplus.yaml")

		content := `
gibberish
foo
bar
`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))

		_, err := load(configPath)

		require.ErrorContains(t, err, "failed to unmarshal config file:")
	})
	t.Run("invalid regex", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		configPath := filepath.Join(dir, ".tfsortplus.yaml")

		content := `
order:
 - "["
`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))

		_, err := load(configPath)

		require.ErrorContains(t, err, "failed to compile patterns: invalid pattern")
	})
}

func TestFindPath(t *testing.T) {
	t.Parallel()

	t.Run("empty dir", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		path := findPath(dir)

		assert.Empty(t, path)
	})
	t.Run("with config file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		configPath := filepath.Join(dir, ".tfsortplus.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("order: []"), 0o600))

		path := findPath(dir)

		assert.Equal(t, configPath, path)
	})
}

func TestGetConfig(t *testing.T) {
	t.Parallel()

	t.Run("with config file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		configPath := filepath.Join(dir, ".tfsortplus.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("alphabeticalTies: false"), 0o600))

		cfg, err := GetConfig(dir)

		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.False(t, cfg.ShouldSortAlphabetically())
	})
	t.Run("no config defaults", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		cfg, err := GetConfig(dir)

		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.True(t, cfg.ShouldSortAlphabetically())
	})
}

func TestDefault(t *testing.T) {
	t.Parallel()

	cfg := Default()

	assert.True(t, cfg.AlphabeticalTies)
	assert.False(t, cfg.UnknownFirst)
	assert.Empty(t, cfg.Order)
}

func TestShouldIgnore(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Ignore: []string{
			`.*\.generated\.tf`,
			`vendor/.*`,
			`_.*`,
		},
	}
	require.NoError(t, cfg.Compile())

	tests := []struct {
		path     string
		expected bool
	}{
		{"main.tf", false},
		{"resources.generated.tf", true},
		{"foo.generated.tf", true},
		{"vendor/module.tf", true},
		{"vendor/nested/file.tf", true},
		{"_override.tf", true},
		{"_test.tf", true},
		{"test_file.tf", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			got := cfg.ShouldIgnore(tt.path)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestAnchorPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"foo", "^foo$"},
		{"^foo", "^foo$"},
		{"foo$", "^foo$"},
		{"^foo$", "^foo$"},
		{".*", "^.*$"},
		{"^resource\\..*", "^resource\\..*$"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := anchorPattern(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
