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
	defaultWorkers      = 1
	defaultMinMovies    = 10
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

type scrapeMetadata struct {
	GeneratedAt    time.Time `json:"generatedAt"`
	ForumMovies    int       `json:"forumMovies"`
	EnrichedMovies int       `json:"enrichedMovies"`
	ApolloMovies   int       `json:"apolloMovies"`
	ApolloMatches  int       `json:"apolloMatches"`
}

func main() {
	if err := execute(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func execute(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: reporter [scrape|report]")
	}

	command := "scrape"
	if len(args) == 1 {
		command = strings.TrimSpace(args[0])
	}

	switch command {
	case "scrape":
		return runScrape()
	case "report":
		return runReport()
	default:
		return fmt.Errorf("unknown command %q; usage: reporter [scrape|report]", command)
	}
}

func runScrape() error {
	apiKey := strings.TrimSpace(os.Getenv("OMDB_API_KEY"))
	if apiKey == "" {
		return errors.New("OMDB_API_KEY is not configured")
	}

	workers := envInt("WORKERS", defaultWorkers)
	minMovies := envInt("MIN_MOVIES", defaultMinMovies)
	dataDir := envString("DATA_DIR", "data")
	client := newHTTPClient()
	ctx := context.Background()

	log.Printf("Fetching now playing movies")
	links, err := scrapeNowPlaying(ctx, client, nowPlayingURL)
	if err != nil {
		return fmt.Errorf("scrape now playing: %w", err)
	}
	if len(links) < minMovies {
		return fmt.Errorf("scrape now playing: found %d movies, require at least %d; keeping previous data", len(links), minMovies)
	}
	log.Printf("Found %d movies; processing with %d workers", len(links), workers)

	cache := loadMovieCache(filepath.Join(dataDir, "movies_enriched.json"))
	movies := enrichMovies(ctx, client, links, apiKey, workers, cache)
	if len(movies) < minMovies {
		return fmt.Errorf("enrichment produced %d movies, require at least %d; keeping previous data", len(movies), minMovies)
	}

	log.Printf("Fetching Apollo Kino Akropole movies")
	apolloMovies, err := scrapeApolloMovies(ctx, client, apolloMoviesURL)
	if err != nil {
		return fmt.Errorf("scrape Apollo Kino: %w; keeping previous data", err)
	}
	matched := attachApolloLinks(movies, apolloMovies)
	if matched == 0 {
		return errors.New("Apollo Kino returned movies but none matched Forum Cinemas; keeping previous data")
	}
	log.Printf("Matched %d of %d Forum Cinemas movies with Apollo Kino Akropole", matched, len(movies))

	generatedAt := time.Now()
	metadata := scrapeMetadata{
		GeneratedAt:    generatedAt,
		ForumMovies:    len(links),
		EnrichedMovies: len(movies),
		ApolloMovies:   len(apolloMovies),
		ApolloMatches:  matched,
	}

	if err := writeJSONAtomic(filepath.Join(dataDir, "now_playing.json"), links); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(dataDir, "movies_enriched.json"), movies); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(dataDir, "scrape_metadata.json"), metadata); err != nil {
		return err
	}

	log.Printf("Done: %d movies written to %s; run 'reporter report' to build the HTML report", len(movies), dataDir)
	return nil
}

func runReport() error {
	dataDir := envString("DATA_DIR", "data")
	reportPath := envString("REPORT_PATH", "index.html")

	var movies []movie
	if err := readJSON(filepath.Join(dataDir, "movies_enriched.json"), &movies); err != nil {
		return err
	}
	if len(movies) == 0 {
		return errors.New("cannot build report: movies_enriched.json contains no movies")
	}

	generatedAt := time.Now()
	var metadata scrapeMetadata
	if err := readJSON(filepath.Join(dataDir, "scrape_metadata.json"), &metadata); err == nil && !metadata.GeneratedAt.IsZero() {
		generatedAt = metadata.GeneratedAt
	}

	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	if err := writeReport(reportPath, movies, generatedAt); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	log.Printf("Done: report for %d movies written to %s", len(movies), reportPath)
	return nil
}

func enrichMovies(
	ctx context.Context,
	client httpClient,
	links []movieLink,
	apiKey string,
	workers int,
	cache map[string]movie,
) []movie {
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
					if cached, ok := cache[item.IMDbID]; ok {
						copyEnrichment(&item, cached)
					} else {
						err = enrichFromOMDb(ctx, client, &item, apiKey)
					}
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

func copyEnrichment(target *movie, cached movie) {
	target.OMDbTitle = cached.OMDbTitle
	target.ReleaseYear = cached.ReleaseYear
	target.Genres = cached.Genres
	target.IMDbRating = cached.IMDbRating
}

func loadMovieCache(path string) map[string]movie {
	var movies []movie
	if err := readJSON(path, &movies); err != nil {
		log.Printf("OMDb cache unavailable; all movies will be refreshed: %v", err)
		return nil
	}

	cache := make(map[string]movie, len(movies))
	for _, item := range movies {
		if item.IMDbID != "" {
			cache[item.IMDbID] = item
		}
	}
	log.Printf("Loaded %d cached OMDb records", len(cache))
	return cache
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}

	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("create temporary %s: %w", path, err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)

	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("write temporary %s: %w", path, err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync temporary %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary %s: %w", path, err)
	}
	if err := os.Chmod(tempPath, 0o644); err != nil {
		return fmt.Errorf("set permissions on temporary %s: %w", path, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
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
