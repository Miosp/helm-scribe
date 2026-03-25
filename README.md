# helm-scribe

A CLI tool that generates a parameters table for Helm chart READMEs from annotated `values.yaml` files.

## How it works

helm-scribe parses your `values.yaml`, extracts parameter metadata from YAML comments, and inserts a formatted markdown table into your `README.md` between marker comments.

Place these markers in your README where the table should appear:

```markdown
<!-- helm-scribe:start -->
<!-- helm-scribe:end -->
```

## Installation

```sh
go install github.com/miosp/helm-scribe@latest
```

## Usage

```sh
# Run in a Helm chart directory
helm-scribe

# Specify a chart directory
helm-scribe ./my-chart

# Preview output without modifying files
helm-scribe --dry-run
```

### Flags

| Flag                  | Short | Description                                    | Default             |
|-----------------------|-------|------------------------------------------------|---------------------|
| `--values-file`       | `-v`  | Path to values file                            | `values.yaml`       |
| `--readme-file`       | `-r`  | Path to README file                            | `README.md`         |
| `--config`            | `-c`  | Path to config file                            | `.helm-scribe.yaml` |
| `--truncate-length`   | `-t`  | Max default value length before truncation     | `80`                |
| `--dry-run`           | `-n`  | Print output to stdout instead of writing file | `false`             |
| `--no-pretty`         |       | Disable table column alignment                 | `false`             |
| `--heading-level`     |       | Heading level for section headers (1-6)        | `2`                 |

## Annotating values.yaml

Add comments above your values to provide descriptions:

```yaml
# @section Common parameters

# Number of replicas for the deployment
replicaCount: 1

# @section Image parameters

# Container image configuration
image:
  # Image repository
  repository: nginx
  # Image tag
  tag: "latest"

# @section Internal

# Internal setting, not user-facing
# @skip
reconcileInterval: 30s
```

### Supported annotations

- **Description**: Any comment line above a value becomes its description.
- **`@section <name>`**: Groups subsequent parameters under a named section heading.
- **`@skip`**: Excludes the parameter from the generated table.

## Configuration file

You can place a `.helm-scribe.yaml` in your chart directory to set defaults:

```yaml
truncateLength: 80
headingLevel: 2
valuesFile: values.yaml
readmeFile: README.md
```

## Output

Given the annotated `values.yaml` above, helm-scribe generates:

```markdown
## Common parameters

| Key              | Description                           | Default  |
|------------------|---------------------------------------|----------|
| `replicaCount`   | Number of replicas for the deployment | `1`      |

## Image parameters

| Key                | Description      | Default    |
|--------------------|------------------|------------|
| `image.repository` | Image repository | `"nginx"`  |
| `image.tag`        | Image tag        | `"latest"` |
```

Parameters marked with `@skip` are excluded, and the `fullnameOverride` (with no description in this example) still appears with an empty description cell.
