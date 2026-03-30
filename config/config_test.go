package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.TruncateLength != 80 {
		t.Errorf("truncateLength: got %d", cfg.TruncateLength)
	}
	if cfg.ValuesFile != "values.yaml" {
		t.Errorf("valuesFile: got %q", cfg.ValuesFile)
	}
	if cfg.ReadmeFile != "README.md" {
		t.Errorf("readmeFile: got %q", cfg.ReadmeFile)
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".helm-scribe.yaml")
	if err := os.WriteFile(cfgPath, []byte("truncateLength: 40\nvaluesFile: my-values.yaml\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TruncateLength != 40 {
		t.Errorf("truncateLength: got %d", cfg.TruncateLength)
	}
	if cfg.ValuesFile != "my-values.yaml" {
		t.Errorf("valuesFile: got %q", cfg.ValuesFile)
	}
	if cfg.ReadmeFile != "README.md" {
		t.Errorf("readmeFile: got %q", cfg.ReadmeFile)
	}
}

func TestLoadConfig_MissingFileUsesDefaults(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/.helm-scribe.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TruncateLength != 80 {
		t.Errorf("truncateLength: got %d", cfg.TruncateLength)
	}
}

func TestLoadConfig_TruncateLengthZeroDisablesTruncation(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".helm-scribe.yaml")
	if err := os.WriteFile(cfgPath, []byte("truncateLength: 0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TruncateLength != 0 {
		t.Errorf("truncateLength: got %d, want 0 (disabled)", cfg.TruncateLength)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".helm-scribe.yaml")
	if err := os.WriteFile(cfgPath, []byte(":\n  :\n    - [invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Error("expected error for invalid YAML config")
	}
}
