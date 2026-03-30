# helm-scribe

[![CI](https://github.com/Miosp/helm-scribe/actions/workflows/ci.yml/badge.svg)](https://github.com/Miosp/helm-scribe/actions/workflows/ci.yml)

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
| `--schema-file`     | `-s`  | Path to schema output file                     | Next to values file    |
| `--config`          | `-c`  | Path to config file                            | `.helm-scribe.yaml`    |
| `--truncate-length` | `-t`  | Max default value length before truncation     | `80`                   |
| `--dry-run`         | `-n`  | Print output to stdout instead of writing file | `false`                |
| `--no-pretty`       |       | Disable table column alignment                 | `false`                |
| `--heading-level`   |       | Heading level for section headers (1-6)        | `2`                    |
| `--schema-only`     |       | Only generate schema, skip README              | `false`                |
| `--readme-only`     |       | Only generate README, skip schema              | `false`                |
| `--type-column`     |       | Show type column in README table               | `false`                |
| `--strict`          |       | Treat warnings as errors (exit code 2)         | `false`                |

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
  # @enum [Always, IfNotPresent, Never]
  pullPolicy: IfNotPresent

# @section Network parameters

# Service port
# @min 1
# @max 65535
port: 80

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

# App name
# @pattern ^[a-z][a-z0-9-]*$
# @example my-custom-app
appName: my-app

# Old setting
# @deprecated Use newSetting instead
oldSetting: true

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
- **`@enum [val1, val2, ...]`**: Restricts the value to one of the listed options. Values are type-converted to match the field type (e.g., integers for integer fields). Quoted values are supported: `@enum ["val 1", "val 2"]`.
- **`@min <n>` / `@max <n>`**: Sets minimum/maximum constraints for numeric fields. Maps to `minimum`/`maximum` in JSON Schema.
- **`@pattern <regex>`**: Validates string values against a regular expression. Maps to `pattern` in JSON Schema.
- **`@default <value>`**: Overrides the displayed default value in the README table. Does not affect the schema (which uses the actual YAML value).
- **`@deprecated [message]`**: Marks a parameter as deprecated. Adds `(DEPRECATED)` prefix in the README table and sets `"deprecated": true` in the schema. The message is optional.
- **`@example <value>`**: Provides an example value. Maps to `"examples": [...]` in the schema.

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
| `object?`   | Nullable object      | `{"type": ["object", "null"], "properties": ...}`             |

The `?` modifier works on all types, including `object`. A nullable object with children retains its `properties` constraint but also accepts `null`. Object properties can independently be nullable:

```yaml
# @type object?
service:
  # @type string?
  description:
  port: 80
```

This produces a schema where `service` itself can be null, and `service.description` can be either a string or null, while `service.port` must be an integer.

## Schema generation

The generated `values.schema.json` is placed next to the values file by default. Use `--schema-file` to override the output path.

The schema is a self-contained JSON Schema draft-07 document with no `$ref` or `$defs`. This ensures compatibility with Helm's built-in schema validator and Artifact Hub.

A property is marked as required when any of the following hold:

- It has an explicit non-null default value (e.g., `replicaCount: 1`, `debug: false`, `name: ""`)
- It is an object with children, and at least one descendant is required (recursively)

A property is **not** required when:

- Its type is nullable (`?` suffix)
- Its default value is null and it has no children
- It is an object whose descendants are all non-required (e.g., all null without `@type`)

Note that zero-values (`false`, `0`, `""`) count as explicit defaults, so fields with these defaults are required. This matches the expectation that a Helm chart defines these values intentionally.

Values that are `null` without an explicit `@type` annotation produce an unconstrained schema (no `type` field, accepts any value) and a warning on stderr. Use `@type` to specify the intended type for null-valued fields.

The generated schema does not set `additionalProperties: false`, so extra properties not defined in `values.yaml` are accepted. This is intentional: Helm passes values through to subcharts, and strict schemas would reject subchart values.

## Limitations

- `@item` paths are split on `.` separators. YAML keys containing literal dots are not supported in `@item` path expressions.
- Arrays without a `@type` or `@item` annotation produce no `items` constraint in the schema (a warning is printed).

## Configuration file

You can place a `.helm-scribe.yaml` in your chart directory:

```yaml
truncateLength: 80
headingLevel: 2
valuesFile: values.yaml
readmeFile: README.md
schemaFile: values.schema.json
typeColumn: false
strict: false
```

CLI flags override config file values.

## Exit codes

| Code | Meaning |
|------|---------|
| 0    | Success |
| 1    | Error (parse failure, missing files, etc.) |
| 2    | Warnings present and `--strict` enabled |

Warnings are always printed to stderr. With `--strict`, the tool still generates all output files before exiting with code 2, so CI pipelines can inspect the results.

## Output example

Given the annotated `values.yaml` above, helm-scribe generates a README table:

```markdown
## Common parameters

| Key            | Description                           | Default |
|----------------|---------------------------------------|---------|
| `replicaCount` | Number of replicas for the deployment | `1`     |

## Image parameters

| Key                | Description       | Default          |
|--------------------|-------------------|------------------|
| `image.repository` | Image repository  | `"nginx"`        |
| `image.pullPolicy` | Image pull policy | `"IfNotPresent"` |

## Network parameters

| Key                  | Description                | Default          |
|----------------------|----------------------------|------------------|
| `port`               | Service port               | `80`             |
| `appName`            | App name                   | `"my-app"`       |
| `oldSetting`         | (DEPRECATED) Old setting   | `true`           |
```

With `--type-column`, an additional column is included:

```markdown
| Key            | Type      | Description                           | Default |
|----------------|-----------|---------------------------------------|---------|
| `replicaCount` | `integer` | Number of replicas for the deployment | `1`     |
```

And a `values.schema.json` with type definitions, nullable types, array item schemas, required field constraints, enum/min/max/pattern validation, and deprecation markers.

## Releasing

1. Tag a commit on master: `git tag v0.1.0`
2. Push the tag: `git push origin v0.1.0`
3. GitHub Actions builds cross-platform binaries and creates a draft release
4. Review the draft release on GitHub and publish it when ready
