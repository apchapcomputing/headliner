// Package analysis examines a corpus of YouTube video titles and extracts
// structural patterns, power words, and formatting signals that correlate
// with high click-through rates.
package analysis

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/headliner/cli/internal/youtube"
)

// ─────────────────────────────────────────────────────────────────────────────
// Data types
// ─────────────────────────────────────────────────────────────────────────────

// PatternReport is the result of analysing a corpus of titles.
type PatternReport struct {
	TotalTitles int `json:"totalTitles"`

	// Length stats (in characters)
	LengthMin  int     `json:"lengthMin"`
	LengthMax  int     `json:"lengthMax"`
	LengthMean float64 `json:"lengthMean"`
	LengthP50  int     `json:"lengthP50"`
	LengthP90  int     `json:"lengthP90"`

	// Word count stats
	WordCountMean float64 `json:"wordCountMean"`
	WordCountP50  int     `json:"wordCountP50"`

	// Structural template frequencies (% of titles matching)
	Templates []TemplateMatch `json:"templates"`

	// Most common title-starting words/phrases
	LeadPhrases []FreqItem `json:"leadPhrases"`

	// Power words most seen across all titles
	PowerWords []FreqItem `json:"powerWords"`

	// Punctuation patterns
	ColonUsagePct    float64 `json:"colonUsagePct"`
	QuestionPct      float64 `json:"questionPct"`
	BracketPct       float64 `json:"bracketPct"`
	PipePct          float64 `json:"pipePct"`
	EllipsisPct      float64 `json:"ellipsisPct"`
	AllCapsWordsPct  float64 `json:"allCapsWordsPct"`
	NumberInTitlePct float64 `json:"numberInTitlePct"`

	// Channel style breakdown
	ChannelStyles []FreqItem `json:"channelStyles"`

	// Example titles illustrating each template
	TemplateExamples map[string][]string `json:"templateExamples"`
}

// TemplateMatch describes how often a named structural template appears.
type TemplateMatch struct {
	Name    string  `json:"name"`
	Pattern string  `json:"pattern"`
	Count   int     `json:"count"`
	Pct     float64 `json:"pct"`
}

// FreqItem is a word/phrase with its occurrence count and percentage.
type FreqItem struct {
	Text  string  `json:"text"`
	Count int     `json:"count"`
	Pct   float64 `json:"pct"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Template definitions
// ─────────────────────────────────────────────────────────────────────────────

type templateDef struct {
	Name    string
	Pattern string
	Re      *regexp.Regexp
}

var templates = []templateDef{
	{
		"How To",
		`^how (to|i|we|you)\b`,
		regexp.MustCompile(`(?i)^how (to|i|we|you)\b`),
	},
	{
		"Number List",
		`^\d+\s+(things|ways|tips|reasons|mistakes|steps|secrets|facts|lessons|ideas|hacks|tricks|rules)`,
		regexp.MustCompile(`(?i)^\d+\s+(things|ways|tips|reasons|mistakes|steps|secrets|facts|lessons|ideas|hacks|tricks|rules)\b`),
	},
	{
		"Why X",
		`^why\b`,
		regexp.MustCompile(`(?i)^why\b`),
	},
	{
		"What X",
		`^what\b`,
		regexp.MustCompile(`(?i)^what\b`),
	},
	{
		"I/We Did X",
		`^(i|we) (did|tried|built|made|learned|spent|quit|left|started|stopped|found)\b`,
		regexp.MustCompile(`(?i)^(i|we) (did|tried|built|made|learned|spent|quit|left|started|stopped|found)\b`),
	},
	{
		"X Things You Need",
		`(need to know|need to|you must|you should|you need)`,
		regexp.MustCompile(`(?i)(need to know|need to|you must|you should|you need)`),
	},
	{
		"Title: Subtitle (Colon Split)",
		`.+:.+`,
		regexp.MustCompile(`.+:\s*.+`),
	},
	{
		"Bracket Qualifier [...]",
		`\[.+\]`,
		regexp.MustCompile(`\[.+\]`),
	},
	{
		"Bracket Qualifier (...)",
		`\(.+\)`,
		regexp.MustCompile(`\(.+\)`),
	},
	{
		"Question Title",
		`\?$`,
		regexp.MustCompile(`\?`),
	},
	{
		"Emotional Hook (Adjective-led)",
		`^(the (best|worst|most|biggest|ultimate|only|real|true|hidden|secret|honest)|stop|never|always|this changed|this is why)\b`,
		regexp.MustCompile(`(?i)^(the (best|worst|most|biggest|ultimate|only|real|true|hidden|secret|honest)|stop|never|always|this changed|this is why)\b`),
	},
	{
		"Year In Title",
		`\b(20\d{2})\b`,
		regexp.MustCompile(`\b(20\d{2})\b`),
	},
	{
		"Versus / Comparison",
		`\b(vs\.?|versus|compared to|or )\b`,
		regexp.MustCompile(`(?i)\b(vs\.?|versus|compared to)\b`),
	},
	{
		"Story / Personal Narrative",
		`^(my |the story of|how i|why i)\b`,
		regexp.MustCompile(`(?i)^(my |the story of|how i|why i)\b`),
	},
}

// ─────────────────────────────────────────────────────────────────────────────
// Power-word list
// ─────────────────────────────────────────────────────────────────────────────

var powerWordList = []string{
	"ultimate", "best", "worst", "secret", "hidden", "proven", "never",
	"always", "stop", "avoid", "instantly", "immediately", "finally",
	"actually", "honest", "truth", "real", "myth", "simple", "easy",
	"hard", "only", "every", "most", "must", "hack", "cheat",
	"complete", "definitive", "surprising", "shocking", "unbelievable",
	"incredible", "powerful", "essential", "critical", "important",
	"changed", "mistake", "fail", "dead", "broke", "free", "new",
	"boost", "grow", "master", "expert", "beginner",
}

// ─────────────────────────────────────────────────────────────────────────────
// Analyser
// ─────────────────────────────────────────────────────────────────────────────

// Analyze processes a slice of Videos and returns a PatternReport.
func Analyze(videos []youtube.Video) *PatternReport {
	titles := make([]string, 0, len(videos))
	for _, v := range videos {
		if v.Title != "" && v.Title != "Private video" && v.Title != "Deleted video" {
			titles = append(titles, v.Title)
		}
	}

	r := &PatternReport{
		TotalTitles:      len(titles),
		TemplateExamples: make(map[string][]string),
	}

	if len(titles) == 0 {
		return r
	}

	// Length & word count
	lengths := make([]int, len(titles))
	wordCounts := make([]int, len(titles))
	totalLen := 0
	totalWords := 0
	for i, t := range titles {
		l := len(t)
		w := len(strings.Fields(t))
		lengths[i] = l
		wordCounts[i] = w
		totalLen += l
		totalWords += w
	}
	sort.Ints(lengths)
	sort.Ints(wordCounts)
	r.LengthMin = lengths[0]
	r.LengthMax = lengths[len(lengths)-1]
	r.LengthMean = math.Round(float64(totalLen)/float64(len(titles))*10) / 10
	r.LengthP50 = lengths[len(lengths)/2]
	r.LengthP90 = lengths[int(float64(len(lengths))*0.9)]
	r.WordCountMean = math.Round(float64(totalWords)/float64(len(titles))*10) / 10
	r.WordCountP50 = wordCounts[len(wordCounts)/2]

	// Punctuation patterns
	colonCount, questionCount, bracketCount, pipeCount, ellipsisCount := 0, 0, 0, 0, 0
	allCapsCount, numberCount := 0, 0
	numberRe := regexp.MustCompile(`\b\d+\b`)
	allCapsWordRe := regexp.MustCompile(`\b[A-Z]{2,}\b`)
	for _, t := range titles {
		if strings.Contains(t, ":") {
			colonCount++
		}
		if strings.Contains(t, "?") {
			questionCount++
		}
		if strings.Contains(t, "[") || strings.Contains(t, "(") {
			bracketCount++
		}
		if strings.Contains(t, "|") {
			pipeCount++
		}
		if strings.Contains(t, "...") || strings.Contains(t, "…") {
			ellipsisCount++
		}
		if allCapsWordRe.MatchString(t) {
			allCapsCount++
		}
		if numberRe.MatchString(t) {
			numberCount++
		}
	}
	n := float64(len(titles))
	r.ColonUsagePct = pct(colonCount, n)
	r.QuestionPct = pct(questionCount, n)
	r.BracketPct = pct(bracketCount, n)
	r.PipePct = pct(pipeCount, n)
	r.EllipsisPct = pct(ellipsisCount, n)
	r.AllCapsWordsPct = pct(allCapsCount, n)
	r.NumberInTitlePct = pct(numberCount, n)

	// Template matching
	templateCounts := make(map[string]int, len(templates))
	for _, t := range titles {
		for _, tmpl := range templates {
			if tmpl.Re.MatchString(t) {
				templateCounts[tmpl.Name]++
				examples := r.TemplateExamples[tmpl.Name]
				if len(examples) < 3 {
					r.TemplateExamples[tmpl.Name] = append(examples, t)
				}
			}
		}
	}
	for _, tmpl := range templates {
		cnt := templateCounts[tmpl.Name]
		if cnt > 0 {
			r.Templates = append(r.Templates, TemplateMatch{
				Name:    tmpl.Name,
				Pattern: tmpl.Pattern,
				Count:   cnt,
				Pct:     pct(cnt, n),
			})
		}
	}
	sort.Slice(r.Templates, func(i, j int) bool {
		return r.Templates[i].Count > r.Templates[j].Count
	})

	// Leading phrases (first 2 words)
	leadFreq := make(map[string]int)
	for _, t := range titles {
		words := strings.Fields(t)
		if len(words) >= 2 {
			lead := strings.ToLower(words[0] + " " + words[1])
			// Strip trailing punctuation
			lead = strings.TrimRight(lead, ".,!?:")
			leadFreq[lead]++
		}
	}
	r.LeadPhrases = topFreqItems(leadFreq, 15, n)

	// Power words
	pwFreq := make(map[string]int)
	for _, t := range titles {
		lower := strings.ToLower(t)
		for _, pw := range powerWordList {
			if containsWord(lower, pw) {
				pwFreq[pw]++
			}
		}
	}
	r.PowerWords = topFreqItems(pwFreq, 20, n)

	// Channel style (leading word per channel)
	channelCount := make(map[string]int)
	for _, v := range videos {
		if v.ChannelTitle != "" {
			channelCount[v.ChannelTitle]++
		}
	}
	r.ChannelStyles = topFreqItems(channelCount, 10, n)

	return r
}

// PrintSummary prints a human-readable summary of the pattern report.
func PrintSummary(r *PatternReport) {
	fmt.Printf("\n╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("║          📊  TITLE PATTERN ANALYSIS REPORT           ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n\n")

	fmt.Printf("Total titles analysed: %d\n\n", r.TotalTitles)

	fmt.Printf("── Length (characters) ──────────────────────────────\n")
	fmt.Printf("  Min: %d   Max: %d   Mean: %.1f   P50: %d   P90: %d\n\n",
		r.LengthMin, r.LengthMax, r.LengthMean, r.LengthP50, r.LengthP90)

	fmt.Printf("── Word Count ───────────────────────────────────────\n")
	fmt.Printf("  Mean: %.1f   P50: %d\n\n", r.WordCountMean, r.WordCountP50)

	fmt.Printf("── Punctuation & Formatting ─────────────────────────\n")
	fmt.Printf("  Colon (:)       %5.1f%%\n", r.ColonUsagePct)
	fmt.Printf("  Question (?)    %5.1f%%\n", r.QuestionPct)
	fmt.Printf("  Brackets []()   %5.1f%%\n", r.BracketPct)
	fmt.Printf("  Numbers         %5.1f%%\n", r.NumberInTitlePct)
	fmt.Printf("  Pipe (|)        %5.1f%%\n", r.PipePct)
	fmt.Printf("  Ellipsis        %5.1f%%\n", r.EllipsisPct)
	fmt.Printf("  ALL CAPS word   %5.1f%%\n\n", r.AllCapsWordsPct)

	fmt.Printf("── Structural Templates (top 10) ────────────────────\n")
	for i, t := range r.Templates {
		if i >= 10 {
			break
		}
		fmt.Printf("  %-35s %4d  (%5.1f%%)\n", t.Name, t.Count, t.Pct)
	}
	fmt.Println()

	fmt.Printf("── Power Words (top 10) ─────────────────────────────\n")
	for i, pw := range r.PowerWords {
		if i >= 10 {
			break
		}
		fmt.Printf("  %-20s %4d  (%5.1f%%)\n", pw.Text, pw.Count, pw.Pct)
	}
	fmt.Println()

	fmt.Printf("── Most Common Lead Phrases (top 10) ────────────────\n")
	for i, lp := range r.LeadPhrases {
		if i >= 10 {
			break
		}
		fmt.Printf("  %-25s %4d  (%5.1f%%)\n", lp.Text, lp.Count, lp.Pct)
	}
	fmt.Println()

	fmt.Printf("── Top Channels in Your Collection ──────────────────\n")
	for i, ch := range r.ChannelStyles {
		if i >= 10 {
			break
		}
		fmt.Printf("  %-35s %4d videos\n", ch.Text, ch.Count)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func pct(count int, total float64) float64 {
	return math.Round(float64(count)/total*1000) / 10
}

func topFreqItems(freq map[string]int, topN int, total float64) []FreqItem {
	items := make([]FreqItem, 0, len(freq))
	for text, cnt := range freq {
		items = append(items, FreqItem{
			Text:  text,
			Count: cnt,
			Pct:   pct(cnt, total),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Text < items[j].Text
	})
	if len(items) > topN {
		items = items[:topN]
	}
	return items
}

func containsWord(s, word string) bool {
	for i := 0; i <= len(s)-len(word); i++ {
		if s[i:i+len(word)] == word {
			before := i == 0 || !unicode.IsLetter(rune(s[i-1]))
			after := i+len(word) == len(s) || !unicode.IsLetter(rune(s[i+len(word)]))
			if before && after {
				return true
			}
		}
	}
	return false
}
