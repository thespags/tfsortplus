package sorter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thespags/tfsortplus/internal/config"
)

func TestSorter_SortFile_AlreadySorted(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Order: []string{
			`data\.gitlab_.*`,
			`resource\.gitlab_.*`,
		},
		AlphabeticalTies: true,
		UnknownFirst:     false,
	}
	require.NoError(t, cfg.Compile())

	sorter := NewSorter(cfg)

	content := []byte(`data "gitlab_group" "parent" {
  full_path = "example"
}

resource "gitlab_project" "test" {
  name = "test"
}
`)

	result := sorter.SortFile("test.tf", content)

	require.NoError(t, result.Error)
	assert.False(t, result.Changed)
	assert.Equal(t, content, result.Content)
}

func TestSorter_SortFile_NeedsSorting(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Order: []string{
			`data\.gitlab_.*`,
			`resource\.gitlab_project`,
			`resource\.gitlab_branch_protection`,
		},
		AlphabeticalTies: true,
		UnknownFirst:     false,
	}
	require.NoError(t, cfg.Compile())

	sorter := NewSorter(cfg)

	input := []byte(`resource "gitlab_branch_protection" "main" {
  branch = "main"
}

resource "gitlab_project" "test" {
  name = "test"
}
`)

	expected := []byte(`resource "gitlab_project" "test" {
  name = "test"
}

resource "gitlab_branch_protection" "main" {
  branch = "main"
}
`)

	result := sorter.SortFile("test.tf", input)

	require.NoError(t, result.Error)
	assert.True(t, result.Changed)
	assert.Equal(t, string(expected), string(result.Content))
}

func TestSorter_SortFile_WithRegex(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Order: []string{
			`data\..*`,
			`resource\.gitlab_.*`,
			`resource\..*`,
		},
		AlphabeticalTies: true,
		UnknownFirst:     false,
	}
	require.NoError(t, cfg.Compile())

	sorter := NewSorter(cfg)

	input := []byte(`resource "aws_instance" "example" {
  ami = "ami-123"
}

resource "gitlab_project" "test" {
  name = "test"
}

data "gitlab_group" "parent" {
  full_path = "example"
}
`)

	expected := []byte(`data "gitlab_group" "parent" {
  full_path = "example"
}

resource "gitlab_project" "test" {
  name = "test"
}

resource "aws_instance" "example" {
  ami = "ami-123"
}
`)

	result := sorter.SortFile("test.tf", input)

	require.NoError(t, result.Error)
	assert.True(t, result.Changed)
	assert.Equal(t, string(expected), string(result.Content))
}

func TestSorter_SortFile_AlphabeticalTies(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Order: []string{
			`resource\.gitlab_project`,
		},
		AlphabeticalTies: true,
		UnknownFirst:     false,
	}
	require.NoError(t, cfg.Compile())

	sorter := NewSorter(cfg)

	input := []byte(`resource "gitlab_project" "zebra" {
  name = "zebra"
}

resource "gitlab_project" "alpha" {
  name = "alpha"
}
`)

	expected := []byte(`resource "gitlab_project" "alpha" {
  name = "alpha"
}

resource "gitlab_project" "zebra" {
  name = "zebra"
}
`)

	result := sorter.SortFile("test.tf", input)

	require.NoError(t, result.Error)
	assert.True(t, result.Changed)
	assert.Equal(t, string(expected), string(result.Content))
}

func TestSorter_SortFile_NoAlphabeticalTies(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Order: []string{
			`resource\.gitlab_project`,
		},
		AlphabeticalTies: false,
		UnknownFirst:     false,
	}
	require.NoError(t, cfg.Compile())

	sorter := NewSorter(cfg)

	// When alphabetical ties is false, original order is preserved for same-order blocks
	input := []byte(`resource "gitlab_project" "zebra" {
  name = "zebra"
}

resource "gitlab_project" "alpha" {
  name = "alpha"
}
`)

	result := sorter.SortFile("test.tf", input)

	require.NoError(t, result.Error)
	assert.False(t, result.Changed) // Order preserved
}

func TestSorter_SortFile_UnknownBefore(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Order: []string{
			`resource\.gitlab_.*`,
		},
		AlphabeticalTies: true,
		UnknownFirst:     true,
	}
	require.NoError(t, cfg.Compile())

	sorter := NewSorter(cfg)

	input := []byte(`resource "gitlab_project" "test" {
  name = "test"
}

resource "aws_instance" "example" {
  ami = "ami-123"
}
`)

	expected := []byte(`resource "aws_instance" "example" {
  ami = "ami-123"
}

resource "gitlab_project" "test" {
  name = "test"
}
`)

	result := sorter.SortFile("test.tf", input)

	require.NoError(t, result.Error)
	assert.True(t, result.Changed)
	assert.Equal(t, string(expected), string(result.Content))
}

func TestSorter_SortFile_PreservesHeader(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Order: []string{
			`resource\.gitlab_project`,
			`resource\.gitlab_branch_protection`,
		},
		AlphabeticalTies: true,
		UnknownFirst:     false,
	}
	require.NoError(t, cfg.Compile())

	sorter := NewSorter(cfg)

	input := []byte(`# This is a header comment
# with multiple lines

resource "gitlab_branch_protection" "main" {
  branch = "main"
}

resource "gitlab_project" "test" {
  name = "test"
}
`)

	expected := []byte(`# This is a header comment
# with multiple lines

resource "gitlab_project" "test" {
  name = "test"
}

resource "gitlab_branch_protection" "main" {
  branch = "main"
}
`)

	result := sorter.SortFile("test.tf", input)

	require.NoError(t, result.Error)
	assert.True(t, result.Changed)
	assert.Equal(t, string(expected), string(result.Content))
}

func TestSorter_SortFile_EmptyFile(t *testing.T) {
	t.Parallel()

	sorter := NewSorter(nil)

	result := sorter.SortFile("test.tf", []byte{})

	require.NoError(t, result.Error)
	assert.False(t, result.Changed)
}

func TestSorter_SortFile_InvalidHCL(t *testing.T) {
	t.Parallel()

	sorter := NewSorter(nil)

	input := []byte(`resource "gitlab_project" "test" {
  name = "test"
  # missing closing brace
`)

	result := sorter.SortFile("test.tf", input)

	assert.Error(t, result.Error)
}

func TestSorter_SortFile_Module(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Order: []string{
			`module\.group`,
			`resource\.gitlab_.*`,
			`module\..*`,
		},
		AlphabeticalTies: true,
		UnknownFirst:     false,
	}
	require.NoError(t, cfg.Compile())

	sorter := NewSorter(cfg)

	input := []byte(`module "vpc" {
  source = "./modules/vpc"
}

resource "gitlab_project" "test" {
  name = "test"
}

module "group" {
  source = "./modules/group"
}
`)

	expected := []byte(`module "group" {
  source = "./modules/group"
}

resource "gitlab_project" "test" {
  name = "test"
}

module "vpc" {
  source = "./modules/vpc"
}
`)

	result := sorter.SortFile("test.tf", input)

	require.NoError(t, result.Error)
	assert.True(t, result.Changed)
	assert.Equal(t, string(expected), string(result.Content))
}

func TestSorter_SortFile_PreservesBlockComments(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Order: []string{
			`data\.gitlab_.*`,
			`resource\.gitlab_.*`,
		},
		AlphabeticalTies: true,
		UnknownFirst:     false,
	}
	require.NoError(t, cfg.Compile())

	sorter := NewSorter(cfg)

	input := []byte(`# This is a header comment

# Resource comment
resource "gitlab_project" "test" {
  name = "test"
}

# Data source comment
# with multiple lines
data "gitlab_group" "parent" {
  full_path = "example"
}
`)

	expected := []byte(`# This is a header comment

# Data source comment
# with multiple lines
data "gitlab_group" "parent" {
  full_path = "example"
}

# Resource comment
resource "gitlab_project" "test" {
  name = "test"
}
`)

	result := sorter.SortFile("test.tf", input)

	require.NoError(t, result.Error)
	assert.True(t, result.Changed)
	assert.Equal(t, string(expected), string(result.Content))
}

func TestSorter_SortFile_InlineComments(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Order: []string{
			`data\..*`,
			`resource\..*`,
		},
		AlphabeticalTies: true,
		UnknownFirst:     false,
	}
	require.NoError(t, cfg.Compile())

	sorter := NewSorter(cfg)

	input := []byte(`resource "aws_instance" "example" {
  ami = "ami-123" # AMI ID
  instance_type = "t2.micro"
}

data "aws_vpc" "main" {
  default = true # Use default VPC
}
`)

	expected := []byte(`data "aws_vpc" "main" {
  default = true # Use default VPC
}

resource "aws_instance" "example" {
  ami = "ami-123" # AMI ID
  instance_type = "t2.micro"
}
`)

	result := sorter.SortFile("test.tf", input)

	require.NoError(t, result.Error)
	assert.True(t, result.Changed)
	assert.Equal(t, string(expected), string(result.Content))
}
