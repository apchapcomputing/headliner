package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/headliner/cli/internal/analysis"
	"github.com/headliner/cli/internal/config"
	"github.com/headliner/cli/internal/generate"
	"github.com/headliner/cli/internal/youtube"
)

var (
	genInput     string
	genCount     int
	genModel     string
	genTone      string
	genExtras    string
	genSkipFetch bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate YouTube title candidates for your content",
	Long: `Generates click-optimized YouTube title candidates by combining:

  1. Pattern intelligence from your liked/watched video title cache
  2. An LLM (default: gpt-4o) to craft titles that match your proven taste

Input can be provided as:
  • --input <file>     path to an article, transcript, or topic summary file
  • stdin              pipe content directly (e.g. cat article.txt | headliner generate)
  • interactive prompt if neither is supplied

Examples:
  headliner generate --input article.txt
  headliner generate --input transcript.txt --count 10 --tone educational
  cat notes.txt | headliner generate --count 7
  headliner generate  (then paste/type content and press Ctrl-D)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if err := requireConfig(cfg.OpenAIAPIKey, "OPENAI_API_KEY"); err != nil {
			return err
		}

		// Resolve model (flag > config > default)
		model := genModel
		if model == "" {
			model = cfg.OpenAIModel
		}
		if model == "" {
			model = "gpt-4o"
		}

		// ── Load or build pattern report ─────────────────────────────────────
		var report *analysis.PatternReport
		if cache, err := youtube.LoadCache(cfg.CacheDir); err == nil {
			fmt.Printf("📊  Loaded %d cached titles for pattern context.\n", len(cache.Videos))
			report = analysis.Analyze(cache.Videos)
		} else if !genSkipFetch {
			fmt.Println("⚠️  No title cache found. Run 'headliner fetch' first for best results.")
			fmt.Println("   Continuing with generic pattern context (use --skip-fetch to silence).")
			report = &analysis.PatternReport{} // empty report — still works
		}

		// ── Read input content ────────────────────────────────────────────────
		var inputText string
		switch {
		case genInput != "":
			b, err := os.ReadFile(genInput)
			if err != nil {
				return fmt.Errorf("reading input file: %w", err)
			}
			inputText = string(b)
		default:
			// Check if stdin is a pipe/redirect
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("reading stdin: %w", err)
				}
				inputText = string(b)
			} else {
				fmt.Println("\n📝  Paste your article, transcript, or topic summary below.")
				fmt.Println("   Press Ctrl-D (Linux/macOS) or Ctrl-Z then Enter (Windows) when done.\n")
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("reading input: %w", err)
				}
				inputText = string(b)
			}
		}

		inputText = strings.TrimSpace(inputText)
		if inputText == "" {
			return fmt.Errorf("no input content provided")
		}

		// ── Generate ─────────────────────────────────────────────────────────
		client := generate.New(cfg.OpenAIAPIKey)
		opts := generate.Options{
			Model:  model,
			Count:  genCount,
			Tone:   genTone,
			Extras: genExtras,
		}

		fmt.Printf("\n🤖  Generating %d title candidates with %s...\n\n", genCount, model)
		titles, err := client.GenerateTitles(context.Background(), inputText, report, opts)
		if err != nil {
			return fmt.Errorf("generating titles: %w", err)
		}

		fmt.Printf("╔══════════════════════════════════════════════════════╗\n")
		fmt.Printf("║               🎬  TITLE CANDIDATES                   ║\n")
		fmt.Printf("╚══════════════════════════════════════════════════════╝\n\n")
		for i, t := range titles {
			fmt.Printf("  %2d.  %s\n", i+1, t)
		}
		fmt.Println()

		return nil
	},
}

func init() {
	generateCmd.Flags().StringVarP(&genInput, "input", "i", "",
		"Path to article, transcript, or topic summary file (default: stdin)")
	generateCmd.Flags().IntVarP(&genCount, "count", "n", 5,
		"Number of title candidates to generate")
	generateCmd.Flags().StringVar(&genModel, "model", "",
		"OpenAI model to use (default: gpt-4o, or openai_model in config)")
	generateCmd.Flags().StringVar(&genTone, "tone", "",
		"Optional tone hint, e.g. 'educational', 'entertaining', 'technical'")
	generateCmd.Flags().StringVar(&genExtras, "extras", "",
		"Additional instructions to include in the prompt")
	generateCmd.Flags().BoolVar(&genSkipFetch, "skip-fetch", false,
		"Skip fetch reminder when no cache exists")
}
