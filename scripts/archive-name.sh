#!/usr/bin/env bash
# Prints the GoReleaser archive filename for a release tag on the current runner.
#
# Env:
#   TAG         release tag, e.g. v0.3.1
#   RUNNER_OS   Linux | macOS | Windows
#   RUNNER_ARCH X64 | ARM64
#
# Naming mirrors .goreleaser.yaml:
#   {ProjectName}_{Version}_{Os}_{Arch}.{ext}  with Version = tag without leading v.
set -euo pipefail

ver="${TAG#v}"

case "${RUNNER_OS}" in
  Linux)   os=linux;   ext=tar.gz ;;
  macOS)   os=darwin;  ext=tar.gz ;;
  Windows) os=windows; ext=zip ;;
  *) echo "unsupported OS: ${RUNNER_OS}" >&2; exit 1 ;;
esac

case "${RUNNER_ARCH}" in
  X64)   arch=amd64 ;;
  ARM64) arch=arm64 ;;
  *) echo "unsupported arch: ${RUNNER_ARCH}" >&2; exit 1 ;;
esac

echo "helm-scribe_${ver}_${os}_${arch}.${ext}"
