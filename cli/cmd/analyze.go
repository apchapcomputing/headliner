package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/headliner/cli/internal/analysis"
	"github.com/headliner/cli/internal/config"
	"github.com/headliner/cli/internal/youtube"
)

var (
	analyzeJSON bool
	analyzeOut  string
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyse cached titles and print a pattern report",
	Long: `Reads cached titles from ~/.headliner/titles.json and runs structural
pattern analysis. The report covers:

  • Title length and word-count distributions
  • Structural template frequencies (How To, Number List, Question, etc.)
  • Punctuation and formatting signals (colons, brackets, numbers, caps)
  • Power-word frequency
  • Most common lead phrases
  • Top channels in your collection

Run "headliner fetch" first to populate the title cache.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		cache, err := youtube.LoadCache(cfg.CacheDir)
		if err != nil {
			return fmt.Errorf(
				"no title cache found — run 'headliner fetch' first: %w", err)
		}

		fmt.Printf("🔬  Analysing %d cached titles...\n", len(cache.Videos))
		report := analysis.Analyze(cache.Videos)

		if analyzeJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(report)
		}

		if analyzeOut != "" {
			f, err := os.Create(analyzeOut)
			if err != nil {
				return fmt.Errorf("creating output file: %w", err)
			}
			defer f.Close()
			enc := json.NewEncoder(f)
			enc.SetIndent("", "  ")
			if err := enc.Encode(report); err != nil {
				return err
			}
			fmt.Printf("📄  Report saved to %s\n", analyzeOut)
		}

		analysis.PrintSummary(report)
		return nil
	},
}

func init() {
	analyzeCmd.Flags().BoolVar(&analyzeJSON, "json", false, "Output full report as JSON to stdout")
	analyzeCmd.Flags().StringVarP(&analyzeOut, "out", "o", "", "Save JSON report to this file path")
}
