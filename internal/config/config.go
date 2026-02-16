package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Paths struct {
	XDGConfigHome         string
	XDGCacheHome          string
	XDGVideosDir          string
	ConfigDir             string
	ConfigFile            string
	CacheDir              string
	PreviewImagesCacheDir string
	PreviewScriptsDir     string
	HelperScript          string
	PreviewDispatcher     string
	SearchHistoryFile     string
}

type State struct {
	CLIName    string
	CLIVersion string
	Platform   string
	Config     map[string]string
	Paths      Paths
}

func DetectPlatform() string {
	sys := strings.ToLower(runtime.GOOS)
	switch {
	case strings.Contains(sys, "darwin"):
		return "mac"
	case strings.Contains(sys, "windows"):
		return "windows"
	default:
		return "linux"
	}
}

func ResolvePaths(cliName string) Paths {
	home, _ := os.UserHomeDir()
	xdgConfigHome := envOr("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	xdgCacheHome := envOr("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	xdgVideosDir := envOr("XDG_VIDEOS_DIR", filepath.Join(home, "Videos"))

	configDir := filepath.Join(xdgConfigHome, cliName)
	cacheDir := filepath.Join(xdgCacheHome, cliName)

	return Paths{
		XDGConfigHome:         xdgConfigHome,
		XDGCacheHome:          xdgCacheHome,
		XDGVideosDir:          xdgVideosDir,
		ConfigDir:             configDir,
		ConfigFile:            filepath.Join(configDir, fmt.Sprintf("%s.conf", cliName)),
		CacheDir:              cacheDir,
		PreviewImagesCacheDir: filepath.Join(cacheDir, "preview_images"),
		PreviewScriptsDir:     filepath.Join(cacheDir, "preview_text"),
		HelperScript:          filepath.Join(configDir, "yt-x-helper.sh"),
		PreviewDispatcher:     filepath.Join(configDir, "yt-x-preview.sh"),
		SearchHistoryFile:     filepath.Join(cacheDir, "search_history.txt"),
	}
}

func DefaultConfig(cliName string, paths Paths) map[string]string {
	editor := envOr("EDITOR", "nano")
	return map[string]string{
		"IMAGE_RENDERER":        "",
		"EDITOR":                editor,
		"PREFERRED_SELECTOR":    "fzf",
		"VIDEO_QUALITY":         "720",
		"ENABLE_PREVIEW":        "false",
		"PLAYER":                "mpv",
		"PREFERRED_BROWSER":     "",
		"NO_OF_SEARCH_RESULTS":  "30",
		"NOTIFICATION_DURATION": "5",
		"SEARCH_HISTORY":        "true",
		"DOWNLOAD_DIRECTORY":    filepath.Join(paths.XDGVideosDir, cliName),
		"AUDIO_ONLY_MODE":       "false",
		"AUTOPLAY_MODE":         "off",
	}
}

func Load(cliName, cliVersion string) (*State, error) {
	paths := ResolvePaths(cliName)
	cfg := DefaultConfig(cliName, paths)

	for _, d := range []string{paths.ConfigDir, paths.PreviewImagesCacheDir, paths.PreviewScriptsDir, paths.CacheDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}

	st := &State{
		CLIName:    cliName,
		CLIVersion: cliVersion,
		Platform:   DetectPlatform(),
		Config:     cfg,
		Paths:      paths,
	}

	if _, err := os.Stat(paths.ConfigFile); os.IsNotExist(err) {
		if err := st.Save(); err != nil {
			return nil, err
		}
	}

	if err := st.loadFile(); err != nil {
		return nil, err
	}

	if st.Config["IMAGE_RENDERER"] == "" {
		if os.Getenv("KITTY_WINDOW_ID") != "" {
			st.Config["IMAGE_RENDERER"] = "icat"
		} else {
			st.Config["IMAGE_RENDERER"] = "chafa"
		}
	}

	if pb := strings.TrimSpace(st.Config["PREFERRED_BROWSER"]); pb != "" && !strings.Contains(pb, "--cookies-from-browser") {
		st.Config["PREFERRED_BROWSER"] = "--cookies-from-browser " + pb
	}

	st.Config["DOWNLOAD_DIRECTORY"] = expandPath(st.Config["DOWNLOAD_DIRECTORY"])
	if err := os.MkdirAll(st.Config["DOWNLOAD_DIRECTORY"], 0o755); err != nil {
		return nil, err
	}

	os.Setenv("FZF_DEFAULT_OPTS", fzfDefaultOpts())
	if v := os.Getenv("YT_X_FZF_OPTS"); strings.TrimSpace(v) != "" {
		os.Setenv("FZF_DEFAULT_OPTS", v)
	}
	os.Setenv("PLATFORM", st.Platform)
	os.Setenv("IMAGE_RENDERER", st.Config["IMAGE_RENDERER"])

	return st, nil
}

func (s *State) loadFile() error {
	f, err := os.Open(s.Paths.ConfigFile)
	if err != nil {
		return err
	}
	defer f.Close()

	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		s.Config[k] = v
	}
	return scan.Err()
}

func (s *State) Save() error {
	f, err := os.Create(s.Paths.ConfigFile)
	if err != nil {
		return err
	}
	defer f.Close()
	for k, v := range s.Config {
		if _, err := fmt.Fprintf(f, "%s: %s\n", k, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) NoOfResults() int {
	n, err := strconv.Atoi(strings.TrimSpace(s.Config["NO_OF_SEARCH_RESULTS"]))
	if err != nil || n <= 0 {
		return 30
	}
	return n
}

func (s *State) NotificationDurationSeconds() int {
	n, err := strconv.Atoi(strings.TrimSpace(s.Config["NOTIFICATION_DURATION"]))
	if err != nil || n < 0 {
		return 5
	}
	return n
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func expandPath(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "~/") || v == "~" {
		home, _ := os.UserHomeDir()
		if v == "~" {
			v = home
		} else {
			v = filepath.Join(home, strings.TrimPrefix(v, "~/"))
		}
	}
	return os.ExpandEnv(v)
}

func fzfDefaultOpts() string {
	return `
    --color=fg:#d0d0d0,fg+:#d0d0d0,bg:#121212,bg+:#262626
    --color=hl:#5f87af,hl+:#5fd7ff,info:#afaf87,marker:#87ff00
    --color=prompt:#d7005f,spinner:#af5fff,pointer:#af5fff,header:#87afaf
    --color=border:#262626,label:#aeaeae,query:#d9d9d9
    --border="rounded" --border-label="" --preview-window="border-rounded" --prompt="> "
    --marker=">" --pointer="◆" --separator="─" --scrollbar="│"
    `
}
