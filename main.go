package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultWorkers      = 2
	nowPlayingURL       = "https://www.forumcinemas.lv/eng/movies/now-playing"
	apolloTheatreAreaID = "1011"
	apolloMoviesURL     = "https://www.apollokino.lv/eng/movies?theatreAreaID=" + apolloTheatreAreaID
)

type movieLink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type movie struct {
	Title           string   `json:"title"`
	OMDbTitle       string   `json:"omdbTitle,omitempty"`
	ReleaseYear     *int     `json:"releaseYear"`
	Genres          string   `json:"genres"`
	IMDbID          string   `json:"imdbId"`
	IMDbURL         string   `json:"imdbUrl"`
	ForumCinemasURL string   `json:"forumcinemasUrl"`
	ApolloKinoURL   string   `json:"apolloKinoUrl,omitempty"`
	IMDbRating      *float64 `json:"imdbRating"`
}

type indexedLink struct {
	index int
	link  movieLink
}

type indexedMovie struct {
	index int
	movie movie
	err   error
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	apiKey := strings.TrimSpace(os.Getenv("OMDB_API_KEY"))
	if apiKey == "" {
		return errors.New("OMDB_API_KEY is not configured")
	}

	workers := envInt("WORKERS", defaultWorkers)
	dataDir := envString("DATA_DIR", "data")
	reportPath := envString("REPORT_PATH", "index.html")
	client := newHTTPClient()
	ctx := context.Background()

	log.Printf("Fetching now playing movies")
	links, err := scrapeNowPlaying(ctx, client, nowPlayingURL)
	if err != nil {
		return fmt.Errorf("scrape now playing: %w", err)
	}
	log.Printf("Found %d movies; processing with %d workers", len(links), workers)

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	if err := writeJSON(filepath.Join(dataDir, "now_playing.json"), links); err != nil {
		return err
	}

	movies := enrichMovies(ctx, client, links, apiKey, workers)

	log.Printf("Fetching Apollo Kino Akropole movies")
	apolloMovies, apolloErr := scrapeApolloMovies(ctx, client, apolloMoviesURL)
	if apolloErr != nil {
		log.Printf("Apollo Kino unavailable; continuing without Apollo links: %v", apolloErr)
	} else {
		matched := attachApolloLinks(movies, apolloMovies)
		log.Printf("Matched %d of %d Forum Cinemas movies with Apollo Kino Akropole", matched, len(movies))
	}

	if err := writeJSON(filepath.Join(dataDir, "movies_enriched.json"), movies); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	if err := writeReport(reportPath, movies, time.Now()); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	log.Printf("Done: %d movies written to %s and %s", len(movies), dataDir, reportPath)
	return nil
}

func enrichMovies(ctx context.Context, client httpClient, links []movieLink, apiKey string, workers int) []movie {
	jobs := make(chan indexedLink)
	results := make(chan indexedMovie)
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				item, err := scrapeMovie(ctx, client, job.link)
				if err == nil {
					err = enrichFromOMDb(ctx, client, &item, apiKey)
				}
				results <- indexedMovie{index: job.index, movie: item, err: err}
			}
		}()
	}

	go func() {
		for i, link := range links {
			jobs <- indexedLink{index: i, link: link}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	ordered := make([]*movie, len(links))
	for result := range results {
		if result.err != nil {
			log.Printf("Skip %q: %v", links[result.index].Title, result.err)
			continue
		}
		item := result.movie
		ordered[result.index] = &item
		log.Printf("Processed %q", item.Title)
	}

	movies := make([]movie, 0, len(ordered))
	for _, item := range ordered {
		if item != nil {
			movies = append(movies, *item)
		}
	}
	return movies
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func envString(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
