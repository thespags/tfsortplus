package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thespags/tfsortplus/internal/config"
	"github.com/thespags/tfsortplus/internal/sorter"
)

// Build time variables are set using -ldflags.
var (
	version = "dev"
	commit  = "none"    //nolint:gochecknoglobals // Set by ldflags at build time
	date    = "unknown" //nolint:gochecknoglobals // Set by ldflags at build time
)

func versionString() string {
	if version != "" && commit != "none" && date != "unknown" {
		return fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
	}

	return version + " (build details not available)"
}

// RootCmd represents the base command.
func RootCmd(dir string) *cobra.Command {
	processor := sorter.NewProcessor()
	cmd := &cobra.Command{
		Use:   "tfsortplus [flags]",
		Short: "Sort Terraform blocks according to a configurable order",
		Long: `tfsortplus sorts Terraform blocks (resources, data, modules, etc.) according to
a configurable order defined in a YAML file.

Configuration file (.tfsortplus.yaml):
  order:
    - module\.group          # exact match (auto-anchored to ^...$)
    - data\.gitlab_.*        # all gitlab data sources
    - resource\.gitlab_.*    # all gitlab resources
    - resource\..*           # all other resources
    - module\..*             # all other modules

  alphabeticalTies: true     # sort same-order blocks alphabetically
  unknownFirst: false        # put unmatched blocks first (true) or last (false, default)

  ignore:
    - .*\.generated\.tf      # regex patterns for files to skip (auto-anchored)

The tool loads config from the current directory and processes .tf files.`,
		Example: `  # Sort all .tf files in current directory
  tfsortplus

  # Sort recursively
  tfsortplus --recursive

  # Check if files are sorted (for CI)
  tfsortplus --check

  # Check with diff output
  tfsortplus --check --diff`,
		Version: versionString(),
		RunE:    run(processor, dir),
	}

	cmd.Flags().BoolVarP(&processor.Recursive, "recursive", "r", false, "Process directories recursively")
	cmd.Flags().BoolVar(&processor.Check, "check", false, "Check if files are sorted (exit 1 if not, for CI)")
	cmd.Flags().BoolVar(&processor.Diff, "diff", false, "Show diff of changes")

	return cmd
}

func run(processor *sorter.Processor, dir string) func(cmd *cobra.Command, _ []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		var err error

		processor.Cfg, err = config.GetConfig(dir)
		if err != nil {
			return fmt.Errorf("error getting config file: %w", err)
		}

		_, err = processor.Process(dir)
		if err != nil {
			return fmt.Errorf("error processing terraform files: %w", err)
		}

		output, err := processor.Print()
		cmd.Println(output)

		if err != nil {
			return fmt.Errorf("error printing output: %w", err)
		}

		return nil
	}
}
