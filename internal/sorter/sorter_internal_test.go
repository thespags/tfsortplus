package sorter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thespags/tfsortplus/internal/config"
)

func TestSortFile(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir("testdata")
	require.NoError(t, err)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		caseName := entry.Name()

		t.Run(caseName, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join("testdata", caseName)
			cfg := loadTestConfig(t, dir)
			sorter := NewSorter(cfg)

			input, err := os.ReadFile(filepath.Join(dir, "input.tf"))
			require.NoError(t, err)

			expected, err := os.ReadFile(filepath.Join(dir, "expected.tf"))
			require.NoError(t, err)

			result := sorter.SortFile("test.tf", input)
			require.NoError(t, result.Error)

			shouldChange := string(expected) != string(input)
			assert.Equal(t, shouldChange, result.Changed)
			assert.Equal(t, string(expected), string(result.Content))
		})
	}
}

func TestSortFile_EmptyFile(t *testing.T) {
	t.Parallel()

	s := NewSorter(config.Default())
	result := s.SortFile("test.tf", []byte{})

	require.NoError(t, result.Error)
	assert.False(t, result.Changed)
}

func TestSortFile_InvalidHCL(t *testing.T) {
	t.Parallel()

	s := NewSorter(config.Default())
	result := s.SortFile("test.tf", []byte(`resource "test" "x" {
  name = "test"
  # missing closing brace
`))

	assert.Error(t, result.Error)
}

func loadTestConfig(t *testing.T, dir string) *config.Config {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		return config.Default()
	}

	var cfg config.Config

	require.NoError(t, yaml.Unmarshal(data, &cfg))
	require.NoError(t, cfg.Compile())

	return &cfg
}
