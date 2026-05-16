# headliner CLI

A Go CLI tool that analyses the structural patterns in your YouTube liked-video titles and uses an LLM to generate click-optimized title candidates for your own content.

## How it works

1. **`fetch`** — Authenticates with YouTube via OAuth2 and downloads the titles of all your liked videos, paginating through the full history. Titles are cached to `~/.headliner/titles.json`.
2. **`analyze`** — Reads the cached titles and runs a structural analysis: template frequencies, punctuation patterns, power words, lead phrases, length distributions, and channel breakdown.
3. **`generate`** — Takes an article, transcript, or topic summary and asks an LLM (default: `gpt-4o`) to write title candidates that match the proven patterns from your own watch history.

---

## Prerequisites

| Requirement | Why |
|---|---|
| Go 1.22+ | Build the CLI |
| Google Cloud OAuth2 credentials | Authenticate with YouTube |
| OpenAI API key | Generate titles via GPT-4o |

---

## Setup

### 1. Create Google OAuth2 credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com/) → **APIs & Services → Credentials**
2. Create an **OAuth 2.0 Client ID** of type **Desktop app**
3. Download the credentials and note your **Client ID** and **Client Secret**
4. Enable the **YouTube Data API v3** in your project

### 2. Configure credentials

**Option A — environment variables (recommended)**

```bash
export GOOGLE_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export GOOGLE_CLIENT_SECRET="your-client-secret"
export OPENAI_API_KEY="sk-..."
```

**Option B — config file**

Create `.headliner.yaml` in your home directory or the directory where you run the CLI:

```yaml
google_client_id: "your-client-id.apps.googleusercontent.com"
google_client_secret: "your-client-secret"
openai_api_key: "sk-..."
openai_model: "gpt-4o"          # optional, default: gpt-4o
cache_dir: "~/.headliner"       # optional
```

### 3. Build & install

```bash
cd cli
go build -o headliner .
# Move to a directory in your PATH, e.g.:
mv headliner /usr/local/bin/
```

---

## Usage

### Step 1 — Fetch your YouTube titles

```bash
headliner fetch
```

On first run a browser window opens for Google OAuth2 consent. After authorizing, an access token is cached at `~/.headliner/token.json` and all your liked video titles are saved to `~/.headliner/titles.json`.

```bash
# Re-fetch (e.g. after liking new videos)
headliner fetch --force
```

### Step 2 — Analyse the patterns

```bash
headliner analyze
```

Prints a report like:

```
── Length (characters) ───────────────────────────────
  Min: 8   Max: 97   Mean: 47.3   P50: 46   P90: 71

── Structural Templates (top 10) ────────────────────
  How To                              312  ( 24.1%)
  Number List                         198  ( 15.3%)
  Title: Subtitle (Colon Split)       167  ( 12.9%)
  ...

── Power Words (top 10) ─────────────────────────────
  best                               87  (  6.7%)
  how                                79  (  6.1%)
  ...
```

Save the full report as JSON:

```bash
headliner analyze --json > report.json
headliner analyze -o report.json   # saves JSON and still prints to terminal
```

### Step 3 — Generate titles

```bash
# From a file
headliner generate --input article.txt

# From stdin
cat transcript.txt | headliner generate --count 10

# Interactive (paste content, press Ctrl-D)
headliner generate

# With tone and model options
headliner generate --input notes.txt --count 8 --tone "educational" --model gpt-4o
```

Example output:

```
╔══════════════════════════════════════════════════════╗
║               🎬  TITLE CANDIDATES                   ║
╚══════════════════════════════════════════════════════╝

   1.  How I Built a SaaS in 30 Days (And What Almost Broke Me)
   2.  7 Mistakes I Made Launching My First Product — Don't Do This
   3.  The Honest Truth About Building in Public
   4.  Why Most Side Projects Fail: The Uncomfortable Reality
   5.  From Idea to $1k MRR: Everything I Learned
```

---

## All flags

### `headliner fetch`

| Flag | Default | Description |
|---|---|---|
| `--force` | false | Re-fetch even if a cache already exists |
| `--config` | auto | Path to `.headliner.yaml` config file |

### `headliner analyze`

| Flag | Default | Description |
|---|---|---|
| `--json` | false | Print full report as JSON to stdout |
| `-o, --out` | — | Save JSON report to this file path |

### `headliner generate`

| Flag | Default | Description |
|---|---|---|
| `-i, --input` | stdin | Path to article, transcript, or topic summary |
| `-n, --count` | 5 | Number of title candidates to generate |
| `--model` | gpt-4o | OpenAI model name |
| `--tone` | — | Tone hint: `educational`, `entertaining`, `technical`, etc. |
| `--extras` | — | Extra instructions appended to the LLM prompt |
| `--skip-fetch` | false | Suppress the "run fetch first" warning |

---

## Known limitations

- **Watch Later playlist** — The YouTube Data API v3 does not expose the Watch Later playlist for OAuth users (a known Google restriction). The CLI focuses on **Liked Videos** which is fully accessible.
- **50-per-page cap** — The CLI fully paginates via `nextPageToken`, so all liked videos are fetched regardless of library size.
- **Token refresh** — Tokens are stored in `~/.headliner/token.json`. If the refresh token expires, delete this file and run `headliner fetch` again to re-authorize.
