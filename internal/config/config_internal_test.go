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
		LocalsFirst:  true,
		OutputLast:   true,
	}
	require.NoError(t, cfg.Compile())

	tests := []struct {
		blockType    string
		resourceType string
		expected     int
	}{
		{"locals", "", 0},                 // default first
		{"module", "group", 1},            // user pattern
		{"data", "gitlab_group", 2},       // user pattern
		{"data", "gitlab_project", 2},     // user pattern
		{"resource", "gitlab_project", 3}, // user pattern
		{"resource", "gitlab_branch_protection", 3},
		{"resource", "aws_instance", 4}, // user pattern
		{"resource", "google_compute_instance", 4},
		{"module", "vpc", 5},       // user pattern
		{"module", "network", 5},   // user pattern
		{"terraform", "", 6},       // unknown
		{"output", "my_output", 7}, // default last (after unknowns)
	}

	for _, tt := range tests {
		t.Run(tt.blockType+"."+tt.resourceType, func(t *testing.T) {
			t.Parallel()

			got := cfg.GetOrder(tt.blockType, tt.resourceType)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGetOrderLabellessBlocks(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Order: []string{
			`locals`,
			`terraform`,
			`module\..*`,
		},
		LocalsFirst: false,
		OutputLast:  true,
	}
	require.NoError(t, cfg.Compile())

	tests := []struct {
		blockType    string
		resourceType string
		expected     int
	}{
		{"locals", "", 0}, // explicit in user config (not re-prepended)
		{"terraform", "", 1},
		{"module", "vpc", 2},
		{"resource", "aws_instance", 3}, // unknown
		{"output", "foo", 4},            // default last (after unknowns)
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
		LocalsFirst:  true,
		OutputLast:   true,
	}
	require.NoError(t, cfg.Compile())

	// Default first
	assert.Equal(t, 0, cfg.GetOrder("locals", ""))

	// Known pattern (shifted by prepended default)
	assert.Equal(t, 1, cfg.GetOrder("resource", "gitlab_project"))

	// Default last (after unknowns)
	assert.Equal(t, 3, cfg.GetOrder("output", "foo"))

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
		assert.True(t, cfg.LocalsFirst)
		assert.True(t, cfg.OutputLast)
		assert.Equal(t, 3, cfg.GetOrder("resource", "gitlab_project"))
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
	assert.True(t, cfg.LocalsFirst)
	assert.True(t, cfg.OutputLast)
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
