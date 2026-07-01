#!/usr/bin/env bats

SCRIPT="${BATS_TEST_DIRNAME}/../select-version.sh"
TAGS=$'v0.1.0\nv0.3.0\nv0.3.1\nv0.10.0\nv1.0.0\nv1.2.0\nnightly'

run_select() { printf '%s\n' "$TAGS" | "$SCRIPT" "$1"; }

@test "latest picks the highest semver tag" {
  run run_select latest
  [ "$status" -eq 0 ]
  [ "$output" = "v1.2.0" ]
}

@test "empty constraint behaves like latest" {
  run run_select ""
  [ "$status" -eq 0 ]
  [ "$output" = "v1.2.0" ]
}

@test "bare major floats within that major" {
  run run_select 0
  [ "$status" -eq 0 ]
  [ "$output" = "v0.10.0" ]
}

@test "major.minor floats within that minor" {
  run run_select 0.3
  [ "$status" -eq 0 ]
  [ "$output" = "v0.3.1" ]
}

@test "bare major with a v prefix floats within that major" {
  run run_select v0
  [ "$status" -eq 0 ]
  [ "$output" = "v0.10.0" ]
}

@test "major.minor with a v prefix floats within that minor" {
  run run_select v0.3
  [ "$status" -eq 0 ]
  [ "$output" = "v0.3.1" ]
}

@test "exact version with v prefix returns that tag" {
  run run_select v0.3.0
  [ "$status" -eq 0 ]
  [ "$output" = "v0.3.0" ]
}

@test "exact version without v prefix returns that tag" {
  run run_select 1.0.0
  [ "$status" -eq 0 ]
  [ "$output" = "v1.0.0" ]
}

@test "unmatched exact version fails" {
  run run_select 9.9.9
  [ "$status" -eq 1 ]
}

@test "unmatched range fails" {
  run run_select 2
  [ "$status" -eq 1 ]
}
