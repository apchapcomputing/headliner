package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/headliner/cli/internal/auth"
	"github.com/headliner/cli/internal/config"
	"github.com/headliner/cli/internal/youtube"
)

var (
	fetchForce bool
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch and cache your liked YouTube video titles",
	Long: `Authenticates with YouTube via OAuth2 and downloads the titles of all
your liked videos. Titles are cached to ~/.headliner/titles.json so that
subsequent analyze and generate commands run instantly without needing
to call the YouTube API again.

Use --force to re-fetch even if a cache already exists.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if err := requireConfig(cfg.GoogleClientID, "GOOGLE_CLIENT_ID"); err != nil {
			return err
		}
		if err := requireConfig(cfg.GoogleClientSecret, "GOOGLE_CLIENT_SECRET"); err != nil {
			return err
		}

		// Check for existing cache
		if !fetchForce {
			if cache, err := youtube.LoadCache(cfg.CacheDir); err == nil {
				fmt.Printf("✅  Cache exists with %d titles (fetched %s).\n",
					len(cache.Videos), cache.FetchedAt.Format("2006-01-02 15:04"))
				fmt.Println("   Use --force to re-fetch.")
				return nil
			}
		}

		ctx := context.Background()

		// OAuth2
		tokenMgr := auth.New(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.CacheDir)
		httpClient, err := tokenMgr.GetClient(ctx)
		if err != nil {
			return fmt.Errorf("authenticating: %w", err)
		}

		yt := youtube.New(httpClient, cfg.CacheDir)

		// Fetch liked videos
		fmt.Println("🔍  Fetching liked videos...")
		liked, err := yt.FetchLiked(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Could not fetch liked videos: %v\n", err)
			liked = nil
		}

		allVideos := liked
		fmt.Printf("\n✅  Fetched %d total titles.\n", len(allVideos))

		if err := yt.SaveCache(allVideos); err != nil {
			return fmt.Errorf("saving cache: %w", err)
		}
		fmt.Printf("💾  Saved to %s/titles.json\n", cfg.CacheDir)
		return nil
	},
}

func init() {
	fetchCmd.Flags().BoolVar(&fetchForce, "force", false, "Re-fetch even if cache exists")
}

func requireConfig(val, name string) error {
	if val == "" {
		return fmt.Errorf(
			"missing required config: %s\n"+
				"  Set it via env var, or in .headliner.yaml:\n"+
				"    %s: <value>",
			name, name,
		)
	}
	return nil
}
