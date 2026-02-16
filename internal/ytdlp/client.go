package ytdlp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Thumbnail struct {
	URL string `json:"url"`
}

type Video struct {
	ID          string      `json:"id"`
	URL         string      `json:"url"`
	Title       string      `json:"title"`
	Channel     string      `json:"channel"`
	Description string      `json:"description"`
	Duration    interface{} `json:"duration"`
	Timestamp   interface{} `json:"timestamp"`
	ViewCount   interface{} `json:"view_count"`
	LiveStatus  string      `json:"live_status"`
	Thumbnails  []Thumbnail `json:"thumbnails"`
}

type Result struct {
	Entries []*Video `json:"entries"`
}

type Client struct {
	PlaylistStart    int
	PlaylistEnd      int
	PreferredBrowser string
}

func (c *Client) Fetch(url string, extraArgs ...string) (*Result, error) {
	args := []string{
		url,
		"-J",
		"--flat-playlist",
		"--extractor-args", "youtubetab:approximate_date",
		"--playlist-start", strconv.Itoa(c.PlaylistStart),
		"--playlist-end", strconv.Itoa(c.PlaylistEnd),
	}
	if strings.TrimSpace(c.PreferredBrowser) != "" {
		args = append(args, strings.Fields(c.PreferredBrowser)...)
	}
	args = append(args, extraArgs...)

	stdout, err := runMaybeWithGum(args...)
	if err != nil {
		return nil, err
	}
	return ParseResult(stdout)
}

func (c *Client) FetchRelatedMix(currentVideoID string) (*Result, error) {
	mixURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s&list=RD%s", currentVideoID, currentVideoID)
	args := []string{mixURL, "-J", "--flat-playlist", "--playlist-start", "1", "--playlist-end", "5"}
	if strings.TrimSpace(c.PreferredBrowser) != "" {
		args = append(args, strings.Fields(c.PreferredBrowser)...)
	}
	cmd := exec.Command("yt-dlp", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return ParseResult(string(out))
}

func ParseResult(stdout string) (*Result, error) {
	stdout = strings.TrimSpace(stdout)
	if stdout == "" {
		return nil, errors.New("empty yt-dlp output")
	}

	var r Result
	if err := json.Unmarshal([]byte(stdout), &r); err == nil {
		return &r, nil
	}
	idx := strings.Index(stdout, "{")
	if idx >= 0 {
		if err := json.Unmarshal([]byte(stdout[idx:]), &r); err == nil {
			return &r, nil
		}
	}
	return nil, errors.New("failed to parse yt-dlp json")
}

func runMaybeWithGum(ytArgs ...string) (string, error) {
	if _, err := exec.LookPath("gum"); err == nil {
		args := append([]string{"spin", "--show-output", "--", "yt-dlp"}, ytArgs...)
		cmd := exec.Command("gum", args...)
		out, err := cmd.Output()
		if err == nil {
			return string(out), nil
		}
	}
	cmd := exec.Command("yt-dlp", ytArgs...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
