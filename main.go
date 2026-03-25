package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/miosp/helm-scribe/config"
	"github.com/miosp/helm-scribe/parser"
	"github.com/miosp/helm-scribe/readme"
)

func main() {
	var (
		valuesFile     string
		readmeFile     string
		configFile     string
		truncateLength int
		dryRun         bool
	)

	flag.StringVar(&valuesFile, "values-file", "", "Path to values file")
	flag.StringVar(&readmeFile, "readme-file", "", "Path to README file")
	flag.StringVar(&configFile, "config", ".helm-scribe.yaml", "Path to config file")
	flag.IntVar(&truncateLength, "truncate-length", 0, "Max default value length before truncation")
	flag.BoolVar(&dryRun, "dry-run", false, "Print output to stdout instead of writing files")
	flag.Parse()

	chartDir := "."
	if flag.NArg() > 0 {
		chartDir = flag.Arg(0)
	}

	cfg, err := config.LoadConfig(filepath.Join(chartDir, configFile))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	// CLI flags override config
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

	if err := run(cfg, valuesPath, readmePath); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
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
