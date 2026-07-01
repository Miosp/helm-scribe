#!/usr/bin/env bats

SCRIPT="${BATS_TEST_DIRNAME}/../archive-name.sh"

@test "linux amd64 uses tar.gz and strips leading v" {
  run env TAG=v0.3.1 RUNNER_OS=Linux RUNNER_ARCH=X64 "$SCRIPT"
  [ "$status" -eq 0 ]
  [ "$output" = "helm-scribe_0.3.1_linux_amd64.tar.gz" ]
}

@test "macos arm64 maps to darwin arm64" {
  run env TAG=v1.2.0 RUNNER_OS=macOS RUNNER_ARCH=ARM64 "$SCRIPT"
  [ "$status" -eq 0 ]
  [ "$output" = "helm-scribe_1.2.0_darwin_arm64.tar.gz" ]
}

@test "windows uses zip" {
  run env TAG=v0.3.1 RUNNER_OS=Windows RUNNER_ARCH=X64 "$SCRIPT"
  [ "$status" -eq 0 ]
  [ "$output" = "helm-scribe_0.3.1_windows_amd64.zip" ]
}

@test "unsupported arch fails" {
  run env TAG=v0.3.1 RUNNER_OS=Linux RUNNER_ARCH=ARM "$SCRIPT"
  [ "$status" -eq 1 ]
}
