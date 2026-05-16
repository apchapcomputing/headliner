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
based on patterns extracted from a creator's own liked and watched video library.
Your titles must feel natural, authentic, and match the tone of content the user actually engages with.

`)

	fmt.Fprintf(&b, "## Title Pattern Intelligence (from %d real titles this user watches)\n\n", r.TotalTitles)

	// ── Length & word count ───────────────────────────────────────────────────
	fmt.Fprintf(&b, "### Length Guidelines\n")
	fmt.Fprintf(&b, "- Target %d–%d characters (mean %.0f, median %d, P90 %d)\n",
		r.LengthMin, r.LengthMax, r.LengthMean, r.LengthP50, r.LengthP90)
	fmt.Fprintf(&b, "- Target ~%d words (median %d)\n\n", r.WordCountP50, r.WordCountP50)

	// ── Structural templates ──────────────────────────────────────────────────
	fmt.Fprintf(&b, "### Structural Templates (ranked by frequency in their watch history)\n")
	fmt.Fprintf(&b, "Use these templates as inspiration — they are proven to resonate with this audience.\n")
	for i, t := range r.Templates {
		if i >= 12 {
			break
		}
		fmt.Fprintf(&b, "\n**%s** (%.1f%% of titles)\n", t.Name, t.Pct)
		if examples, ok := r.TemplateExamples[t.Name]; ok && len(examples) > 0 {
			for _, ex := range examples {
				fmt.Fprintf(&b, "  → \"%s\"\n", ex)
			}
		}
	}
	b.WriteString("\n")

	// ── Emotional triggers ────────────────────────────────────────────────────
	if len(r.EmotionalTriggers) > 0 {
		fmt.Fprintf(&b, "### Emotional Hooks (ranked by prevalence)\n")
		fmt.Fprintf(&b, "These emotional angles appear most in titles the user clicks:\n")
		for i, e := range r.EmotionalTriggers {
			if i >= 5 {
				break
			}
			fmt.Fprintf(&b, "- **%s** (%.1f%%)\n", e.Text, e.Pct)
		}
		b.WriteString("\n")
	}

	// ── Curiosity gap patterns ────────────────────────────────────────────────
	if len(r.CuriosityGaps) > 0 {
		fmt.Fprintf(&b, "### Curiosity Gap Devices\n")
		fmt.Fprintf(&b, "These patterns create information gaps that compel clicks:\n")
		for i, c := range r.CuriosityGaps {
			if i >= 6 || c.Count == 0 {
				break
			}
			fmt.Fprintf(&b, "- %s (%.1f%%)\n", c.Text, c.Pct)
		}
		b.WriteString("\n")
	}

	// ── Formatting signals ────────────────────────────────────────────────────
	fmt.Fprintf(&b, "### Formatting Signals\n")
	fmt.Fprintf(&b, "- Positive framing (\"How to\", \"Build\", \"Best\"): %.0f%% of titles\n", r.PositiveFramingPct)
	fmt.Fprintf(&b, "- Negative framing (\"Stop\", \"Don't\", \"Worst\"): %.0f%% of titles\n", r.NegativeFramingPct)
	fmt.Fprintf(&b, "- Titles with numbers: %.0f%%\n", r.NumberInTitlePct)
	fmt.Fprintf(&b, "- Titles with time reference (\"in 30 days\", \"5 hours\"): %.0f%%\n", r.HasTimePct)
	fmt.Fprintf(&b, "- Titles with money/$ reference: %.0f%%\n", r.HasMoneyPct)
	fmt.Fprintf(&b, "- Titles starting with a number (list format): %.0f%%\n", r.HasListNumPct)
	if r.ColonUsagePct > 5 {
		fmt.Fprintf(&b, "- Colon splits (\"Main Idea: Specific Detail\"): %.0f%%\n", r.ColonUsagePct)
	}
	if r.QuestionPct > 5 {
		fmt.Fprintf(&b, "- Question titles: %.0f%%\n", r.QuestionPct)
	}
	if r.BracketPct > 5 {
		fmt.Fprintf(&b, "- Bracket/paren qualifiers [like this]: %.0f%%\n", r.BracketPct)
	}
	if r.AllCapsWordsPct > 5 {
		fmt.Fprintf(&b, "- Titles containing an ALL CAPS word: %.0f%% — use sparingly for emphasis\n", r.AllCapsWordsPct)
	}
	b.WriteString("\n")

	// ── Power words ───────────────────────────────────────────────────────────
	if len(r.PowerWords) > 0 {
		words := make([]string, 0, 12)
		for i, pw := range r.PowerWords {
			if i >= 12 {
				break
			}
			words = append(words, pw.Text)
		}
		fmt.Fprintf(&b, "### High-Frequency Power Words\n")
		fmt.Fprintf(&b, "Use sparingly and only when natural: %s\n\n", strings.Join(words, ", "))
	}

	// ── Lead phrases & bigrams ────────────────────────────────────────────────
	if len(r.LeadPhrases) > 0 {
		fmt.Fprintf(&b, "### Common Opening Phrases (proven openers in this niche)\n")
		for i, lp := range r.LeadPhrases {
			if i >= 8 {
				break
			}
			fmt.Fprintf(&b, "- \"%s\" (%.1f%%)\n", lp.Text, lp.Pct)
		}
		b.WriteString("\n")
	}

	if len(r.Bigrams) > 0 {
		phrases := make([]string, 0, 8)
		for i, bg := range r.Bigrams {
			if i >= 8 {
				break
			}
			phrases = append(phrases, "\""+bg.Text+"\"")
		}
		fmt.Fprintf(&b, "### Frequently Occurring Phrases\n")
		fmt.Fprintf(&b, "These 2-word phrases resonate with this audience: %s\n\n", strings.Join(phrases, ", "))
	}

	// ── Topic clusters ────────────────────────────────────────────────────────
	if len(r.TopicClusters) > 0 {
		fmt.Fprintf(&b, "### Top Topic Clusters\n")
		fmt.Fprintf(&b, "Stay within these domains for maximum relevance:\n")
		for i, tc := range r.TopicClusters {
			if i >= 5 || tc.Count == 0 {
				break
			}
			fmt.Fprintf(&b, "- %s (%.1f%%)\n", tc.Text, tc.Pct)
		}
		b.WriteString("\n")
	}

	// ── Tone override ─────────────────────────────────────────────────────────
	if opts.Tone != "" {
		fmt.Fprintf(&b, "### Tone\nThe titles should feel: **%s**\n\n", opts.Tone)
	}

	b.WriteString(`### Rules
- Output ONLY the titles, one per line, numbered 1. 2. 3. etc.
- Do NOT add explanations, intros, or commentary.
- Do NOT use quotation marks around titles.
- Keep each title under 100 characters.
- Vary the templates across your suggestions — don't repeat the same structure.
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
