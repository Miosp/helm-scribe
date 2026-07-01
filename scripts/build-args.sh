#!/usr/bin/env bash
# Emits helm-scribe CLI flags from INPUT_* environment variables, one token per line.
#
# String inputs  -> "--flag\nvalue" when the value is non-empty.
# Boolean inputs -> "--flag" when the value equals "true".
#
# The positional chart directory is NOT emitted here; the caller appends it.
set -euo pipefail

args=()

add_string() {
  local flag="$1" val="$2"
  if [ -n "$val" ]; then args+=("$flag" "$val"); fi
}

add_bool() {
  local flag="$1" val="$2"
  if [ "$val" = "true" ]; then args+=("$flag"); fi
}

add_string --values-file     "${INPUT_VALUES_FILE:-}"
add_string --readme-file     "${INPUT_README_FILE:-}"
add_string --config          "${INPUT_CONFIG:-}"
add_string --truncate-length "${INPUT_TRUNCATE_LENGTH:-}"
add_string --heading-level   "${INPUT_HEADING_LEVEL:-}"
add_string --schema-file     "${INPUT_SCHEMA_FILE:-}"
add_bool   --dry-run         "${INPUT_DRY_RUN:-false}"
add_bool   --no-pretty       "${INPUT_NO_PRETTY:-false}"
add_bool   --schema-only     "${INPUT_SCHEMA_ONLY:-false}"
add_bool   --readme-only     "${INPUT_README_ONLY:-false}"
add_bool   --strict          "${INPUT_STRICT:-false}"
add_bool   --type-column     "${INPUT_TYPE_COLUMN:-false}"

[ "${#args[@]}" -gt 0 ] && printf '%s\n' "${args[@]}"
exit 0
