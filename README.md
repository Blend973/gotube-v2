# gotube (gotube-v2)

gotube is a terminal-first YouTube search / playback / download utility written in Go. It uses yt-dlp to fetch video and playlist metadata, fzf for interactive selection and optional previews, and external players (mpv or vlc) for playback. It includes convenience features for downloading video/audio, search history, and preview image generation.

## Highlights / features

- Search YouTube from the terminal and browse results interactively (fzf integration).
- Play videos using mpv or vlc.
- Download videos or extract audio via yt-dlp.
- Optional previews (image/text) integrated into fzf.
- Search history and simple filters (recent hour/day/week/etc).
- Configurable via a simple config file placed under XDG config.

## Requirements

System commands the app expects to be present:

- yt-dlp
- fzf
- jq
- curl
- mpv or vlc (at least one is required)

The app checks for these commands on startup and will exit with an error if required dependencies are missing.

## Build / install

You can build the CLI with Go:

- With Go installed:

  - Clone the repo:
    git clone https://github.com/Blend973/gotube-v2.git
    cd gotube-v2

  - Build:
    go build -o yt ./cmd/yt
    sudo mv yt
  - Run:
    yt

- Or run directly:
  go run ./cmd/yt -S "lofi"

## Usage

Run the CLI (example after building):

- Start interactive launcher:
  yt

- Search directly from the command line:
  yt -S "lofi"
  yt --search "lofi"

- Edit the config file (opens configured editor):
  yt -e
  yt --edit-config

- Print version:
  yt -v
  yt --version

When launched, the main actions are "Search", "Edit Config", and "Exit". Searching will open an interactive results list (powered by fzf by default) with options to Play, Download, Autoplay, fetch related mixes, Next/Previous pages, etc.

Downloads are handled via yt-dlp; the download directory and options are configurable.

## Configuration

Configuration is stored in the XDG config directory for the app. Default config path (for the default CLI name `gotube`) is:

- $XDG_CONFIG_HOME/gotube/gotube.conf
  (defaults to `~/.config/gotube/gotube.conf` if XDG_CONFIG_HOME is not set)

Default configuration keys and their default values (as discovered in code):

- IMAGE_RENDERER: ""  
- EDITOR: env $EDITOR or `nano`  
- PREFERRED_SELECTOR: `fzf`  
- VIDEO_QUALITY: `720`  
- ENABLE_PREVIEW: `false`  
- PLAYER: `mpv`  
- PREFERRED_BROWSER: "" (used to pass browser cookie flags to yt-dlp)  
- NO_OF_SEARCH_RESULTS: `30`  
- NOTIFICATION_DURATION: `5`  
- SEARCH_HISTORY: `true`  
- DOWNLOAD_DIRECTORY: `$XDG_VIDEOS_DIR/gotube` (defaults to `~/Videos/gotube`)  
- AUDIO_ONLY_MODE: `false`  
- AUTOPLAY_MODE: `off`

The config loader will create required directories (config dir, cache dirs) when first run. The app also generates helper scripts for previews (helper script and preview dispatcher) in the config directory.

Preview-related paths (created by the app):

- Preview dispatcher script: $XDG_CONFIG_HOME/gotube/yt-x-preview.sh  
- Helper script: $XDG_CONFIG_HOME/gotube/yt-x-helper.sh  
- Preview images cache: $XDG_CACHE_HOME/gotube/preview_images  
- Preview text cache: $XDG_CACHE_HOME/gotube/preview_text

Environment variables that affect behavior:

- YT_X_APP_NAME — override the CLI name (used for config directory naming)
- YT_X_FZF_OPTS — override FZF default options used by the app
- KITTY_WINDOW_ID — if set, the app may choose `icat` for IMAGE_RENDERER

## Examples

- Search and immediately open results:
  yt -S "lofi hip hop"

- Download a video audio-only mode via interactive GUI:
  Launch ./gotube, search, select a video and choose "Download" (audio mode can be set in config or via selection)

- Edit config:
  yt -e

- Build and run:
  go build -o yt ./cmd/yt && ./yt

## Testing

This project contains some unit tests (for example internal/download). Run tests with:

  go test ./...

or run package-specific tests:

  go test ./internal/download

## Troubleshooting

- Missing dependencies error on startup:
  The app checks for yt-dlp, fzf, jq, curl and mpv OR vlc. Install missing packages via your package manager (apt, brew, pacman, etc) or otherwise ensure the binaries are in PATH.

- Preview images not shown:
  Ensure IMAGE_RENDERER in config is set appropriately (e.g., `chafa` for terminal image rendering or `icat` if using kitty), and that preview feature is enabled in config (`ENABLE_PREVIEW=true`) and selector is `fzf`.

## Contributing

- Open issues or pull requests in the repository:
  https://github.com/Blend973/gotube-v2/issues

- Coding style:
  The project is written in Go. Please follow gofmt and typical Go idioms. Run `go test ./...` to ensure tests pass.

## License

- No license file was found in the repository during analysis. If you are the project owner, add a LICENSE file to specify the project's license and how others can use it.

## Maintainers / Contact

- Repository: https://github.com/Blend973/gotube-v2
- For issues and feature requests: https://github.com/Blend973/gotube-v2/issues
