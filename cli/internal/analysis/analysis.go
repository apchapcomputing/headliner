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

	// Structural template frequencies
	Templates        []TemplateMatch     `json:"templates"`
	TemplateExamples map[string][]string `json:"templateExamples"`

	// Most common title-starting words/phrases
	LeadPhrases []FreqItem `json:"leadPhrases"`
	FirstWords  []FreqItem `json:"firstWords"`

	// Most common meaningful 2-gram and 3-gram phrases
	Bigrams  []FreqItem `json:"bigrams"`
	Trigrams []FreqItem `json:"trigrams"`

	// Power words
	PowerWords []FreqItem `json:"powerWords"`

	// Emotional trigger category counts
	EmotionalTriggers []FreqItem `json:"emotionalTriggers"`

	// Curiosity gap patterns
	CuriosityGaps []FreqItem `json:"curiosityGaps"`

	// Negative vs positive framing
	NegativeFramingPct float64 `json:"negativeFramingPct"`
	PositiveFramingPct float64 `json:"positiveFramingPct"`

	// Specificity signals
	HasTimePct    float64 `json:"hasTimePct"`
	HasMoneyPct   float64 `json:"hasMoneyPct"`
	HasListNumPct float64 `json:"hasListNumPct"`

	// Punctuation patterns
	ColonUsagePct    float64 `json:"colonUsagePct"`
	QuestionPct      float64 `json:"questionPct"`
	BracketPct       float64 `json:"bracketPct"`
	PipePct          float64 `json:"pipePct"`
	EllipsisPct      float64 `json:"ellipsisPct"`
	AllCapsWordsPct  float64 `json:"allCapsWordsPct"`
	NumberInTitlePct float64 `json:"numberInTitlePct"`

	// Topic clusters
	TopicClusters []FreqItem `json:"topicClusters"`

	// Channel breakdown
	ChannelStyles []FreqItem `json:"channelStyles"`
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
	// ── How To / Tutorial ────────────────────────────────────────────────────
	{
		"How To",
		`^how (to|i|we|you)\b`,
		regexp.MustCompile(`(?i)^how (to|i|we|you)\b`),
	},

	// ── Number List ──────────────────────────────────────────────────────────
	{
		"Number List (N things/ways/tips...)",
		`^\d+\s+(things|ways|tips|reasons|mistakes|steps|secrets|facts|lessons|ideas|hacks|tricks|rules|signs|tools|apps|books|questions|habits|principles)`,
		regexp.MustCompile(`(?i)^\d+\s+(things|ways|tips|reasons|mistakes|steps|secrets|facts|lessons|ideas|hacks|tricks|rules|signs|tools|apps|books|questions|habits|principles)\b`),
	},

	// ── Sacrifice / Vicarious effort ─────────────────────────────────────────
	{
		"I Did X So You Don't Have To",
		`so you don.?t have to`,
		regexp.MustCompile(`(?i)so you don.?t have to`),
	},
	{
		"I Spent X [time/money] on Y",
		`(i |we )(spent|paid|wasted|used|tested|tried).{0,40}(hour|day|week|month|year|\$|dollar)`,
		regexp.MustCompile(`(?i)(i |we )(spent|paid|wasted|used|tested|tried).{0,40}(hour|day|week|month|year|\$|dollar)`),
	},
	{
		"I Tested/Tried/Reviewed X",
		`^(i |we )(tested|tried|reviewed|ranked|compared|rated|used|bought|read|watched)\b`,
		regexp.MustCompile(`(?i)^(i |we )(tested|tried|reviewed|ranked|compared|rated|used|bought|read|watched)\b`),
	},

	// ── Personal narrative / confession ──────────────────────────────────────
	{
		"I/We Did/Built/Made X",
		`^(i|we) (did|built|made|learned|quit|left|started|stopped|found|wrote|switched|deleted|gave|lost|won|ran|sold)\b`,
		regexp.MustCompile(`(?i)^(i|we) (did|built|made|learned|quit|left|started|stopped|found|wrote|switched|deleted|gave|lost|won|ran|sold)\b`),
	},
	{
		"Story / Personal Narrative (My...)",
		`^(my |the story of|how i|why i)\b`,
		regexp.MustCompile(`(?i)^(my |the story of|how i|why i)\b`),
	},
	{
		"What Happened When I...",
		`what happened (when|after|if|while)\b`,
		regexp.MustCompile(`(?i)what happened (when|after|if|while)\b`),
	},

	// ── Challenge / time-box experiment ──────────────────────────────────────
	{
		"N Day/Week/Month/Year Challenge",
		`\d+[\s-]*(day|week|month|year)s? (challenge|experiment|of|test|later|streak)`,
		regexp.MustCompile(`(?i)\d+[\s-]*(day|week|month|year)s? (challenge|experiment|of|test|later|streak)`),
	},
	{
		"Using/Doing X for N days/weeks",
		`(using|trying|doing|with) .{0,30}\d+[\s-]*(day|week|month|year)`,
		regexp.MustCompile(`(?i)(using|trying|doing|with) .{0,30}\d+[\s-]*(day|week|month|year)`),
	},

	// ── Ranking / comparison ─────────────────────────────────────────────────
	{
		"Ranking / Tier List",
		`\b(rank(ing|ed)?|tier list|every .{0,30} ranked)\b`,
		regexp.MustCompile(`(?i)\b(rank(ing|ed)?|tier list|every .{0,30} ranked)\b`),
	},
	{
		"Versus / Comparison",
		`\b(vs\.?|versus|compared to)\b`,
		regexp.MustCompile(`(?i)\b(vs\.?|versus|compared to)\b`),
	},
	{
		"Every X Explained / Reviewed",
		`^every\b`,
		regexp.MustCompile(`(?i)^every\b`),
	},

	// ── Why / What / Question ────────────────────────────────────────────────
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
		"Question Title (?)",
		`\?`,
		regexp.MustCompile(`\?`),
	},

	// ── Advice / warning ─────────────────────────────────────────────────────
	{
		"X Things You Need / Should Know",
		`(need to know|need to|you must|you should|you need)`,
		regexp.MustCompile(`(?i)(need to know|need to|you must|you should|you need)`),
	},
	{
		"Stop Doing X",
		`^stop\b`,
		regexp.MustCompile(`(?i)^stop\b`),
	},
	{
		"Don't Do X",
		`^don.?t\b`,
		regexp.MustCompile(`(?i)^don.?t\b`),
	},
	{
		"The Problem With X",
		`^the (problem|issue|truth|reality|dark side|downside|catch) (with|of|about|is)\b`,
		regexp.MustCompile(`(?i)^the (problem|issue|truth|reality|dark side|downside|catch) (with|of|about|is)\b`),
	},

	// ── Formatting / structural ──────────────────────────────────────────────
	{
		"Title: Subtitle (Colon Split)",
		`.+:\s*.+`,
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
		"Year In Title",
		`\b20\d{2}\b`,
		regexp.MustCompile(`\b20\d{2}\b`),
	},

	// ── Emotional hooks ──────────────────────────────────────────────────────
	{
		"Emotional Hook (The Best/Worst/Ultimate...)",
		`^the (best|worst|most|biggest|ultimate|only|real|true|hidden|secret|honest)\b`,
		regexp.MustCompile(`(?i)^the (best|worst|most|biggest|ultimate|only|real|true|hidden|secret|honest)\b`),
	},
	{
		"Honest / Unpopular Opinion",
		`\b(honest(ly)?|unpopular opinion|controversial|hot take|overrated|underrated|brutal(ly)?)\b`,
		regexp.MustCompile(`(?i)\b(honest(ly)?|unpopular opinion|controversial|hot take|overrated|underrated|brutal(ly)?)\b`),
	},
	{
		"Mistake / Regret / Failure",
		`\b(mistake|regret|wrong|failed|failure|i wish|shouldn.?t have|never should)\b`,
		regexp.MustCompile(`(?i)\b(mistake|regret|wrong|failed|failure|i wish|shouldn.?t have|never should)\b`),
	},

	// ── Revelation / curiosity gap ───────────────────────────────────────────
	{
		"The Real / Hidden / Untold Reason",
		`\b(the real|the actual|the hidden|the secret|the true|the untold)\b`,
		regexp.MustCompile(`(?i)\b(the real|the actual|the hidden|the secret|the true|the untold)\b`),
	},
	{
		"Nobody Tells You / Talks About",
		`nobody (tells|talks|shows|knows|warned)\b`,
		regexp.MustCompile(`(?i)nobody (tells|talks|shows|knows|warned)\b`),
	},
	{
		"They Don't Want You To Know",
		`they (don.?t|won.?t|never)\b`,
		regexp.MustCompile(`(?i)they (don.?t|won.?t|never)\b`),
	},
	{
		"This Is Why",
		`^this is why\b`,
		regexp.MustCompile(`(?i)^this is why\b`),
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
// Emotional trigger categories
// ─────────────────────────────────────────────────────────────────────────────

type emotionDef struct {
	Name string
	Re   *regexp.Regexp
}

var emotionDefs = []emotionDef{
	{
		"Fear / Warning",
		regexp.MustCompile(`(?i)\b(danger|warning|risk|mistake|wrong|avoid|stop|never|worst|fail|broke|ruined|dying|dead|scam|trap|lied|lies|toxic|threat|crisis|problem|issue|terrible|horrible|disaster)\b`),
	},
	{
		"Aspiration / Achievement",
		regexp.MustCompile(`(?i)\b(success|achieve|goal|dream|rich|wealth|profit|win|growth|improve|better|best|master|level up|build|create|launch|earn|income|freedom|passive)\b`),
	},
	{
		"Curiosity / Mystery",
		regexp.MustCompile(`(?i)\b(secret|hidden|nobody|unknown|discover|reveal|truth|real reason|actually|surprising|shocking|untold|mystery|exposed|behind|inside)\b`),
	},
	{
		"Controversy / Challenge",
		regexp.MustCompile(`(?i)\b(unpopular|controversial|disagree|wrong about|overrated|underrated|myth|lie|lied|vs\.|versus|debate|honest|brutal)\b`),
	},
	{
		"Urgency / FOMO",
		regexp.MustCompile(`(?i)\b(before|now|today|immediately|instantly|fast|quick|urgent|limited|running out|last chance|don't miss|while you can|deadline)\b`),
	},
	{
		"Relatability / Struggle",
		regexp.MustCompile(`(?i)\b(struggle|hard|difficult|failed|gave up|burnout|exhausted|overwhelmed|stuck|confused|lost|scared|anxious|lonely|broke|regret)\b`),
	},
	{
		"Social Proof / Authority",
		regexp.MustCompile(`(?i)\b(expert|professional|years|studied|research|science|data|proven|tested|according|evidence|study|fact|statistics)\b`),
	},
	{
		"Transformation / Before & After",
		regexp.MustCompile(`(?i)\b(changed|transformed|went from|before|after|used to|now|journey|progress|results|difference|turned|became|switched)\b`),
	},
}

// ─────────────────────────────────────────────────────────────────────────────
// Curiosity gap patterns
// ─────────────────────────────────────────────────────────────────────────────

type curiosityDef struct {
	Name string
	Re   *regexp.Regexp
}

var curiosityDefs = []curiosityDef{
	{`"The Real Reason"`, regexp.MustCompile(`(?i)\bthe real reason\b`)},
	{`"Nobody Talks/Tells About"`, regexp.MustCompile(`(?i)\bnobody (talks|tells|shows|knows)\b`)},
	{`"You Won't Believe"`, regexp.MustCompile(`(?i)\byou won.?t believe\b`)},
	{`"What They Don't Tell You"`, regexp.MustCompile(`(?i)\bthey don.?t (tell|want|show)\b`)},
	{`"This Is Why"`, regexp.MustCompile(`(?i)^this is why\b`)},
	{`"Here's What Happened"`, regexp.MustCompile(`(?i)\b(what happened|here.?s what)\b`)},
	{`"I Had No Idea / Didn't Know"`, regexp.MustCompile(`(?i)\b(had no idea|didn.?t know|didn.?t expect|never knew)\b`)},
	{`"The Truth About"`, regexp.MustCompile(`(?i)\bthe truth (about|behind|is)\b`)},
	{`Ellipsis tease "..."`, regexp.MustCompile(`(\.{3}|` + "\u2026" + `)`)},
	{`Cliffhanger ("until I", "what I found")`, regexp.MustCompile(`(?i)\b(until i|then this|what i (found|learned|discovered)|you.?ll (never|always))\b`)},
	{`"So You Don't Have To"`, regexp.MustCompile(`(?i)so you don.?t have to`)},
	{`"I Spent X So That You..."`, regexp.MustCompile(`(?i)(i |we )(spent|tested|tried|wasted).{0,40}so (you|we)\b`)},
}

// ─────────────────────────────────────────────────────────────────────────────
// Topic cluster keywords
// ─────────────────────────────────────────────────────────────────────────────

type topicDef struct {
	Name string
	Re   *regexp.Regexp
}

var topicDefs = []topicDef{
	{"Programming / Dev", regexp.MustCompile(`(?i)\b(code|coding|program|software|developer|engineer|python|javascript|typescript|rust|golang|web|api|database|git|linux|terminal|cli|backend|frontend|fullstack|devops|kubernetes|docker|cloud|aws)\b`)},
	{"AI / Machine Learning", regexp.MustCompile(`(?i)\b(ai|artificial intelligence|machine learning|deep learning|llm|gpt|chatgpt|neural|model|training|dataset|openai|claude|gemini|copilot|prompt)\b`)},
	{"Productivity / Self-Improvement", regexp.MustCompile(`(?i)\b(productivity|habit|routine|focus|discipline|motivation|procrastinat|time management|morning|system|workflow|efficiency|goal|mindset)\b`)},
	{"Finance / Business", regexp.MustCompile(`(?i)\b(money|invest|stock|market|crypto|bitcoin|finance|business|startup|entrepreneur|revenue|profit|income|salary|wealth|budget|saving|spending|side hustle)\b`)},
	{"Design / Creative", regexp.MustCompile(`(?i)\b(design|ui|ux|figma|css|animation|graphic|brand|logo|typography|color|font|illustration|creative|art|visual)\b`)},
	{"Career / Work", regexp.MustCompile(`(?i)\b(career|job|interview|resume|hire|freelance|remote|work|company|promotion|skill|portfolio|linkedin|salary)\b`)},
	{"Health / Fitness", regexp.MustCompile(`(?i)\b(health|fitness|workout|gym|diet|nutrition|weight|exercise|run|sleep|mental health|anxiety|depression|meditation|yoga)\b`)},
	{"Gaming", regexp.MustCompile(`(?i)\b(game|gaming|play|player|esport|stream|twitch|minecraft|fortnite|valorant|steam|console|xbox|playstation|nintendo)\b`)},
	{"Science / Tech", regexp.MustCompile(`(?i)\b(science|physics|math|research|experiment|technology|engineering|space|nasa|quantum|robot|future|innovation)\b`)},
	{"Education / Learning", regexp.MustCompile(`(?i)\b(learn|study|school|course|tutorial|lesson|teach|university|degree|student|knowledge|skill|education)\b`)},
	{"Lifestyle / Vlog", regexp.MustCompile(`(?i)\b(life|living|travel|food|cook|recipe|home|apartment|minimalist|vlog|day in|week in|year in|challenge)\b`)},
	{"Content Creation / YouTube", regexp.MustCompile(`(?i)\b(youtube|content|creator|video|channel|subscriber|views|algorithm|thumbnail|editing|camera|filming|grow|audience)\b`)},
}

// ─────────────────────────────────────────────────────────────────────────────
// Framing & specificity
// ─────────────────────────────────────────────────────────────────────────────

var (
	negativeRe = regexp.MustCompile(`(?i)^(stop|don.?t|never|avoid|worst|why you (shouldn.?t|can.?t|won.?t)|the problem|mistake|wrong|fail|broke|quit|bad|terrible|horrible|warning|danger)`)
	positiveRe = regexp.MustCompile(`(?i)^(how to|the best|why you should|start|do this|build|create|grow|improve|master|learn|achieve|win|success|best way)`)
	timeRe     = regexp.MustCompile(`(?i)\b(\d+\s*(day|week|month|year|hour|minute|second)s?|in \d+|within \d+)\b`)
	moneyRe    = regexp.MustCompile(`(?i)(\$\d+|\d+[kK]\b|\d+ (dollar|million|billion|grand)|making money|earning)`)
	listNumRe  = regexp.MustCompile(`^\d+\s+\S`)
)

// stopWords filtered from n-gram analysis.
var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
	"with": true, "is": true, "it": true, "as": true, "be": true, "by": true,
	"i": true, "my": true, "you": true, "your": true, "we": true, "our": true,
	"this": true, "that": true, "they": true, "them": true, "their": true,
	"was": true, "are": true, "has": true, "have": true, "had": true,
	"not": true, "no": true, "do": true, "did": true, "so": true,
	"from": true, "about": true, "up": true, "out": true, "into": true,
	"what": true, "when": true, "where": true, "which": true, "who": true,
	"if": true, "than": true, "just": true, "me": true, "been": true,
	"its": true, "his": true, "her": true, "him": true, "she": true, "he": true,
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

	n := float64(len(titles))

	// ── Length & word count ───────────────────────────────────────────────────
	lengths := make([]int, len(titles))
	wordCounts := make([]int, len(titles))
	totalLen, totalWords := 0, 0
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
	r.LengthMean = math.Round(float64(totalLen)/n*10) / 10
	r.LengthP50 = lengths[len(lengths)/2]
	r.LengthP90 = lengths[int(n*0.9)]
	r.WordCountMean = math.Round(float64(totalWords)/n*10) / 10
	r.WordCountP50 = wordCounts[len(wordCounts)/2]

	// ── Punctuation & formatting ──────────────────────────────────────────────
	colonC, questionC, bracketC, pipeC, ellipsisC, allCapsC, numberC := 0, 0, 0, 0, 0, 0, 0
	numberRe := regexp.MustCompile(`\b\d+\b`)
	allCapsWordRe := regexp.MustCompile(`\b[A-Z]{2,}\b`)
	for _, t := range titles {
		if strings.Contains(t, ":") {
			colonC++
		}
		if strings.Contains(t, "?") {
			questionC++
		}
		if strings.Contains(t, "[") || strings.Contains(t, "(") {
			bracketC++
		}
		if strings.Contains(t, "|") {
			pipeC++
		}
		if strings.Contains(t, "...") || strings.Contains(t, "\u2026") {
			ellipsisC++
		}
		if allCapsWordRe.MatchString(t) {
			allCapsC++
		}
		if numberRe.MatchString(t) {
			numberC++
		}
	}
	r.ColonUsagePct = pct(colonC, n)
	r.QuestionPct = pct(questionC, n)
	r.BracketPct = pct(bracketC, n)
	r.PipePct = pct(pipeC, n)
	r.EllipsisPct = pct(ellipsisC, n)
	r.AllCapsWordsPct = pct(allCapsC, n)
	r.NumberInTitlePct = pct(numberC, n)

	// ── Structural templates ──────────────────────────────────────────────────
	templateCounts := make(map[string]int, len(templates))
	for _, t := range titles {
		for _, tmpl := range templates {
			if tmpl.Re.MatchString(t) {
				templateCounts[tmpl.Name]++
				if len(r.TemplateExamples[tmpl.Name]) < 3 {
					r.TemplateExamples[tmpl.Name] = append(r.TemplateExamples[tmpl.Name], t)
				}
			}
		}
	}
	for _, tmpl := range templates {
		if cnt := templateCounts[tmpl.Name]; cnt > 0 {
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

	// ── Lead phrases (first 2 words) & first words ───────────────────────────
	leadFreq := make(map[string]int)
	firstWordFreq := make(map[string]int)
	for _, t := range titles {
		words := strings.Fields(t)
		if len(words) >= 1 {
			fw := strings.ToLower(strings.TrimRight(words[0], ".,!?:"))
			firstWordFreq[fw]++
		}
		if len(words) >= 2 {
			lead := strings.ToLower(words[0] + " " + words[1])
			lead = strings.TrimRight(lead, ".,!?:")
			leadFreq[lead]++
		}
	}
	r.LeadPhrases = topFreqItems(leadFreq, 20, n)
	r.FirstWords = topFreqItems(firstWordFreq, 20, n)

	// ── N-grams (meaningful 2/3-word phrases, stop-words filtered) ───────────
	bigramFreq := make(map[string]int)
	trigramFreq := make(map[string]int)
	for _, t := range titles {
		words := tokenize(t)
		for i := 0; i < len(words); i++ {
			if i+1 < len(words) {
				w1, w2 := words[i], words[i+1]
				if !stopWords[w1] || !stopWords[w2] {
					bigramFreq[w1+" "+w2]++
				}
			}
			if i+2 < len(words) {
				w1, w2, w3 := words[i], words[i+1], words[i+2]
				if !stopWords[w1] && !stopWords[w3] {
					trigramFreq[w1+" "+w2+" "+w3]++
				}
			}
		}
	}
	r.Bigrams = topFreqItems(bigramFreq, 20, n)
	r.Trigrams = topFreqItems(trigramFreq, 20, n)

	// ── Power words ───────────────────────────────────────────────────────────
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

	// ── Emotional triggers ────────────────────────────────────────────────────
	emotFreq := make(map[string]int)
	for _, t := range titles {
		for _, e := range emotionDefs {
			if e.Re.MatchString(t) {
				emotFreq[e.Name]++
			}
		}
	}
	r.EmotionalTriggers = topFreqItems(emotFreq, len(emotionDefs), n)

	// ── Curiosity gap patterns ────────────────────────────────────────────────
	cgFreq := make(map[string]int)
	for _, t := range titles {
		for _, c := range curiosityDefs {
			if c.Re.MatchString(t) {
				cgFreq[c.Name]++
			}
		}
	}
	r.CuriosityGaps = topFreqItems(cgFreq, len(curiosityDefs), n)

	// ── Framing ───────────────────────────────────────────────────────────────
	negC, posC := 0, 0
	for _, t := range titles {
		if negativeRe.MatchString(t) {
			negC++
		}
		if positiveRe.MatchString(t) {
			posC++
		}
	}
	r.NegativeFramingPct = pct(negC, n)
	r.PositiveFramingPct = pct(posC, n)

	// ── Specificity signals ───────────────────────────────────────────────────
	timeC, moneyC, listC := 0, 0, 0
	for _, t := range titles {
		if timeRe.MatchString(t) {
			timeC++
		}
		if moneyRe.MatchString(t) {
			moneyC++
		}
		if listNumRe.MatchString(t) {
			listC++
		}
	}
	r.HasTimePct = pct(timeC, n)
	r.HasMoneyPct = pct(moneyC, n)
	r.HasListNumPct = pct(listC, n)

	// ── Topic clusters ────────────────────────────────────────────────────────
	topicFreq := make(map[string]int)
	for _, t := range titles {
		for _, td := range topicDefs {
			if td.Re.MatchString(t) {
				topicFreq[td.Name]++
			}
		}
	}
	r.TopicClusters = topFreqItems(topicFreq, len(topicDefs), n)

	// ── Channel breakdown ─────────────────────────────────────────────────────
	channelCount := make(map[string]int)
	for _, v := range videos {
		if v.ChannelTitle != "" {
			channelCount[v.ChannelTitle]++
		}
	}
	r.ChannelStyles = topFreqItems(channelCount, 15, n)

	return r
}

// ─────────────────────────────────────────────────────────────────────────────
// PrintSummary
// ─────────────────────────────────────────────────────────────────────────────

func PrintSummary(r *PatternReport) {
	sep := "────────────────────────────────────────────────────"
	header := func(s string) {
		pad := 48 - len(s)
		if pad < 0 {
			pad = 0
		}
		fmt.Printf("\n── %s %s\n", s, sep[:pad])
	}

	fmt.Printf("\n╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("║          📊  TITLE PATTERN ANALYSIS REPORT           ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n")
	fmt.Printf("\nTotal titles analysed: %d\n", r.TotalTitles)

	header("Length (characters)")
	fmt.Printf("  Min %-4d  Max %-4d  Mean %-5.1f  Median %-4d  P90 %d\n",
		r.LengthMin, r.LengthMax, r.LengthMean, r.LengthP50, r.LengthP90)

	header("Word Count")
	fmt.Printf("  Mean %.1f   Median %d\n", r.WordCountMean, r.WordCountP50)

	header("Punctuation & Formatting")
	fmt.Printf("  Colon (:)       %5.1f%%\n", r.ColonUsagePct)
	fmt.Printf("  Question (?)    %5.1f%%\n", r.QuestionPct)
	fmt.Printf("  Brackets []()   %5.1f%%\n", r.BracketPct)
	fmt.Printf("  Numbers         %5.1f%%\n", r.NumberInTitlePct)
	fmt.Printf("  Pipe (|)        %5.1f%%\n", r.PipePct)
	fmt.Printf("  Ellipsis (...)  %5.1f%%\n", r.EllipsisPct)
	fmt.Printf("  ALL CAPS word   %5.1f%%\n", r.AllCapsWordsPct)

	header("Framing & Specificity")
	fmt.Printf("  Positive framing    %5.1f%%\n", r.PositiveFramingPct)
	fmt.Printf("  Negative framing    %5.1f%%\n", r.NegativeFramingPct)
	fmt.Printf("  Mentions time       %5.1f%%\n", r.HasTimePct)
	fmt.Printf("  Mentions money/$    %5.1f%%\n", r.HasMoneyPct)
	fmt.Printf("  Starts with number  %5.1f%%\n", r.HasListNumPct)

	header("Structural Templates (top 15 with examples)")
	for i, t := range r.Templates {
		if i >= 15 {
			break
		}
		fmt.Printf("  %-42s %4d  (%5.1f%%)\n", t.Name, t.Count, t.Pct)
		if examples, ok := r.TemplateExamples[t.Name]; ok {
			for _, ex := range examples {
				fmt.Printf("      → %s\n", truncate(ex, 80))
			}
		}
	}

	header("Emotional Triggers")
	for _, e := range r.EmotionalTriggers {
		fmt.Printf("  %-35s %4d  (%5.1f%%)\n", e.Text, e.Count, e.Pct)
	}

	header("Curiosity Gap Patterns")
	for _, c := range r.CuriosityGaps {
		if c.Count == 0 {
			continue
		}
		fmt.Printf("  %-42s %4d  (%5.1f%%)\n", c.Text, c.Count, c.Pct)
	}

	header("Power Words (top 15)")
	for i, pw := range r.PowerWords {
		if i >= 15 {
			break
		}
		fmt.Printf("  %-20s %4d  (%5.1f%%)\n", pw.Text, pw.Count, pw.Pct)
	}

	header("Most Common First Words (top 15)")
	for i, fw := range r.FirstWords {
		if i >= 15 {
			break
		}
		fmt.Printf("  %-20s %4d  (%5.1f%%)\n", fw.Text, fw.Count, fw.Pct)
	}

	header("Most Common Lead Phrases / 2-word openers (top 15)")
	for i, lp := range r.LeadPhrases {
		if i >= 15 {
			break
		}
		fmt.Printf("  %-28s %4d  (%5.1f%%)\n", lp.Text, lp.Count, lp.Pct)
	}

	header("Top Bigrams (meaningful 2-word phrases, top 15)")
	for i, b := range r.Bigrams {
		if i >= 15 {
			break
		}
		fmt.Printf("  %-28s %4d  (%5.1f%%)\n", b.Text, b.Count, b.Pct)
	}

	header("Top Trigrams (meaningful 3-word phrases, top 15)")
	for i, t := range r.Trigrams {
		if i >= 15 {
			break
		}
		fmt.Printf("  %-35s %4d  (%5.1f%%)\n", t.Text, t.Count, t.Pct)
	}

	header("Topic Clusters")
	for _, tc := range r.TopicClusters {
		if tc.Count == 0 {
			continue
		}
		fmt.Printf("  %-35s %4d  (%5.1f%%)\n", tc.Text, tc.Count, tc.Pct)
	}

	header("Top Channels in Your Collection")
	for i, ch := range r.ChannelStyles {
		if i >= 15 {
			break
		}
		fmt.Printf("  %-40s %4d videos\n", ch.Text, ch.Count)
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
		if cnt == 0 {
			continue
		}
		items = append(items, FreqItem{Text: text, Count: cnt, Pct: pct(cnt, total)})
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

// tokenize lowercases and strips punctuation for n-gram analysis.
func tokenize(s string) []string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Fields(b.String())
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
