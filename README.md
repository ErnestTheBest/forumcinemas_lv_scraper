# ForumCinemas Movie Reporter 🎬

A Dockerized Node.js app that scrapes ForumCinemas for currently playing movies and generates a beautiful HTML report with rating, year, and genres fetched from OMDb by IMDb ID.

## 📋 Table of Contents
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Quick Start (Docker)](#quick-start-docker)
- [Usage](#usage)
- [What the Application Does](#what-the-application-does)
- [Output Files](#output-files)
- [Report Features](#report-features)
- [Debugging Output](#debugging-output)
- [Troubleshooting](#troubleshooting)
- [License](#license)

## Features

- **Scrapes ForumCinemas** for currently playing movies
- **Fetches rating, year, and genres** from OMDb by IMDb ID
- **Generates beautiful HTML report** with sortable columns
- **Clickable movie titles** linking to ForumCinemas pages
- **Docker support** for easy, reproducible runs

## Prerequisites

- **Docker Desktop** (or any Docker runtime)
- **OMDb API key**

## Quick Start (Docker)

If you have Docker installed, this is the simplest way to run the application:

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd forumcinemas_films
   ```

2. **Create `.env` from `.env.example` and set your OMDb key:**
   ```env
   OMDB_API_KEY=your_omdb_api_key
   ```

3. **Run with Docker:**
   ```bash
   docker-compose up --build
   ```

That's it! The application will:
- Install all dependencies automatically
- Scrape ForumCinemas for current movies
- Fetch rating, year, and genres from OMDb
- Generate a beautiful HTML report
- Exit when complete

**Benefits of Docker approach:**
- ✅ **No local Node.js installation required**
- ✅ **No dependency conflicts**
- ✅ **Works on any machine with Docker**
- ✅ **No browser automation required**

<!-- Local installation instructions removed: Docker-only workflow -->

## Usage

```bash
# Run once and exit
docker-compose up --build

# Run in background
docker-compose up -d

# Stop when done
docker-compose down
```

## What the Application Does

1. **Scrapes ForumCinemas** now-playing page for movie links
2. **Extracts movie details** (title, tentative year, genres, IMDb link)
3. **Fetches rating, year, and genres** from OMDb using the IMDb ID
4. **Generates HTML report** with all data in a sortable table

## Output Files

- **`data/now_playing.json`** - List of movie detail page URLs
- **`data/movies_enriched.json`** - Complete movie data with IMDb details (rating, year, genres)
- **`index.html`** - Beautiful, sortable HTML report with clickable movie titles (ready for GitHub Pages)

## Report Features

The generated HTML report includes:
- **Sortable columns** (click headers to sort)
- **Responsive design** (works on mobile and desktop)
- **Beautiful styling** with gradients and hover effects
- **Statistics** showing movie count and rating coverage
- **Clickable movie titles** linking to ForumCinemas pages
- **Direct links** to IMDb pages

## Debugging Output

The script provides detailed logging:
- `📅 Scraped release year: XXXX` - Year inferred from ForumCinemas page
- `📅 Updated release year to XXXX from OMDb` - Year corrected using OMDb
- `🎭 Fetching OMDb details for ttXXXXXXX...` - OMDb lookup by IMDb ID
- `⚠️ No release year found from scraping` - When no year could be extracted

## Troubleshooting

### Docker Issues
- **Make sure Docker Desktop is running**
- **Rebuild after updates**: `docker-compose up --build`

### General Issues
- **Check the console output** for detailed error messages
- **Ensure network access** (needs ForumCinemas + OMDb)
- **Set `OMDB_API_KEY`** in `.env` locally and as a GitHub Actions repository secret in CI
- **Ensure write permissions** in the project directory

## License

MIT License - feel free to modify and distribute!
