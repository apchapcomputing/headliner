// Package generate uses an LLM to create YouTube titles informed by a
// PatternReport from the user's own watch history.
package generate

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"

	"github.com/headliner/cli/internal/analysis"
)

// Options controls title generation.
type Options struct {
	Model  string // OpenAI model name, e.g. "gpt-4o"
	Count  int    // Number of title candidates to generate
	Tone   string // Optional tone hint, e.g. "educational", "entertaining"
	Extras string // Any additional instructions from the user
}

// DefaultOptions returns sensible generation defaults.
func DefaultOptions() Options {
	return Options{
		Model: "gpt-4o",
		Count: 5,
	}
}

// Client wraps the OpenAI client.
type Client struct {
	ai *openai.Client
}

// New creates a generation Client using the provided OpenAI API key.
func New(apiKey string) *Client {
	return &Client{ai: openai.NewClient(apiKey)}
}

// GenerateTitles calls the LLM with the pattern context and user input.
func (c *Client) GenerateTitles(ctx context.Context, input string, report *analysis.PatternReport, opts Options) ([]string, error) {
	if opts.Count <= 0 {
		opts.Count = 5
	}
	systemPrompt := buildSystemPrompt(report, opts)
	userPrompt := buildUserPrompt(input, opts)

	resp, err := c.ai.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: opts.Model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.85,
	})
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no completions returned from OpenAI")
	}

	raw := strings.TrimSpace(resp.Choices[0].Message.Content)
	return parseTitles(raw), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Prompt builders
// ─────────────────────────────────────────────────────────────────────────────

func buildSystemPrompt(r *analysis.PatternReport, opts Options) string {
	var b strings.Builder

	b.WriteString(`You are an expert YouTube title copywriter. You generate click-optimized YouTube titles 
based on the proven patterns extracted from a creator's own liked and watched video library.
Your titles must feel natural, authentic, and match the tone of content the user actually engages with.

`)

	// Inject pattern data
	fmt.Fprintf(&b, "## Title Pattern Intelligence (from %d real titles this user watches)\n\n", r.TotalTitles)

	fmt.Fprintf(&b, "### Length Guidelines\n")
	fmt.Fprintf(&b, "- Typical length: %d–%d characters (mean %.0f, median %d)\n",
		r.LengthMin, r.LengthMax, r.LengthMean, r.LengthP50)
	fmt.Fprintf(&b, "- Typical word count: median %d words\n\n", r.WordCountP50)

	fmt.Fprintf(&b, "### Structural Templates (most to least common in their watch history)\n")
	for i, t := range r.Templates {
		if i >= 8 {
			break
		}
		fmt.Fprintf(&b, "- **%s** (%.1f%% of titles) — pattern: `%s`\n", t.Name, t.Pct, t.Pattern)
		if examples, ok := r.TemplateExamples[t.Name]; ok {
			for _, ex := range examples {
				fmt.Fprintf(&b, "  - e.g. \"%s\"\n", ex)
			}
		}
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "### Formatting Signals\n")
	if r.ColonUsagePct > 20 {
		fmt.Fprintf(&b, "- Colons are common (%.0f%%) — consider \"Main Idea: Specific Detail\" format\n", r.ColonUsagePct)
	}
	if r.NumberInTitlePct > 20 {
		fmt.Fprintf(&b, "- Numbers appear in %.0f%% of titles — numbered lists perform well\n", r.NumberInTitlePct)
	}
	if r.QuestionPct > 15 {
		fmt.Fprintf(&b, "- Question titles appear in %.0f%% — questions drive curiosity\n", r.QuestionPct)
	}
	if r.BracketPct > 15 {
		fmt.Fprintf(&b, "- Bracket qualifiers [like this] or (like this) appear in %.0f%%\n", r.BracketPct)
	}
	b.WriteString("\n")

	if len(r.PowerWords) > 0 {
		fmt.Fprintf(&b, "### High-Frequency Power Words\n")
		words := make([]string, 0, 10)
		for i, pw := range r.PowerWords {
			if i >= 10 {
				break
			}
			words = append(words, pw.Text)
		}
		fmt.Fprintf(&b, "Use these sparingly and only when natural: %s\n\n", strings.Join(words, ", "))
	}

	if len(r.LeadPhrases) > 0 {
		fmt.Fprintf(&b, "### Common Opening Phrases\n")
		for i, lp := range r.LeadPhrases {
			if i >= 6 {
				break
			}
			fmt.Fprintf(&b, "- \"%s\"\n", lp.Text)
		}
		b.WriteString("\n")
	}

	if opts.Tone != "" {
		fmt.Fprintf(&b, "### Tone\nThe titles should feel: **%s**\n\n", opts.Tone)
	}

	b.WriteString(`### Rules
- Output ONLY the titles, one per line, numbered 1. 2. 3. etc.
- Do NOT add explanations, intros, or commentary.
- Do NOT use quotation marks around titles.
- Keep each title under 100 characters.
- Every title must be immediately usable on YouTube.
`)

	if opts.Extras != "" {
		fmt.Fprintf(&b, "\n### Additional Instructions\n%s\n", opts.Extras)
	}

	return b.String()
}

func buildUserPrompt(input string, opts Options) string {
	return fmt.Sprintf(
		"Generate %d YouTube title options for the following content:\n\n---\n%s\n---",
		opts.Count, strings.TrimSpace(input),
	)
}

// parseTitles splits the LLM response into individual title strings.
func parseTitles(raw string) []string {
	lines := strings.Split(raw, "\n")
	numRe := strings.NewReplacer(
		"1. ", "", "2. ", "", "3. ", "", "4. ", "", "5. ", "",
		"6. ", "", "7. ", "", "8. ", "", "9. ", "", "10. ", "",
	)
	var titles []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip leading number+dot pattern robustly
		stripped := strings.TrimSpace(numRe.Replace(line))
		if stripped == "" {
			stripped = line
		}
		// Remove surrounding quotes if present
		stripped = strings.Trim(stripped, `"'`)
		if stripped != "" {
			titles = append(titles, stripped)
		}
	}
	return titles
}
