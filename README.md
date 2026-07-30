# Forum Cinemas Reporter

Fast Go scraper that builds a searchable, sortable report of movies currently
playing at Forum Cinemas, enriched with IMDb ratings from OMDb.

## Run

Create `.env` from the example and add your OMDb API key:

```env
OMDB_API_KEY=your_key
```

Then run:

```bash
docker compose up --build
```

The scraper uses six concurrent workers and writes:

- `data/now_playing.json`
- `data/movies_enriched.json`
- `index.html`

## Develop

Requires Go 1.26 or Docker.

```bash
go test ./...
go run .
```

Optional environment variables:

- `WORKERS` — concurrent movie workers, default `6`
- `DATA_DIR` — JSON output directory, default `data`
- `REPORT_PATH` — HTML report path, default `index.html`
