package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

const (
	DefaultTruncateLength = 80
	DefaultValuesFile     = "values.yaml"
	DefaultReadmeFile     = "README.md"
	DefaultSchemaFile     = "values.schema.json"
)

type Config struct {
	TruncateLength int    `yaml:"truncateLength"`
	ValuesFile     string `yaml:"valuesFile"`
	ReadmeFile     string `yaml:"readmeFile"`
	SchemaFile     string `yaml:"schemaFile"`
	TypeColumn     bool   `yaml:"typeColumn"`
	Strict         bool   `yaml:"strict"`
	NoPrettyPrint  bool   `yaml:"noPrettyPrint"`
	DryRun         bool   `yaml:"-"`
	SchemaOnly     bool   `yaml:"-"`
	ReadmeOnly     bool   `yaml:"-"`
}

func DefaultConfig() Config {
	return Config{
		TruncateLength: DefaultTruncateLength,
		ValuesFile:     DefaultValuesFile,
		ReadmeFile:     DefaultReadmeFile,
		SchemaFile:     DefaultSchemaFile,
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	if cfg.TruncateLength == 0 {
		cfg.TruncateLength = DefaultTruncateLength
	}
	if cfg.ValuesFile == "" {
		cfg.ValuesFile = DefaultValuesFile
	}
	if cfg.ReadmeFile == "" {
		cfg.ReadmeFile = DefaultReadmeFile
	}
	if cfg.SchemaFile == "" {
		cfg.SchemaFile = DefaultSchemaFile
	}

	return cfg, nil
}
