# tfsortplus

A CLI tool for sorting Terraform blocks (resources, data, modules, etc.) according to a configurable order.

## Features

- Sort Terraform blocks by custom regex-based ordering rules
- Support for `.tf`, `.hcl`, and `.tofu` files
- Recursive directory processing
- CI-friendly check mode with diff output
- Alphabetical sorting for blocks with the same priority
- Configurable file ignore patterns

## Installation

### Go

```bash
go install github.com/thespags/tfsortplus@latest
```

### From Releases

Download the latest binary from the [releases page](https://github.com/thespags/tfsortplus/releases).

## Usage

```bash
# Sort all .tf files in current directory
tfsortplus

# Sort recursively
tfsortplus --recursive

# Check if files are sorted (for CI)
tfsortplus --check

# Check with diff output
tfsortplus --check --diff
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--recursive` | `-r` | Process directories recursively |
| `--check` | | Check if files are sorted (exit 1 if not) |
| `--diff` | | Show diff of changes |
| `--version` | `-v` | Print version information |
| `--help` | `-h` | Print help |

## Configuration

Create a `.tfsortplus.yaml` file in your project directory:

```yaml
# Define the order of blocks using regex patterns
# Patterns are auto-anchored (wrapped with  and )
order:
  - module\.group          # exact match
  - data\.gitlab_.*        # all gitlab data sources
  - resource\.gitlab_.*    # all gitlab resources
  - resource\..*           # all other resources
  - module\..*             # all other modules

# Sort blocks with the same priority alphabetically
alphabetical_ties: true

# Put unmatched blocks first (true) or last (false)
unknown_first: false

# Regex patterns for files to skip
ignore:
  - .*\.generated\.tf
```

### Pattern Matching

Patterns match against the block type and labels joined with dots:
- `resource "aws_instance" "main"` → `resource.aws_instance.main`
- `data "aws_ami" "ubuntu"` → `data.aws_ami.ubuntu`
- `module "vpc"` → `module.vpc`
- `locals` → `locals`

`^` and `$` are automatically added to the beginning and end of each pattern.

### Example Configuration

For GitLab Terraform resources:

```yaml
order:
  # Module group first
  - "module\\.group"

  # Data sources
  - "data\\.gitlab_group"
  - "data\\.gitlab_user"
  - "data\\.gitlab_project"

  # Group resources
  - "resource\\.gitlab_group_membership"

  # Project resources
  - "resource\\.gitlab_project"

  # Project settings
  - "resource\\.gitlab_project_job_token_scopes"
  - "resource\\.gitlab_branch_protection"

  # Access/membership
  - "resource\\.gitlab_project_membership"
  - "resource\\.gitlab_deploy_key"

  # CI/CD
  - "resource\\.gitlab_pipeline_schedule"
  - "resource\\.gitlab_project_variable"

alphabetical_ties: true
unknown_first: false
```

## CI Integration

Add to your CI pipeline to ensure Terraform files stay sorted:

```yaml
# GitHub Actions
- name: Check Terraform sorting
  run: tfsortplus --check --diff
```

The `--check` flag exits with code 1 if any files need sorting, making it suitable for CI enforcement.

## Development

### Prerequisites

With mise,
`mise install`

### Build

```bash
go build -o tfsortplus .
```

### Test

```bash
go test -v ./...
```

### Lint

```bash
golangci-lint run
```

## License

See [LICENSE](LICENSE) for details.
