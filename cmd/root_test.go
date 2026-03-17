//nolint:testpackage // testing internal run function
package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thespags/tfsortplus/internal/sorter"
)

func TestRun_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(`variable "test" {}`), 0o600))

	processor := sorter.NewProcessor()
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := run(processor, dir)(cmd, nil)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "0 changed, 1 unchanged")
}

func TestRun_ConfigError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".tfsortplus.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte("order:\n  - '['"), 0o600)) // invalid regex

	processor := sorter.NewProcessor()
	cmd := &cobra.Command{}

	err := run(processor, dir)(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error getting config file")
}

func TestRun_ProcessError(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nonexistent")

	processor := sorter.NewProcessor()
	cmd := &cobra.Command{}

	err := run(processor, dir)(cmd, nil)

	require.Error(t, err)
	assert.ErrorContains(t, err, "error processing terraform files")
}

func TestRun_PrintError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tfFile := filepath.Join(dir, "main.tf")
	content := `resource "aws" "b" {}

resource "aws" "a" {}
`
	require.NoError(t, os.WriteFile(tfFile, []byte(content), 0o600))

	configFile := filepath.Join(dir, ".tfsortplus.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte("order:\n  - ^resource\\.aws$\nalphabeticalTies: true"), 0o600))

	processor := sorter.NewProcessor()
	processor.Check = true // check mode returns error when files are unsorted
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := run(processor, dir)(cmd, nil)

	require.ErrorContains(t, err, "error printing output")
	assert.Contains(t, buf.String(), "not sorted")
}

func TestRootCmd(t *testing.T) {
	t.Parallel()

	cmd := RootCmd(".")

	assert.NotNil(t, cmd)
}
