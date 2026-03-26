# helm-scribe

A CLI tool that generates a parameters table and JSON Schema from Helm chart `values.yaml` files.

## How it works

helm-scribe parses your `values.yaml`, extracts parameter metadata from YAML comments, and produces:

1. A formatted markdown table inserted into your `README.md` between marker comments
2. A `values.schema.json` file for Helm value validation (JSON Schema draft-07)

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
# Run in a Helm chart directory (generates both README table and schema)
helm-scribe

# Specify a chart directory
helm-scribe ./my-chart

# Preview output without modifying files
helm-scribe --dry-run

# Generate only the schema
helm-scribe --schema-only

# Generate only the README table
helm-scribe --readme-only
```

### Flags

| Flag                | Short | Description                                    | Default                |
|---------------------|-------|------------------------------------------------|------------------------|
| `--values-file`     | `-v`  | Path to values file                            | `values.yaml`          |
| `--readme-file`     | `-r`  | Path to README file                            | `README.md`            |
| `--schema-file`     | `-s`  | Path to schema output file                     | `values.schema.json`   |
| `--config`          | `-c`  | Path to config file                            | `.helm-scribe.yaml`    |
| `--truncate-length` | `-t`  | Max default value length before truncation     | `80`                   |
| `--dry-run`         | `-n`  | Print output to stdout instead of writing file | `false`                |
| `--no-pretty`       |       | Disable table column alignment                 | `false`                |
| `--heading-level`   |       | Heading level for section headers (1-6)        | `2`                    |
| `--schema-only`     |       | Only generate schema, skip README              | `false`                |
| `--readme-only`     |       | Only generate README, skip schema              | `false`                |

## Annotating values.yaml

Add comments above your values to provide descriptions and type information:

```yaml
# @section Common parameters

# Number of replicas for the deployment
replicaCount: 1

# @section Image parameters

# Container image configuration
image:
  # Image repository
  repository: nginx
  # Image pull policy
  # @type string
  pullPolicy: IfNotPresent

# @section Network parameters

# Optional service description
# @type string?
serviceDescription:

# Allowed tags
# @type string[]
tags: []

# List of ingress hosts
# @item host: string
# @item paths: object[]
# @item paths[].path: string
# @item paths[].pathType: string
hosts: []

# @section Internal

# Internal setting, not user-facing
# @skip
reconcileInterval: 30s
```

### Supported annotations

- **Description**: Any comment line above a value becomes its description. Multi-line comments are joined with spaces. An empty comment line (`#`) creates a line break.
- **`@section <name>`**: Groups subsequent parameters under a named section heading.
- **`@skip`**: Excludes the parameter from generated output.
- **`@type <type>`**: Overrides the inferred type. Useful when the YAML value doesn't reflect the intended type (e.g., a null value that should be a string).
- **`@item <path>: <type>`**: Defines the shape of items in an object array. Implies `object[]` type if no `@type` is set.

### Type system

Scalar types: `string`, `integer`, `number`, `boolean`, `object`

Modifiers:

| Syntax      | Meaning              | Schema output                                      |
|-------------|----------------------|----------------------------------------------------|
| `string`    | Plain type           | `{"type": "string"}`                               |
| `string?`   | Nullable             | `{"type": ["string", "null"]}`                     |
| `string[]`  | Array of type        | `{"type": "array", "items": {"type": "string"}}`              |
| `string[]?` | Nullable array       | `{"type": ["array", "null"], "items": {"type": "string"}}`    |
| `string?[]` | Array of nullable    | `{"type": "array", "items": {"type": ["string", "null"]}}`    |
| `string?[]?`| Both nullable        | `{"type": ["array", "null"], "items": {"type": ["string", "null"]}}` |
| `object[]`  | Array of objects     | Use with `@item` to define item properties                    |

## Schema generation

The generated `values.schema.json` is a self-contained JSON Schema draft-07 document with no `$ref` or `$defs`. This ensures compatibility with Helm's built-in schema validator and Artifact Hub.

A property is marked as required unless its default value is null or its type is nullable (`?` suffix).

Values that are `null` without an explicit `@type` annotation produce an empty schema (`{}`, accepting any value) and a warning on stderr.

## Configuration file

You can place a `.helm-scribe.yaml` in your chart directory:

```yaml
truncateLength: 80
headingLevel: 2
valuesFile: values.yaml
readmeFile: README.md
schemaFile: values.schema.json
```

CLI flags override config file values.

## Output example

Given the annotated `values.yaml` above, helm-scribe generates a README table:

```markdown
## Common parameters

| Key              | Description                           | Default |
|------------------|---------------------------------------|---------|
| `replicaCount`   | Number of replicas for the deployment | `1`     |

## Image parameters

| Key                | Description        | Default         |
|--------------------|--------------------|-----------------|
| `image.repository` | Image repository   | `"nginx"`       |
| `image.pullPolicy` | Image pull policy  | `"IfNotPresent"`|
```

And a `values.schema.json` with type definitions, nullable types, array item schemas, and required field constraints.
