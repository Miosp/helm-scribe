package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/miosp/helm-scribe/config"
	"github.com/miosp/helm-scribe/parser"
	"github.com/miosp/helm-scribe/readme"
	"github.com/miosp/helm-scribe/schema"
	"github.com/spf13/cobra"
)

type WarningsError struct {
	Count int
}

func (e *WarningsError) Error() string {
	return fmt.Sprintf("%d warning(s) found", e.Count)
}

var rootCmd = &cobra.Command{
	Use:   "helm-scribe [chart-directory]",
	Short: "Generate README parameters table and values.schema.json from Helm values.yaml",
	Long:  "helm-scribe reads a Helm chart's values.yaml and generates\na parameters table in README.md and a values.schema.json for validation.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  execute,
}

func init() {
	f := rootCmd.Flags()
	f.StringP("values-file", "v", "", "Path to values file")
	f.StringP("readme-file", "r", "", "Path to README file")
	f.StringP("config", "c", ".helm-scribe.yaml", "Path to config file")
	f.IntP("truncate-length", "t", 0, "Max default value length before truncation")
	f.BoolP("dry-run", "n", false, "Print output to stdout instead of writing files")
	f.Bool("no-pretty", false, "Disable table column alignment")
	f.Int("heading-level", 0, "Heading level for section headers (1-6, default: 2)")
	f.StringP("schema-file", "s", "", "Path to schema output file")
	f.Bool("schema-only", false, "Only generate schema, skip README")
	f.Bool("readme-only", false, "Only generate README, skip schema")
	f.Bool("strict", false, "Treat warnings as errors (exit code 2)")
	f.Bool("type-column", false, "Show type column in README table")
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		var we *WarningsError
		if errors.As(err, &we) {
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) error {
	chartDir := "."
	if len(args) > 0 {
		chartDir = args[0]
	}

	f := cmd.Flags()
	configFile, _ := f.GetString("config")
	valuesFile, _ := f.GetString("values-file")
	readmeFile, _ := f.GetString("readme-file")
	truncateLength, _ := f.GetInt("truncate-length")
	dryRun, _ := f.GetBool("dry-run")
	noPretty, _ := f.GetBool("no-pretty")
	headingLevel, _ := f.GetInt("heading-level")

	cfg, err := config.LoadConfig(filepath.Join(chartDir, configFile))
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if valuesFile != "" {
		cfg.ValuesFile = valuesFile
	}
	if readmeFile != "" {
		cfg.ReadmeFile = readmeFile
	}
	if f.Changed("truncate-length") {
		cfg.TruncateLength = truncateLength
	}
	if f.Changed("heading-level") {
		cfg.HeadingLevel = headingLevel
	}
	schemaFile, _ := f.GetString("schema-file")
	schemaOnly, _ := f.GetBool("schema-only")
	readmeOnly, _ := f.GetBool("readme-only")

	if schemaFile != "" {
		cfg.SchemaFile = schemaFile
	}
	if f.Changed("dry-run") {
		cfg.DryRun = dryRun
	}
	if f.Changed("no-pretty") {
		cfg.NoPrettyPrint = noPretty
	}
	if f.Changed("type-column") {
		typeColumn, _ := f.GetBool("type-column")
		cfg.TypeColumn = typeColumn
	}
	if schemaOnly && readmeOnly {
		return fmt.Errorf("--schema-only and --readme-only are mutually exclusive")
	}
	cfg.SchemaOnly = schemaOnly
	cfg.ReadmeOnly = readmeOnly
	if f.Changed("strict") {
		strict, _ := f.GetBool("strict")
		cfg.Strict = strict
	}

	valuesPath := filepath.Join(chartDir, cfg.ValuesFile)
	readmePath := filepath.Join(chartDir, cfg.ReadmeFile)
	schemaPath := filepath.Join(filepath.Dir(valuesPath), cfg.SchemaFile)

	return run(cfg, valuesPath, readmePath, schemaPath)
}

func run(cfg config.Config, valuesPath, readmePath, schemaPath string) error {
	data, err := os.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("reading values file: %w", err)
	}

	nodes, warnings, err := parser.Parse(data)
	if err != nil {
		return fmt.Errorf("parsing values: %w", err)
	}

	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}
	if len(warnings) > 0 {
		fmt.Fprintf(os.Stderr, "%d warning(s) found\n", len(warnings))
	}

	if !cfg.ReadmeOnly {
		schemaBytes, err := schema.Generate(nodes)
		if err != nil {
			return fmt.Errorf("generating schema: %w", err)
		}
		if cfg.DryRun {
			fmt.Println(string(schemaBytes))
		} else {
			if err := os.WriteFile(schemaPath, schemaBytes, 0644); err != nil {
				return fmt.Errorf("writing schema: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Updated %s\n", schemaPath)
		}
	}

	if !cfg.SchemaOnly {
		opts := readme.Options{TruncateLength: cfg.TruncateLength, HeadingLevel: cfg.HeadingLevel, NoPrettyPrint: cfg.NoPrettyPrint, TypeColumn: cfg.TypeColumn}
		table := readme.Generate(nodes, opts)

		if cfg.DryRun {
			fmt.Print(table)
		} else {
			readmeData, err := os.ReadFile(readmePath)
			if err != nil {
				return fmt.Errorf("reading README: %w", err)
			}

			result, err := readme.InsertIntoReadme(string(readmeData), table)
			if err != nil {
				return err
			}

			if err := os.WriteFile(readmePath, []byte(result), 0644); err != nil {
				return fmt.Errorf("writing README: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Updated %s\n", readmePath)
		}
	}

	if cfg.Strict && len(warnings) > 0 {
		return &WarningsError{Count: len(warnings)}
	}

	return nil
}
