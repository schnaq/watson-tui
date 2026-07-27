# watson-tui

A lean terminal UI for [Watson](https://jazzband.github.io/Watson/) time
tracking, in Go. It reads and writes Watson's data format directly — you do not
need Watson installed, and existing data keeps working as it is.

## Installation

```sh
brew tap schnaq/tap
brew install --cask watson-tui
```

The one-liner `brew install --cask schnaq/tap/watson-tui` works too, but
Homebrew has deprecated it (see [Tap-Trust](https://docs.brew.sh/Tap-Trust)).

Updates come through `brew upgrade`. Homebrew casks are macOS-only; on Linux,
grab the matching archive from a
[release](https://github.com/schnaq/watson-tui/releases) (`linux_amd64` or
`linux_arm64`) and put `watson-tui` on your `PATH`.

## Usage

Start `watson-tui` — it opens on the current week.

| Key | Action |
|---|---|
| `j`/`k` | move |
| `enter` | edit frame |
| `n` | new frame |
| `d` | delete frame |
| `s` | start/stop timer |
| `S` | discard timer |
| `/` | filter (project, tag, ID) |
| `←` / `→` (or `[` / `]`) | previous/next period |
| `t`/`w`/`m`/`a` | day/week/month/everything |
| `r` | report |
| `o` | overview (billing) |
| `R` | reload |
| `?` | help |
| `q` | quit |

The header shows which period you are looking at — the angles `‹ ›` mean `←`
and `→` shift it (`[` and `]` do the same) — plus that period's total, a
comparison against the previous period and the next larger one (for a week,
that is the previous week and the month), the number of frames and projects,
and any running timer. `all frames` cannot be shifted, so it gets neither
angles nor a comparison. The list header's total matches the list below it and
therefore leaves a running timer out; when `+ running` follows it, that is exactly
why the report and the overview show more for the same period — their headers
count it, because their tables do too.

The interface uses your terminal theme's ANSI colours: context header on top,
framed main panel, key hints at the bottom. The chrome hands its space back to
the list in four steps: from 24 terminal lines the header is framed and shows
four field rows above two hint lines; between 20 and 23 lines it stays framed
with three field rows and one hint line; between 14 and 19 it collapses to a
single line; below 14 it disappears. When the hints do not all fit the width,
the trailing ones drop out entirely rather than ending mid-word; when only one
hint line is left, the period keys and the two exits `?` and `q` move into it.
`?` shows the full key map; it fits any terminal that still has a header (from
14 lines up), below that its tail is cut off.

The frame list adapts to the width as well, in this order: from about 88
columns the tags shrink (with `…`), below about 60 they are gone and the
project name shrinks, below about 42 the ID column drops **entirely**, below
about 24 the start and end times go too. Only labels are ever shortened. The
duration always stays complete, and the ID is shown either with all seven
characters or not at all: Watson resolves IDs by their prefix, so a shortened ID
would match several frames — or the wrong one. You always get the full ID in the
edit dialog (`enter`) and in the delete prompt (`d`). The exact thresholds
depend on the widest duration in the list — `130h 00m` needs one column more
than `6h 30m`.

Data directory: `$WATSON_DIR`, otherwise the OS default (macOS:
`~/Library/Application Support/watson`). Override with `--dir PATH`.

### Report and overview

The report (`r`) sums the selected period per project, with the tag totals
below each one. Tags are counted individually: a frame with two tags counts
towards both, so the tag totals can exceed the project total. They answer "how
much time per tag", not "how does this project break down".

The overview (`o`) is the billing view: totals per project across this week,
last week, this month, last month and everything. The table adapts to the
terminal width: below about 78 columns the month column titles are shortened
(`prev mo`), below about 70 the project column shrinks as well.
Below about 62 columns whole value columns drop out — which ones is printed
above the table, and `all` always stays. No duration is ever cut off.

**How time is attributed to a period** — this matters if you write invoices
from it:

- A frame counts in full towards the period it **starts** in. A session from
  31 July 23:00 to 1 August 02:00 therefore counts three hours towards July,
  not split across the two. Watson itself does the same.
- A running timer counts up to now and is called out below the table — in the
  overview and the report alike, and in their header totals as well. Only the
  frame list leaves it out of its header total, so that total still adds up to
  the day totals below it, and writes `+ running` after it.
- Everything is based on your local time zone; the week start comes from
  Watson's `config` (`[options] week_start`, Monday by default).

## Development

```sh
go test ./...
WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui
```

Releases: push a `v*` git tag — GitHub Actions builds with GoReleaser and
updates the Homebrew cask in
[schnaq/homebrew-tap](https://github.com/schnaq/homebrew-tap).
