#!/usr/bin/env bats

SCRIPT="${BATS_TEST_DIRNAME}/../build-args.sh"

@test "no inputs produces no arguments" {
  run env -i "$SCRIPT"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "string inputs emit flag and value on separate lines" {
  run env -i INPUT_VALUES_FILE=custom.yaml INPUT_HEADING_LEVEL=3 "$SCRIPT"
  [ "$status" -eq 0 ]
  [ "$output" = $'--values-file\ncustom.yaml\n--heading-level\n3' ]
}

@test "boolean true emits a bare flag" {
  run env -i INPUT_STRICT=true "$SCRIPT"
  [ "$status" -eq 0 ]
  [ "$output" = "--strict" ]
}

@test "boolean false emits nothing" {
  run env -i INPUT_STRICT=false "$SCRIPT"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "empty string input is skipped" {
  run env -i INPUT_VALUES_FILE= INPUT_TYPE_COLUMN=true "$SCRIPT"
  [ "$status" -eq 0 ]
  [ "$output" = "--type-column" ]
}
