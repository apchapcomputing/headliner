// Package youtube provides a client for fetching video titles from the
// YouTube Data API v3, with full pagination support.
package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const apiBase = "https://www.googleapis.com/youtube/v3"

// Video represents a minimal YouTube video record.
type Video struct {
	ID           string    `json:"id"`
	VideoID      string    `json:"videoId"`
	Title        string    `json:"title"`
	ChannelTitle string    `json:"channelTitle"`
	PublishedAt  time.Time `json:"publishedAt"`
}

// Cache is the on-disk cache format.
type Cache struct {
	FetchedAt time.Time `json:"fetchedAt"`
	Videos    []Video   `json:"videos"`
}

// Client wraps an authenticated HTTP client for YouTube API calls.
type Client struct {
	http     *http.Client
	cacheDir string
}

// New creates a YouTube client.
func New(httpClient *http.Client, cacheDir string) *Client {
	return &Client{http: httpClient, cacheDir: cacheDir}
}

// FetchLiked retrieves all liked videos for the authenticated user.
func (c *Client) FetchLiked(ctx context.Context) ([]Video, error) {
	fmt.Println("📥  Fetching liked videos playlist ID...")
	playlistID, err := c.likedPlaylistID(ctx)
	if err != nil {
		return nil, err
	}
	fmt.Printf("   Playlist ID: %s\n", playlistID)
	return c.fetchPlaylist(ctx, playlistID, "liked videos")
}

// likedPlaylistID retrieves the "liked videos" playlist ID from the channels endpoint.
func (c *Client) likedPlaylistID(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/channels?part=contentDetails&mine=true", apiBase)
	var resp struct {
		Items []struct {
			ContentDetails struct {
				RelatedPlaylists struct {
					Likes string `json:"likes"`
				} `json:"relatedPlaylists"`
			} `json:"contentDetails"`
		} `json:"items"`
	}
	if err := c.getJSON(ctx, url, &resp); err != nil {
		return "", fmt.Errorf("fetching channel info: %w", err)
	}
	if len(resp.Items) == 0 {
		return "", fmt.Errorf("no channel found for authenticated user")
	}
	id := resp.Items[0].ContentDetails.RelatedPlaylists.Likes
	if id == "" {
		return "", fmt.Errorf("liked videos playlist ID is empty")
	}
	return id, nil
}

// fetchPlaylist paginates through all items in a playlist.
func (c *Client) fetchPlaylist(ctx context.Context, playlistID, label string) ([]Video, error) {
	var videos []Video
	pageToken := ""
	page := 1

	for {
		url := fmt.Sprintf(
			"%s/playlistItems?part=snippet&playlistId=%s&maxResults=50",
			apiBase, playlistID,
		)
		if pageToken != "" {
			url += "&pageToken=" + pageToken
		}

		var resp struct {
			NextPageToken string `json:"nextPageToken"`
			PageInfo      struct {
				TotalResults int `json:"totalResults"`
			} `json:"pageInfo"`
			Items []struct {
				ID      string `json:"id"`
				Snippet struct {
					Title        string `json:"title"`
					ChannelTitle string `json:"channelTitle"`
					PublishedAt  string `json:"publishedAt"`
					ResourceID   struct {
						VideoID string `json:"videoId"`
					} `json:"resourceId"`
				} `json:"snippet"`
			} `json:"items"`
		}

		if err := c.getJSON(ctx, url, &resp); err != nil {
			return nil, fmt.Errorf("fetching playlist page %d: %w", page, err)
		}

		for _, item := range resp.Items {
			pub, _ := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
			videos = append(videos, Video{
				ID:           item.ID,
				VideoID:      item.Snippet.ResourceID.VideoID,
				Title:        item.Snippet.Title,
				ChannelTitle: item.Snippet.ChannelTitle,
				PublishedAt:  pub,
			})
		}

		fmt.Printf("   Page %d — fetched %d/%d %s\n", page, len(videos), resp.PageInfo.TotalResults, label)

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
		page++
	}

	return videos, nil
}

// getJSON performs a GET request and JSON-decodes the response body.
func (c *Client) getJSON(ctx context.Context, url string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return fmt.Errorf("YouTube API %d: %s", apiErr.Error.Code, apiErr.Error.Message)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// SaveCache writes a Cache to <cacheDir>/titles.json.
func (c *Client) SaveCache(videos []Video) error {
	if err := os.MkdirAll(c.cacheDir, 0700); err != nil {
		return err
	}
	cache := Cache{FetchedAt: time.Now(), Videos: videos}
	f, err := os.Create(filepath.Join(c.cacheDir, "titles.json"))
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cache)
}

// LoadCache reads the cached titles from disk.
func LoadCache(cacheDir string) (*Cache, error) {
	f, err := os.Open(filepath.Join(cacheDir, "titles.json"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cache Cache
	if err := json.NewDecoder(f).Decode(&cache); err != nil {
		return nil, err
	}
	return &cache, nil
}
