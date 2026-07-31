package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

type apolloMovie struct {
	Title string
	URL   string
}

func scrapeApolloMovies(ctx context.Context, client httpClient, pageURL string) ([]apolloMovie, error) {
	doc, err := fetchDocument(ctx, client, pageURL)
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(pageURL)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	movies := make([]apolloMovie, 0, 32)
	doc.Find(".movie-card__title a").Each(func(_ int, selection *goquery.Selection) {
		title := strings.TrimSpace(selection.Text())
		href, ok := selection.Attr("href")
		if !ok || title == "" || len(title) >= 1000 {
			return
		}

		relative, parseErr := url.Parse(href)
		if parseErr != nil {
			return
		}
		absolute := base.ResolveReference(relative)
		if !strings.EqualFold(absolute.Hostname(), "www.apollokino.lv") ||
			!strings.Contains(absolute.Path, "/eng/event/") ||
			absolute.Query().Get("theatreAreaID") != apolloTheatreAreaID {
			return
		}

		absolute.Fragment = "section-shows"
		target := absolute.String()
		if seen[target] {
			return
		}
		seen[target] = true
		movies = append(movies, apolloMovie{Title: title, URL: target})
	})

	if len(movies) == 0 {
		return nil, fmt.Errorf("no Apollo Kino Akropole movies found")
	}
	return movies, nil
}

func attachApolloLinks(movies []movie, apolloMovies []apolloMovie) int {
	byTitle := make(map[string][]apolloMovie)
	for _, item := range apolloMovies {
		key := normalizeMovieTitle(item.Title)
		if key != "" {
			byTitle[key] = append(byTitle[key], item)
		}
	}

	matched := 0
	for i := range movies {
		keys := []string{normalizeMovieTitle(movies[i].Title)}
		omdbKey := normalizeMovieTitle(movies[i].OMDbTitle)
		if omdbKey != "" && titlesCompatible(keys[0], omdbKey) {
			keys = append(keys, omdbKey)
		}

		candidates := make(map[string]apolloMovie)
		for _, key := range keys {
			for _, candidate := range byTitle[key] {
				candidates[candidate.URL] = candidate
			}
		}
		if len(candidates) != 1 {
			continue
		}
		for _, candidate := range candidates {
			movies[i].ApolloKinoURL = candidate.URL
			matched++
		}
	}
	return matched
}

func normalizeMovieTitle(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 6 && value[len(value)-1] == ')' {
		open := len(value) - 6
		if open >= 0 && value[open] == '(' && isFourDigits(value[open+1:len(value)-1]) {
			value = strings.TrimSpace(value[:open])
		}
	}

	var normalized strings.Builder
	spacePending := false
	for _, char := range strings.ToLower(value) {
		switch {
		case char == '&':
			appendWord(&normalized, "and", &spacePending)
		case char == '\'' || char == '’' || char == '´':
			// Apostrophes do not separate words.
		case unicode.IsLetter(char) || unicode.IsDigit(char):
			if spacePending && normalized.Len() > 0 {
				normalized.WriteByte(' ')
			}
			normalized.WriteRune(char)
			spacePending = false
		default:
			spacePending = normalized.Len() > 0
		}
	}
	return normalized.String()
}

func appendWord(builder *strings.Builder, word string, spacePending *bool) {
	if (*spacePending || builder.Len() > 0) && builder.Len() > 0 {
		builder.WriteByte(' ')
	}
	builder.WriteString(word)
	*spacePending = true
}

func isFourDigits(value string) bool {
	if len(value) != 4 {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func titlesCompatible(forumTitle, omdbTitle string) bool {
	if forumTitle == omdbTitle {
		return true
	}
	shorter, longer := forumTitle, omdbTitle
	if len(shorter) > len(longer) {
		shorter, longer = longer, shorter
	}
	if !strings.HasPrefix(longer, shorter+" ") {
		return false
	}
	remainder := strings.TrimPrefix(longer, shorter+" ")
	for _, char := range remainder {
		if !unicode.IsDigit(char) && !unicode.IsSpace(char) {
			return false
		}
	}
	return remainder != ""
}
