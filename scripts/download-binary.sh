#!/usr/bin/env bash
# Downloads and extracts the helm-scribe binary for the current runner.
#
# Env:
#   TAG         release tag to download, e.g. v0.3.1
#   REPO        owner/name, e.g. Miosp/helm-scribe
#   RUNNER_OS   Linux | macOS | Windows   (read by archive-name.sh)
#   RUNNER_ARCH X64 | ARM64               (read by archive-name.sh)
#   DEST_DIR    directory to download into and extract into
#   GH_TOKEN    token for the gh CLI
#
# Prints the path to the extracted binary on stdout.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
archive="$("${here}/archive-name.sh")"

mkdir -p "${DEST_DIR}"
gh release download "${TAG}" \
  --repo "${REPO}" \
  --pattern "${archive}" \
  --dir "${DEST_DIR}" \
  --clobber

case "${archive}" in
  *.tar.gz) tar -xzf "${DEST_DIR}/${archive}" -C "${DEST_DIR}" ;;
  *.zip)    unzip -o "${DEST_DIR}/${archive}" -d "${DEST_DIR}" >/dev/null ;;
  *) echo "unknown archive format: ${archive}" >&2; exit 1 ;;
esac

bin="${DEST_DIR}/helm-scribe"
[ "${RUNNER_OS}" = "Windows" ] && bin="${DEST_DIR}/helm-scribe.exe"

if [ ! -f "${bin}" ]; then
  echo "binary not found after extraction: ${bin}" >&2
  exit 1
fi
chmod +x "${bin}" 2>/dev/null || true
echo "${bin}"
