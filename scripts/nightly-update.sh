#!/bin/sh
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_dir"

export GOMAXPROCS="${GOMAXPROCS:-1}"
export WORKERS="${WORKERS:-1}"
export MIN_MOVIES="${MIN_MOVIES:-10}"
export GOCACHE="${GOCACHE:-$repo_dir/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$repo_dir/.cache/go-mod}"
mkdir -p "$GOCACHE" "$GOMODCACHE"

go_bin="${GO_BIN:-}"
if [ -z "$go_bin" ]; then
	go_bin=$(command -v go || true)
fi
if [ -z "$go_bin" ] || [ ! -x "$go_bin" ]; then
	echo "Go compiler not found; set GO_BIN in the systemd environment file" >&2
	exit 1
fi

git pull --ff-only origin main

needs_build=false
if [ ! -x reporter ]; then
	needs_build=true
elif find . -maxdepth 1 -type f \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \) -newer reporter -print -quit | grep -q .; then
	needs_build=true
fi

if [ "$needs_build" = true ]; then
	"$go_bin" build -trimpath -ldflags="-s -w" -o reporter .
fi

./reporter scrape

git add data/now_playing.json data/movies_enriched.json data/scrape_metadata.json
if git diff --staged --quiet; then
	echo "Movie data did not change"
	exit 0
fi

git -c user.name="Raspberry Pi Reporter" \
	-c user.email="action@github.com" \
	commit -m "Update movie data"
git push origin main
