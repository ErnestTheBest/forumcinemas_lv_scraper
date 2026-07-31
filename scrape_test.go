package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) Do(request *http.Request) (*http.Response, error) {
	return function(request)
}

func staticClient(body string) httpClient {
	return roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		}, nil
	})
}

func TestScrapeNowPlayingDeduplicatesLinks(t *testing.T) {
	html := `<a href="/eng/event/1/title/one/">One</a>
		<a href="/eng/event/1/title/one/">One</a>
		<a href="/eng/event/2/title/two/">Buy tickets</a>`

	links, err := scrapeNowPlaying(context.Background(), staticClient(html), nowPlayingURL)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].Title != "One" {
		t.Fatalf("unexpected links: %#v", links)
	}
}

func TestScrapeMovieFindsIMDbID(t *testing.T) {
	html := `<h1 class="list-item-desc-title">Movie</h1>
		<a href="https://www.imdb.com/title/tt1234567/">IMDb</a>`

	item, err := scrapeMovie(context.Background(), staticClient(html), movieLink{
		Title: "Fallback",
		URL:   "https://example.com/movie",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Title != "Movie" || item.IMDbID != "tt1234567" {
		t.Fatalf("unexpected movie: %#v", item)
	}
}

func TestScrapeApolloMoviesFiltersAkropoleAndDeduplicates(t *testing.T) {
	html := `<p class="movie-card__title"><a href="/eng/event/303993/the_odyssey?theatreAreaID=1011">The Odyssey</a></p>
		<p class="movie-card__title"><a href="/eng/event/303993/the_odyssey?theatreAreaID=1011">The Odyssey</a></p>
		<p class="movie-card__title"><a href="/eng/event/303993/the_odyssey?theatreAreaID=1013">The Odyssey</a></p>
		<p class="movie-card__title"><a href="https://example.com/eng/event/1/movie?theatreAreaID=1011">Other site</a></p>`

	movies, err := scrapeApolloMovies(context.Background(), staticClient(html), apolloMoviesURL)
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 1 {
		t.Fatalf("unexpected Apollo movies: %#v", movies)
	}
	if movies[0].Title != "The Odyssey" ||
		movies[0].URL != "https://www.apollokino.lv/eng/event/303993/the_odyssey?theatreAreaID=1011#section-shows" {
		t.Fatalf("unexpected Apollo movie: %#v", movies[0])
	}
}

func TestAttachApolloLinksUsesSafeNormalizedTitles(t *testing.T) {
	movies := []movie{
		{Title: "Minions & Monsters"},
		{Title: "Howl’s Moving Castle (2004)"},
		{Title: "Scary Movie", OMDbTitle: "Scary Movie 6"},
		{Title: "Ulya", OMDbTitle: "The Invite"},
	}
	apolloMovies := []apolloMovie{
		{Title: "Minions and Monsters", URL: "https://apollo/minions"},
		{Title: "Howl's Moving Castle", URL: "https://apollo/howl"},
		{Title: "Scary Movie 6", URL: "https://apollo/scary"},
		{Title: "The Invite", URL: "https://apollo/invite"},
	}

	if matched := attachApolloLinks(movies, apolloMovies); matched != 3 {
		t.Fatalf("matched %d movies, want 3", matched)
	}
	if movies[0].ApolloKinoURL != "https://apollo/minions" ||
		movies[1].ApolloKinoURL != "https://apollo/howl" ||
		movies[2].ApolloKinoURL != "https://apollo/scary" {
		t.Fatalf("unexpected matches: %#v", movies)
	}
	if movies[3].ApolloKinoURL != "" {
		t.Fatalf("unsafe OMDb fallback matched unrelated title: %#v", movies[3])
	}
}

func TestAttachApolloLinksRejectsAmbiguousTitles(t *testing.T) {
	movies := []movie{{Title: "Movie"}}
	apolloMovies := []apolloMovie{
		{Title: "Movie", URL: "https://apollo/one"},
		{Title: "Movie", URL: "https://apollo/two"},
	}

	if matched := attachApolloLinks(movies, apolloMovies); matched != 0 {
		t.Fatalf("matched %d movies, want 0", matched)
	}
	if movies[0].ApolloKinoURL != "" {
		t.Fatalf("ambiguous title received a link: %#v", movies[0])
	}
}

func TestWriteReportRendersSeparateCinemaColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.html")
	movies := []movie{{
		Title:           "The Odyssey",
		ForumCinemasURL: "https://forum/movie",
		ApolloKinoURL:   "https://apollo/movie#section-shows",
		IMDbURL:         "https://imdb/title",
	}}

	if err := writeReport(path, movies, time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)

	for _, expected := range []string{
		`<th>Forum</th><th>Apollo</th>`,
		`<td class="movie-title">The Odyssey</td>`,
		`href="https://forum/movie"`,
		`href="https://apollo/movie#section-shows"`,
	} {
		if !strings.Contains(report, expected) {
			t.Errorf("report does not contain %q", expected)
		}
	}
	if strings.Contains(report, `<td class="movie-title"><a`) {
		t.Error("movie title is still rendered as a hyperlink")
	}
}

func TestExecuteReportBuildsOfflineFromCommittedData(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	reportPath := filepath.Join(t.TempDir(), "index.html")
	generatedAt := time.Date(2026, time.July, 31, 4, 0, 0, 0, time.FixedZone("EEST", 3*60*60))

	movies := []movie{{
		Title:           "The Odyssey",
		ForumCinemasURL: "https://forum/movie",
		ApolloKinoURL:   "https://apollo/movie#section-shows",
		IMDbURL:         "https://imdb/title",
	}}
	metadata := scrapeMetadata{
		GeneratedAt:    generatedAt,
		ForumMovies:    1,
		EnrichedMovies: 1,
		ApolloMovies:   1,
		ApolloMatches:  1,
	}
	if err := writeJSONAtomic(filepath.Join(dataDir, "movies_enriched.json"), movies); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONAtomic(filepath.Join(dataDir, "scrape_metadata.json"), metadata); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("REPORT_PATH", reportPath)
	if err := execute([]string{"report"}); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)
	for _, expected := range []string{
		"Updated 31 Jul 2026",
		`href="https://apollo/movie#section-shows"`,
	} {
		if !strings.Contains(report, expected) {
			t.Errorf("report does not contain %q", expected)
		}
	}
}

func TestCopyEnrichmentReusesOMDbData(t *testing.T) {
	year := 2026
	rating := 8.4
	target := movie{Title: "Current title", IMDbID: "tt123"}
	cached := movie{
		OMDbTitle:   "OMDb title",
		ReleaseYear: &year,
		Genres:      "Adventure",
		IMDbRating:  &rating,
	}

	copyEnrichment(&target, cached)

	if target.OMDbTitle != cached.OMDbTitle ||
		target.ReleaseYear == nil || *target.ReleaseYear != year ||
		target.Genres != cached.Genres ||
		target.IMDbRating == nil || *target.IMDbRating != rating {
		t.Fatalf("cached enrichment was not copied: %#v", target)
	}
}

func TestDefaultWorkerCountProtectsRaspberryPi(t *testing.T) {
	if defaultWorkers != 1 {
		t.Fatalf("defaultWorkers = %d, want 1", defaultWorkers)
	}
}
