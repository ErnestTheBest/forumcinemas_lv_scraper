package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const maxResponseBytes = 5 << 20

var imdbPattern = regexp.MustCompile(`/title/(tt\d+)`)

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type omdbResponse struct {
	Response   string `json:"Response"`
	Error      string `json:"Error"`
	Title      string `json:"Title"`
	Year       string `json:"Year"`
	Genre      string `json:"Genre"`
	IMDbRating string `json:"imdbRating"`
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

func scrapeNowPlaying(ctx context.Context, client httpClient, pageURL string) ([]movieLink, error) {
	doc, err := fetchDocument(ctx, client, pageURL)
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(pageURL)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	links := make([]movieLink, 0, 64)
	doc.Find(`a[href*="/title/"]`).Each(func(_ int, selection *goquery.Selection) {
		title := strings.TrimSpace(selection.Text())
		href, ok := selection.Attr("href")
		if !ok || title == "" || len(title) >= 1000 ||
			strings.Contains(title, "Buy tickets") ||
			strings.Contains(title, "Trailer") ||
			!strings.Contains(href, "/eng/event/") ||
			!strings.Contains(href, "/title/") {
			return
		}

		relative, parseErr := url.Parse(href)
		if parseErr != nil {
			return
		}
		absolute := base.ResolveReference(relative).String()
		if !seen[absolute] {
			seen[absolute] = true
			links = append(links, movieLink{Title: title, URL: absolute})
		}
	})
	return links, nil
}

func scrapeMovie(ctx context.Context, client httpClient, link movieLink) (movie, error) {
	doc, err := fetchDocument(ctx, client, link.URL)
	if err != nil {
		return movie{}, err
	}

	title := strings.TrimSpace(doc.Find("h1.list-item-desc-title").First().Text())
	if title == "" {
		title = link.Title
	}

	var imdbID string
	doc.Find(`a[href*="imdb.com/title/"]`).EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		href, _ := selection.Attr("href")
		match := imdbPattern.FindStringSubmatch(href)
		if len(match) == 2 {
			imdbID = match[1]
			return false
		}
		return true
	})
	if imdbID == "" {
		return movie{}, fmt.Errorf("IMDb link not found")
	}

	return movie{
		Title:           title,
		IMDbID:          imdbID,
		IMDbURL:         "https://www.imdb.com/title/" + imdbID + "/",
		ForumCinemasURL: link.URL,
	}, nil
}

func enrichFromOMDb(ctx context.Context, client httpClient, item *movie, apiKey string) error {
	endpoint, _ := url.Parse("https://www.omdbapi.com/")
	query := endpoint.Query()
	query.Set("apikey", apiKey)
	query.Set("i", item.IMDbID)
	query.Set("r", "json")
	endpoint.RawQuery = query.Encode()

	body, err := get(ctx, client, endpoint.String())
	if err != nil {
		return fmt.Errorf("OMDb: %w", err)
	}

	var response omdbResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("decode OMDb: %w", err)
	}
	if response.Response == "False" {
		return fmt.Errorf("OMDb: %s", response.Error)
	}

	if response.Title != "" && response.Title != "N/A" {
		item.OMDbTitle = strings.TrimSpace(response.Title)
	}
	if year := firstYear(response.Year); year != nil {
		item.ReleaseYear = year
	}
	if response.Genre != "" && response.Genre != "N/A" {
		item.Genres = response.Genre
	}
	if rating, err := strconv.ParseFloat(response.IMDbRating, 64); err == nil {
		item.IMDbRating = &rating
	}
	fetchedAt := time.Now()
	item.OMDbFetchedAt = &fetchedAt
	return nil
}

func firstYear(value string) *int {
	for i := 0; i+4 <= len(value); i++ {
		year, err := strconv.Atoi(value[i : i+4])
		if err == nil && year >= 1888 && year <= time.Now().Year()+5 {
			return &year
		}
	}
	return nil
}

func fetchDocument(ctx context.Context, client httpClient, pageURL string) (*goquery.Document, error) {
	body, err := get(ctx, client, pageURL)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}
	return doc, nil
}

func get(ctx context.Context, client httpClient, target string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(time.Duration(attempt) * 500 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}

		request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("User-Agent", "forumcinemas-go-reporter/1.0")
		request.Header.Set("Accept", "text/html,application/json")

		response, err := client.Do(request)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
		response.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return body, nil
		}

		lastErr = fmt.Errorf("HTTP %d", response.StatusCode)
		if response.StatusCode != http.StatusTooManyRequests && response.StatusCode < 500 {
			break
		}
	}
	return nil, lastErr
}
