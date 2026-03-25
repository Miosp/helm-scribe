package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/miosp/helm-scribe/config"
	"github.com/miosp/helm-scribe/parser"
	"github.com/miosp/helm-scribe/readme"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "helm-scribe [chart-directory]",
	Short: "Generate README parameters table from Helm values.yaml",
	Long:  "helm-scribe reads a Helm chart's values.yaml and generates\na parameters table in README.md between helm-scribe markers.",
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
}

func main() {
	if err := rootCmd.Execute(); err != nil {
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
	if truncateLength > 0 {
		cfg.TruncateLength = truncateLength
	}
	cfg.DryRun = dryRun

	valuesPath := filepath.Join(chartDir, cfg.ValuesFile)
	readmePath := filepath.Join(chartDir, cfg.ReadmeFile)

	return run(cfg, valuesPath, readmePath)
}

func run(cfg config.Config, valuesPath, readmePath string) error {
	data, err := os.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("reading values file: %w", err)
	}

	nodes, err := parser.Parse(data)
	if err != nil {
		return fmt.Errorf("parsing values: %w", err)
	}

	opts := readme.Options{TruncateLength: cfg.TruncateLength}
	table := readme.Generate(nodes, opts)

	if cfg.DryRun {
		fmt.Print(table)
		return nil
	}

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
	return nil
}
