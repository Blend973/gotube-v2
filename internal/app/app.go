package app

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"yt/internal/config"
	"yt/internal/download"
	"yt/internal/history"
	"yt/internal/player"
	"yt/internal/playlist"
	"yt/internal/preview"
	"yt/internal/ui"
	"yt/internal/util"
	"yt/internal/ytdlp"
)

var ErrExit = errors.New("exit")

var filterPattern = regexp.MustCompile(`^(:[a-z]+)\s+(.+)`)

type App struct {
	State         *config.State
	Page          playlist.Window
	playerRunning atomic.Bool
}

func New(state *config.State) (*App, error) {
	a := &App{State: state, Page: playlist.NewWindow(state.NoOfResults())}
	if err := preview.CreateBashHelpers(state.Paths, state.Config["IMAGE_RENDERER"]); err != nil {
		return nil, err
	}
	preview.CleanupCache(state.Paths)
	return a, nil
}

func (a *App) Run(initialAction string, searchTerm string) error {
	return a.mainMenu(initialAction, searchTerm)
}

func (a *App) IsPlayerRunning() bool {
	return a.playerRunning.Load()
}

func (a *App) mainMenu(initialAction string, searchTerm string) error {
	action := initialAction
	term := searchTerm
	for {
		util.ClearScreen()
		if action == "" {
			sel := ui.Launcher([]string{"Search", "Edit Config", "Exit"}, "Select Action", nil)
			action = strings.TrimSpace(sel)
		}

		switch action {
		case "Exit", "":
			return ErrExit
		case "Search":
			if err := a.runSearch(term); err != nil {
				if errors.Is(err, ErrExit) {
					return err
				}
				a.sendNotification("Failed to process search")
			}
			action = ""
			term = ""
		case "Edit Config":
			if err := a.editConfig(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: edit config failed: %v\n", err)
			}
			action = ""
			term = ""
		default:
			action = ""
		}
	}
}

func (a *App) editConfig() error {
	editor := strings.TrimSpace(a.State.Config["EDITOR"])
	if editor == "" {
		editor = "nano"
	}
	cmd := exec.Command(editor, a.State.Paths.ConfigFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	st, err := config.Load(a.State.CLIName, a.State.CLIVersion)
	if err != nil {
		return err
	}
	a.State = st
	a.Page = playlist.NewWindow(st.NoOfResults())
	if err := preview.CreateBashHelpers(st.Paths, st.Config["IMAGE_RENDERER"]); err != nil {
		return err
	}
	preview.CleanupCache(st.Paths)
	return nil
}

func (a *App) runSearch(searchTerm string) error {
	util.ClearScreen()
	if strings.TrimSpace(searchTerm) == "" {
		historyText := ""
		if strings.EqualFold(a.State.Config["SEARCH_HISTORY"], "true") {
			lines, _ := history.Read(a.State.Paths.SearchHistoryFile)
			historyText = history.FormatLatest(lines, 10)
		}
		searchTerm = ui.Prompt("Enter term to search for", "", historyText)

		if strings.HasPrefix(searchTerm, "!") {
			lines, _ := history.Read(a.State.Paths.SearchHistoryFile)
			searchTerm = history.ResolveBangSelection(searchTerm, lines)
		}
	}

	if strings.TrimSpace(searchTerm) == "" {
		return nil
	}

	sp, term := ParseSearchFilter(searchTerm)
	if strings.EqualFold(a.State.Config["SEARCH_HISTORY"], "true") {
		_ = history.AppendUnique(a.State.Paths.SearchHistoryFile, term)
	}

	encoded := url.QueryEscape(term)
	searchURL := fmt.Sprintf("https://www.youtube.com/results?search_query=%s&sp=%s", encoded, sp)
	client := &ytdlp.Client{
		PlaylistStart:    a.Page.Start,
		PlaylistEnd:      a.Page.End,
		PreferredBrowser: a.State.Config["PREFERRED_BROWSER"],
	}
	results, err := client.Fetch(searchURL)
	if err != nil {
		a.sendNotification("Failed to fetch data. Check connection or update yt-dlp.")
		return nil
	}

	if err := a.playlistExplorer(results, searchURL); err != nil {
		if errors.Is(err, ErrExit) {
			return err
		}
	}
	a.Page.Reset()
	return nil
}

func (a *App) playlistExplorer(searchResults *ytdlp.Result, searchURL string) error {
	audioOnlyMode := strings.EqualFold(a.State.Config["AUDIO_ONLY_MODE"], "true")
	autoplayMode := strings.TrimSpace(a.State.Config["AUTOPLAY_MODE"])
	if autoplayMode == "" {
		autoplayMode = "off"
	}
	downloadImages := false

	for {
		if searchResults == nil || len(searchResults.Entries) == 0 {
			break
		}
		entries := searchResults.Entries
		titles := buildTitles(entries, !downloadImages)

		enablePreview := strings.EqualFold(a.State.Config["ENABLE_PREVIEW"], "true") && strings.EqualFold(a.State.Config["PREFERRED_SELECTOR"], "fzf")
		if enablePreview && !downloadImages {
			_ = preview.DownloadPreviewImages(searchResults, a.State.Paths, "")
			downloadImages = true
		}

		options := append([]string{}, titles...)
		options = append(options, "Next", "Previous", "Back", "Exit")
		var p *ui.PreviewOptions
		if enablePreview {
			p = &ui.PreviewOptions{Mode: "video", Dispatcher: a.State.Paths.PreviewDispatcher}
		}
		selection := ui.Launcher(options, "select video", p)
		selection = normalizeSelection(selection)
		util.ClearScreen()

		switch selection {
		case "Next":
			a.Page.Next()
			client := &ytdlp.Client{PlaylistStart: a.Page.Start, PlaylistEnd: a.Page.End, PreferredBrowser: a.State.Config["PREFERRED_BROWSER"]}
			nextResults, err := client.Fetch(searchURL)
			if err != nil {
				a.sendNotification("Failed to fetch data. Check connection or update yt-dlp.")
				continue
			}
			searchResults = nextResults
			downloadImages = false
			continue
		case "Previous":
			a.Page.Previous()
			client := &ytdlp.Client{PlaylistStart: a.Page.Start, PlaylistEnd: a.Page.End, PreferredBrowser: a.State.Config["PREFERRED_BROWSER"]}
			prevResults, err := client.Fetch(searchURL)
			if err != nil {
				a.sendNotification("Failed to fetch data. Check connection or update yt-dlp.")
				continue
			}
			searchResults = prevResults
			downloadImages = false
			continue
		case "Back", "":
			return nil
		case "Exit":
			return ErrExit
		}

		selID, currentIndex, video := selectedVideo(selection, entries)
		if selID == 0 || video == nil {
			continue
		}
		cleanTitle := strings.TrimSpace(trimNumericPrefix(video.Title))

		for {
			audioState := "[ ]"
			if audioOnlyMode {
				audioState = "[x]"
			}
			autoplayLabel := "[Off]"
			switch autoplayMode {
			case "playlist":
				autoplayLabel = "[Playlist]"
			case "related":
				autoplayLabel = "[Related]"
			}
			actionSel := ui.Launcher([]string{
				"Play",
				"Toggle Audio Only " + audioState,
				"Toggle Autoplay " + autoplayLabel,
				"Download",
				"Back",
				"Exit",
			}, "Select Media Action", nil)
			util.ClearScreen()
			if actionSel == "Exit" {
				return ErrExit
			}
			if actionSel == "Back" || actionSel == "" {
				break
			}

			if strings.HasPrefix(actionSel, "Toggle Audio Only") {
				audioOnlyMode = !audioOnlyMode
				a.State.Config["AUDIO_ONLY_MODE"] = strconv.FormatBool(audioOnlyMode)
				_ = a.State.Save()
				continue
			}
			if strings.HasPrefix(actionSel, "Toggle Autoplay") {
				autoplayMode = nextAutoplayMode(autoplayMode)
				a.State.Config["AUTOPLAY_MODE"] = autoplayMode
				_ = a.State.Save()
				continue
			}

			videoURL := strings.TrimSpace(video.URL)
			if actionSel == "Play" {
				for {
					fmt.Printf("Now playing: %s\n", cleanTitle)
					cmd := player.BuildCommand(a.State.Config["PLAYER"], videoURL, cleanTitle, audioOnlyMode, a.State.Config["VIDEO_QUALITY"])
					a.playerRunning.Store(true)
					code, err := player.Run(cmd)
					a.playerRunning.Store(false)
					if player.IsInterrupted(code, err) {
						fmt.Println("\nStopping playback...")
						break
					}
					if err != nil || code != 0 {
						fmt.Println("Player exited with error. Stopping autoplay.")
						break
					}
					if autoplayMode == "off" {
						break
					}

					if autoplayMode == "playlist" {
						currentIndex++
						if currentIndex >= len(entries) {
							fmt.Println("End of current list. Fetching next page...")
							a.Page.Next()
							client := &ytdlp.Client{PlaylistStart: a.Page.Start, PlaylistEnd: a.Page.End, PreferredBrowser: a.State.Config["PREFERRED_BROWSER"]}
							nextRes, err := client.Fetch(searchURL)
							if err != nil || nextRes == nil || len(nextRes.Entries) == 0 {
								break
							}
							searchResults = nextRes
							entries = searchResults.Entries
							currentIndex = 0
							downloadImages = false
						}
						if currentIndex < len(entries) && entries[currentIndex] != nil {
							video = entries[currentIndex]
							videoURL = video.URL
							cleanTitle = trimNumericPrefix(video.Title)
						} else {
							break
						}
						continue
					}

					if autoplayMode == "related" {
						fmt.Println("Fetching related video...")
						mix, err := (&ytdlp.Client{PreferredBrowser: a.State.Config["PREFERRED_BROWSER"]}).FetchRelatedMix(video.ID)
						if err != nil || mix == nil {
							fmt.Println("Failed to fetch related videos.")
							break
						}
						found := false
						for _, entry := range mix.Entries {
							if entry != nil && entry.ID != video.ID {
								video = entry
								videoURL = video.URL
								cleanTitle = video.Title
								found = true
								break
							}
						}
						if !found {
							fmt.Println("No related videos found.")
							break
						}
					}
				}
				break
			}

			if actionSel == "Download" {
				cmd := download.BuildCommand(videoURL, a.State.Config, audioOnlyMode)
				_ = download.StartDetached(cmd)
				a.sendNotification("Started downloading " + cleanTitle)
			}
		}
	}
	return nil
}

func (a *App) sendNotification(message string) {
	_, _ = fmt.Fprintf(os.Stderr, "\033[94m[Info]\033[0m %s\n", message)
	time.Sleep(time.Duration(a.State.NotificationDurationSeconds()) * time.Second)
}

func ParseSearchFilter(term string) (sp string, cleanTerm string) {
	sp = "EgIQAQ%253D%253D"
	cleanTerm = strings.TrimSpace(term)
	m := filterPattern.FindStringSubmatch(cleanTerm)
	if len(m) == 3 {
		filter := m[1]
		cleanTerm = m[2]
		switch filter {
		case ":hour":
			sp = "EgIIAQ%253D%253D"
		case ":today":
			sp = "EgIIAg%253D%253D"
		case ":week":
			sp = "EgIIAw%253D%253D"
		case ":month":
			sp = "EgIIBA%253D%253D"
		case ":year":
			sp = "EgIIBQ%253D%253D"
		}
	}
	return sp, cleanTerm
}

func buildTitles(entries []*ytdlp.Video, withNumber bool) []string {
	out := make([]string, 0, len(entries))
	for i, entry := range entries {
		if entry == nil {
			continue
		}
		title := strings.ReplaceAll(entry.Title, "\n", " ")
		if withNumber {
			num := strconv.Itoa(i + 1)
			if len(entries) < 10 && len(num) < 2 {
				num = "0" + num
			}
			title = num + " " + title
			entry.Title = title
		}
		out = append(out, title)
	}
	return out
}

func normalizeSelection(sel string) string {
	sel = strings.TrimSpace(sel)
	if sel == "" {
		return ""
	}
	re := regexp.MustCompile(`^[^0-9]*\s\s`)
	return re.ReplaceAllString(sel, "")
}

func selectedVideo(selection string, entries []*ytdlp.Video) (int, int, *ytdlp.Video) {
	parts := strings.Fields(selection)
	if len(parts) == 0 {
		return 0, 0, nil
	}
	selID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, nil
	}
	idx := selID - 1
	if idx < 0 || idx >= len(entries) || entries[idx] == nil {
		return 0, 0, nil
	}
	return selID, idx, entries[idx]
}

func trimNumericPrefix(v string) string {
	v = strings.TrimSpace(v)
	i := 0
	for i < len(v) && v[i] >= '0' && v[i] <= '9' {
		i++
	}
	if i < len(v) && i > 0 && v[i] == ' ' {
		return strings.TrimSpace(v[i+1:])
	}
	return v
}

func nextAutoplayMode(curr string) string {
	switch curr {
	case "off":
		return "playlist"
	case "playlist":
		return "related"
	default:
		return "off"
	}
}
