#!/bin/sh
set -eu

source_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
remote_url=$(git -C "$source_dir" remote get-url origin)
target_dir="${REPORTER_INSTALL_DIR:-$HOME/.local/share/forumcinemas-reporter}"
unit_dir="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
environment_file="${XDG_CONFIG_HOME:-$HOME/.config}/forumcinemas-reporter.env"

mkdir -p "$(dirname "$target_dir")" "$unit_dir"

if [ -d "$target_dir/.git" ]; then
	git -C "$target_dir" pull --ff-only origin main
elif [ -e "$target_dir" ]; then
	echo "$target_dir exists but is not a git checkout" >&2
	exit 1
else
	git clone --branch main "$remote_url" "$target_dir"
fi

install -m 0644 "$target_dir/systemd/forumcinemas-scraper.service" "$unit_dir/forumcinemas-scraper.service"
install -m 0644 "$target_dir/systemd/forumcinemas-scraper.timer" "$unit_dir/forumcinemas-scraper.timer"

if [ ! -f "$environment_file" ]; then
	umask 077
	{
		echo "OMDB_API_KEY="
		echo "GO_BIN=$HOME/Code/.tools/go1.24.13/bin/go"
	} > "$environment_file"
	echo "Created $environment_file; add OMDB_API_KEY before starting the timer."
fi

systemctl --user daemon-reload
systemctl --user enable forumcinemas-scraper.timer

if grep -q '^OMDB_API_KEY=..*' "$environment_file"; then
	systemctl --user start forumcinemas-scraper.timer
	systemctl --user list-timers forumcinemas-scraper.timer
else
	echo "Timer installed but not started because OMDB_API_KEY is empty in $environment_file."
fi
