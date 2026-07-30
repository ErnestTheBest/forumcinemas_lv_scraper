package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
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
