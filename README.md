# Riga Cinema Reporter

Fast Go scraper that builds a searchable, sortable report of movies currently
playing at Forum Cinemas, enriched with IMDb ratings from OMDb and direct links
to matching movies at Apollo Kino Akropole Rīga.

## Run

Create `.env` from the example and add your OMDb API key:

```env
OMDB_API_KEY=your_key
```

Then run:

```bash
docker compose up --build
```

The scraper uses two concurrent workers by default and writes:

- `data/now_playing.json`
- `data/movies_enriched.json`
- `index.html`

## Develop

Requires Go 1.24 or Docker.

```bash
go test ./...
go run .
```

Optional environment variables:

- `WORKERS` — concurrent movie workers, default `2`
- `DATA_DIR` — JSON output directory, default `data`
- `REPORT_PATH` — HTML report path, default `index.html`

Forum Cinemas remains the source of the movie list. Apollo Kino is an optional
secondary source: if its catalogue is unavailable or a movie cannot be matched
confidently, the report is still generated without that Apollo link.
