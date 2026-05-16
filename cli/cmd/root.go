package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   "headliner",
		Short: "Generate click-optimized YouTube titles from your watch history",
		Long: `Headliner is a CLI tool that:

  1. Fetches the titles of your liked YouTube videos
  2. Analyses structural patterns, power words, and formatting signals
  3. Uses an LLM to generate click-optimized titles for your content

Commands:
  fetch     Pull and cache YouTube titles from your liked videos
  analyze   Run pattern analysis on cached titles and print a report
  generate  Generate title candidates for an article, transcript, or topic

Run "headliner <command> --help" for details on each command.`,
	}
)

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default: .headliner.yaml in current dir or $HOME)")
	rootCmd.AddCommand(fetchCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(generateCmd)
}
