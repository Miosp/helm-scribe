# helm-scribe

[![CI](https://github.com/Miosp/helm-scribe/actions/workflows/ci.yml/badge.svg)](https://github.com/Miosp/helm-scribe/actions/workflows/ci.yml)

A CLI tool that generates a parameters table and JSON Schema from Helm chart `values.yaml` files.

This tool was inspired by [helm-docs](https://github.com/norwoodj/helm-docs) and [readme-generator-for-helm](https://github.com/bitnami/readme-generator-for-helm), which I found helpful, but limited in some ways

## How does it work

helm-scribe parses your `values.yaml`, extracts parameter metadata from YAML comments, and produces:

1. A formatted markdown table inserted into your `README.md` between marker comments
2. A `values.schema.json` file for Helm value validation (JSON Schema draft-07)

Place these markers in your README where the table should appear:

```markdown
<!-- helm-scribe:start -->
<!-- helm-scribe:end -->
```

For more information check out the [wiki tab](https://github.com/Miosp/helm-scribe/wiki).

# Quickstart

## Installation

Pre-built binaries for Linux, macOS, and Windows (amd64/arm64) are available on the [GitHub Releases](https://github.com/Miosp/helm-scribe/releases) page.

Or, if you wish to install with go:

```sh
go install github.com/miosp/helm-scribe@latest
```

## Add README markers

Insert these where the parameters table should appear:

```markdown
<!-- helm-scribe:start -->
<!-- helm-scribe:end -->
```

## Annotate values.yaml

```yaml
# @section Network

# Service port
# @min 1
# @max 65535
port: 80

# Image pull policy
# @type string
# @enum [Always, IfNotPresent, Never]
pullPolicy: IfNotPresent

# Optional description
# @type string?
serviceDescription:

# Internal setting
# @skip
reconcileInterval: 30s
```

## Run

```sh
helm-scribe
```

This writes a parameters table into your README and generates `values.schema.json` next to `values.yaml`.

**README table:**

| Key | Description | Default |
|---|---|---|
| `port` | Service port | `80` |
| `pullPolicy` | Image pull policy | `"IfNotPresent"` |
| `serviceDescription` | Optional description | `null` |

`reconcileInterval` is excluded by `@skip`.

**Schema:** A JSON Schema draft-07 file with types, `enum`, `minimum`/`maximum`, nullable fields, and required-field logic. Helm validates values against this schema during `helm install` and `helm upgrade`.

# GitHub Action

Run helm-scribe in CI with the [`Miosp/helm-scribe`](https://github.com/marketplace/actions/helm-scribe) Action.

## Generate mode

Regenerate the table and schema, then let a later step commit or open a PR:

```yaml
- uses: actions/checkout@v4
- uses: Miosp/helm-scribe@v0
  with:
    chart-directory: charts/my-app
```

## Check mode

Fail a pull request when the generated files are out of date. Requires a checkout so the Action can diff the working tree:

```yaml
- uses: actions/checkout@v4
- uses: Miosp/helm-scribe@v0
  with:
    chart-directory: charts/my-app
    check: "true"
```

## Inputs

| Input             | Default | Description                                             |
| ----------------- | ------- | ------------------------------------------------------- |
| `chart-directory` | `.`     | Chart directory to process.                             |
| `values-file`     | unset   | Path to the values file (`--values-file`).              |
| `readme-file`     | unset   | Path to the README file (`--readme-file`).              |
| `config`          | unset   | Path to the config file (`--config`).                   |
| `truncate-length` | unset   | Max default length before truncation (`--truncate-length`). |
| `heading-level`   | unset   | Section heading level, 1-6 (`--heading-level`).         |
| `schema-file`     | unset   | Path to the schema output file (`--schema-file`).       |
| `dry-run`         | `false` | Print to stdout instead of writing (`--dry-run`).       |
| `no-pretty`       | `false` | Disable table alignment (`--no-pretty`).                |
| `schema-only`     | `false` | Only generate the schema (`--schema-only`).             |
| `readme-only`     | `false` | Only generate the README (`--readme-only`).             |
| `strict`          | `false` | Treat warnings as errors (`--strict`).                  |
| `type-column`     | `false` | Show the type column (`--type-column`).                 |
| `version`         | `0`     | Version constraint: `0`, `0.3`, `0.3.1`, or `latest`.   |
| `check`           | `false` | Fail when generated files drift.                        |
| `binary`          | unset   | Advanced: use a prebuilt binary instead of downloading. |

## Outputs

| Output          | Description                                          |
| --------------- | ---------------------------------------------------- |
| `version`       | Resolved release tag used (`local` with `binary`).   |
| `drift`         | `true`/`false`; whether check mode found stale files.|
| `changed-files` | Newline-separated list of files that drifted.        |
