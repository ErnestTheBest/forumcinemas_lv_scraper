# Riga Cinema Reporter

Fast Go scraper that builds a searchable, sortable report of movies currently
playing at Forum Cinemas, enriched with IMDb ratings from OMDb and direct links
to matching movies at Apollo Kino Akropole Rīga.

The network scraper and HTML report builder are separate commands. The scraper
runs on the Raspberry Pi because Apollo Kino rejects GitHub-hosted runners. GitHub
Actions only builds the HTML report from committed JSON data.

## Local run

Create `.env` from the example and add your OMDb API key:

```bash
cp .env.example .env
```

Build the binary once:

```bash
/home/mrmiles/Code/.tools/go1.24.13/bin/go build -o reporter .
```

Load the environment and scrape sequentially:

```bash
set -a
. ./.env
set +a
./reporter scrape
```

The scraper uses one worker and one Go CPU by default. It reuses previously
enriched OMDb records, so unchanged movies do not consume additional OMDb
requests. It refuses to replace the last good data when Forum Cinemas returns
too few movies, Apollo Kino is unavailable, or no Apollo titles match.

The scraper writes:

- `data/now_playing.json`
- `data/movies_enriched.json`
- `data/scrape_metadata.json`

Build `index.html` without any network or OMDb requests:

```bash
./reporter report
```

## Daily Raspberry Pi timer

The included user timer runs at 04:00 in the computer's local timezone. A missed
run is started after the machine comes back online. The service is limited to
one Go CPU, 50% CPU time, idle I/O priority, and 384 MB of memory.

After the changes are on `main`, install it with:

```bash
./scripts/install-user-timer.sh
```

Then put the OMDb key in:

```text
~/.config/forumcinemas-reporter.env
```

Start and inspect the timer:

```bash
systemctl --user start forumcinemas-scraper.timer
systemctl --user list-timers forumcinemas-scraper.timer
journalctl --user -u forumcinemas-scraper.service
```

The timer works from a dedicated checkout at
`~/.local/share/forumcinemas-reporter`, so it does not touch an active development
working tree. After a successful scrape it commits only the JSON files and pushes
them to `main`; that push triggers the offline GitHub report build.

## Develop

Requires Go 1.24 or Docker.

```bash
/home/mrmiles/Code/.tools/go1.24.13/bin/go test ./...
./reporter report
```

Environment variables:

- `OMDB_API_KEY` — required by `scrape`
- `GOMAXPROCS` — Go CPU limit, configured as `1` on the Raspberry Pi
- `WORKERS` — concurrent movie workers, default `1`
- `MIN_MOVIES` — minimum safe result size, default `10`
- `DATA_DIR` — JSON directory, default `data`
- `REPORT_PATH` — HTML report path, default `index.html`

Forum Cinemas remains the source of the movie list. Apollo Kino is an optional
secondary link source in the report, but a failed Apollo fetch aborts publication
so a transient block cannot erase the last good links.
