package sorter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thespags/tfsortplus/internal/config"
)

func TestNewProcessor(t *testing.T) {
	t.Parallel()

	processor := NewProcessor()

	assert.NotNil(t, processor)
	assert.False(t, processor.Recursive)
	assert.False(t, processor.Check)
	assert.False(t, processor.Diff)
	assert.Equal(t, []string{".tf", ".hcl", ".tofu"}, processor.extensions)
	assert.Equal(t, []string{".git", ".terraform"}, processor.excludedDirs)
}

func TestProcessor_IsValidExtension(t *testing.T) {
	t.Parallel()

	processor := NewProcessor()

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"terraform file", "main.tf", true},
		{"hcl file", "config.hcl", true},
		{"tofu file", "main.tofu", true},
		{"uppercase tf", "MAIN.TF", true},
		{"mixed case", "Main.Tf", true},
		{"go file", "main.go", false},
		{"yaml file", "config.yaml", false},
		{"no extension", "README", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := processor.isValidExtension(tt.filename)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestProcessor_IsExcludedDir(t *testing.T) {
	t.Parallel()

	processor := NewProcessor()

	tests := []struct {
		name     string
		dirname  string
		expected bool
	}{
		{"git dir", ".git", true},
		{"terraform dir", ".terraform", true},
		{"modules dir", "modules", false},
		{"src dir", "src", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := processor.isExcludedDir(tt.dirname)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestProcessor_Process_SingleFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tfFile := filepath.Join(dir, "main.tf")

	content := `resource "aws_instance" "example" {
  ami = "ami-123"
}
`
	require.NoError(t, os.WriteFile(tfFile, []byte(content), 0o600))

	cfg := config.Default()

	processor := NewProcessor()
	processor.Cfg = cfg
	processor.Check = true // Don't write files

	results, err := processor.Process(dir)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, tfFile, results[0].Path)
	assert.False(t, results[0].Changed)
	assert.NoError(t, results[0].Error)
}

func TestProcessor_Process_MultipleFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	files := []string{"main.tf", "variables.tf", "outputs.tf"}
	for _, f := range files {
		content := `variable "test" {
  type = string
}
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte(content), 0o600))
	}

	cfg := config.Default()

	processor := NewProcessor()
	processor.Cfg = cfg
	processor.Check = true

	results, err := processor.Process(dir)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestProcessor_Process_SkipsNonTfFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create various file types
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte("variable \"x\" {}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# README"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("key: value"), 0o600))

	cfg := config.Default()

	processor := NewProcessor()
	processor.Cfg = cfg
	processor.Check = true

	results, err := processor.Process(dir)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results[0].Path, "main.tf")
}

func TestProcessor_Process_Recursive(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	subdir := filepath.Join(dir, "modules", "vpc")
	require.NoError(t, os.MkdirAll(subdir, 0o750))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte("variable \"x\" {}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "main.tf"), []byte("variable \"y\" {}"), 0o600))

	cfg := config.Default()

	// Non-recursive should only find root file
	processor := NewProcessor()
	processor.Cfg = cfg
	processor.Check = true
	processor.Recursive = false

	results, err := processor.Process(dir)
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// Recursive should find both
	processor.Recursive = true
	results, err = processor.Process(dir)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestProcessor_Process_SkipsExcludedDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	terraformDir := filepath.Join(dir, ".terraform")

	require.NoError(t, os.MkdirAll(gitDir, 0o750))
	require.NoError(t, os.MkdirAll(terraformDir, 0o750))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte("variable \"x\" {}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte(""), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(terraformDir, "main.tf"), []byte("variable \"y\" {}"), 0o600))

	cfg := config.Default()

	processor := NewProcessor()
	processor.Cfg = cfg
	processor.Check = true
	processor.Recursive = true

	results, err := processor.Process(dir)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results[0].Path, "main.tf")
}

func TestProcessor_Process_IgnorePatterns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte("variable \"x\" {}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "generated.tf"), []byte("variable \"y\" {}"), 0o600))

	cfg := &config.Config{
		Ignore: []string{`generated\.tf$`},
	}
	require.NoError(t, cfg.Compile())

	processor := NewProcessor()
	processor.Cfg = cfg
	processor.Check = true

	results, err := processor.Process(dir)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results[0].Path, "main.tf")
}

func TestProcessor_Process_WritesChanges(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tfFile := filepath.Join(dir, "main.tf")

	input := `resource "gitlab_project" "b" {
  name = "b"
}

resource "gitlab_project" "a" {
  name = "a"
}
`
	require.NoError(t, os.WriteFile(tfFile, []byte(input), 0o600))

	cfg := &config.Config{
		Order: []string{
			`^resource\.gitlab_project$`,
		},
		AlphabeticalTies: true,
	}
	require.NoError(t, cfg.Compile())

	processor := NewProcessor()
	processor.Cfg = cfg
	processor.Check = false

	results, err := processor.Process(dir)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Changed)

	// Verify file was modified
	modified, err := os.ReadFile(tfFile)
	require.NoError(t, err)
	assert.Contains(t, string(modified), "resource \"gitlab_project\" \"a\"")
}

func TestProcessor_Print_NoChanges(t *testing.T) {
	t.Parallel()

	processor := NewProcessor()
	processor.results = []*Result{
		{Path: "main.tf", Changed: false},
		{Path: "vars.tf", Changed: false},
	}

	output, err := processor.Print()
	require.NoError(t, err)
	assert.Contains(t, output, "0 changed")
	assert.Contains(t, output, "2 unchanged")
}

func TestProcessor_Print_WithChanges(t *testing.T) {
	t.Parallel()

	processor := NewProcessor()
	processor.results = []*Result{
		{Path: "main.tf", Changed: true, Original: []byte("a"), Content: []byte("b")},
		{Path: "vars.tf", Changed: false},
	}

	output, err := processor.Print()
	require.NoError(t, err)
	assert.Contains(t, output, "changed: main.tf")
	assert.Contains(t, output, "1 changed")
	assert.Contains(t, output, "1 unchanged")
}

func TestProcessor_Print_CheckMode(t *testing.T) {
	t.Parallel()

	processor := NewProcessor()
	processor.Check = true
	processor.results = []*Result{
		{Path: "main.tf", Changed: true},
		{Path: "vars.tf", Changed: false},
	}

	output, err := processor.Print()
	require.Error(t, err)
	assert.Contains(t, output, "not sorted: main.tf")
	assert.Contains(t, output, "1 not sorted")
	assert.Contains(t, output, "1 sorted")
}

func TestProcessor_Print_WithErrors(t *testing.T) {
	t.Parallel()

	processor := NewProcessor()
	processor.results = []*Result{
		{Path: "main.tf", Error: assert.AnError},
		{Path: "vars.tf", Changed: false},
	}

	output, err := processor.Print()
	require.Error(t, err)
	assert.Contains(t, output, "error: main.tf")
	assert.Contains(t, output, "1 errors")
}

func TestProcessor_Print_WithDiff(t *testing.T) {
	t.Parallel()

	processor := NewProcessor()
	processor.Diff = true
	processor.results = []*Result{
		{
			Path:     "main.tf",
			Changed:  true,
			Original: []byte("original content"),
			Content:  []byte("sorted content"),
		},
	}

	output, err := processor.Print()
	require.NoError(t, err)
	assert.Contains(t, output, "--- main.tf (original)")
	assert.Contains(t, output, "+++ main.tf (sorted)")
	assert.Contains(t, output, "original content")
	assert.Contains(t, output, "sorted content")
}

func TestProcessor_Process_ParseError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tfFile := filepath.Join(dir, "main.tf")

	// Invalid HCL
	content := `resource "aws_instance" "example" {
  name = "test
`
	require.NoError(t, os.WriteFile(tfFile, []byte(content), 0o600))

	cfg := config.Default()

	processor := NewProcessor()
	processor.Cfg = cfg
	processor.Check = true

	results, err := processor.Process(dir)
	require.NoError(t, err) // Process doesn't fail, just records error
	require.Len(t, results, 1)
	assert.Error(t, results[0].Error)
}
