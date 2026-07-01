#!/usr/bin/env bash
# Selects the highest release tag matching a version constraint.
#
# Usage:  printf '%s\n' v0.3.1 v0.10.0 ... | select-version.sh <constraint>
#
# Constraint forms:
#   "" | "latest"  -> highest of all tags
#   "N"/"vN"       -> highest vN.*.* (major floats minor+patch)
#   "N.M"/"vN.M"   -> highest vN.M.* (patch floats)
#   "N.M.P"/"vN.M.P" -> that exact tag
#
# Candidate tags are read from stdin, one per line. Non-semver lines are ignored.
# Prints the selected tag to stdout. Exits 1 with a message on stderr if none match.
set -euo pipefail

constraint="${1:-latest}"
[ -z "$constraint" ] && constraint="latest"

mapfile -t raw
valid=()
for t in "${raw[@]}"; do
  [[ "$t" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] && valid+=("$t")
done
if [ "${#valid[@]}" -eq 0 ]; then
  echo "no released versions available" >&2
  exit 1
fi

highest() {
  # Reads tags on stdin, prints the highest by semver order.
  sed 's/^v//' | sort -V | tail -n 1 | sed 's/^/v/'
}

case "$constraint" in
  v[0-9]*.[0-9]*.[0-9]* | [0-9]*.[0-9]*.[0-9]*)
    want="v${constraint#v}"
    for t in "${valid[@]}"; do
      if [ "$t" = "$want" ]; then
        echo "$t"
        exit 0
      fi
    done
    echo "no release matches exact version ${constraint}" >&2
    exit 1
    ;;
  latest)
    printf '%s\n' "${valid[@]}" | highest
    ;;
  [0-9]* | [0-9]*.[0-9]* | v[0-9]* | v[0-9]*.[0-9]*)
    escaped="$(printf '%s' "${constraint#v}" | sed 's/\./\\./g')"
    filter="^v${escaped}\\."
    matched=()
    for t in "${valid[@]}"; do
      [[ "$t" =~ $filter ]] && matched+=("$t")
    done
    if [ "${#matched[@]}" -eq 0 ]; then
      echo "no release matches constraint ${constraint}" >&2
      exit 1
    fi
    printf '%s\n' "${matched[@]}" | highest
    ;;
  *)
    echo "invalid version constraint: ${constraint}" >&2
    exit 1
    ;;
esac
