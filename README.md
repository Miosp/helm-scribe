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
3. GitHub Actions builds cross-platform binaries and creates a draft release
4. Review the draft release on GitHub and publish it when ready
