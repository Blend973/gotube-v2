package player

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"yt/internal/util"
)

type PlayerOpts struct {
	Player        string
	VideoURL      string
	CleanTitle    string
	AudioOnly     bool
	VideoQuality  string
	BufferSecs    string
	NetTimeout    string
	StreamBufSize string
	HWDecoding    string
}

func BuildCommand(opts PlayerOpts) []string {
	cmd := []string{opts.Player, opts.VideoURL}

	switch strings.ToLower(strings.TrimSpace(opts.Player)) {
	case "mpv":
		if opts.AudioOnly {
			cmd = append(cmd, "--no-video", "--force-window=no")
		} else if isDigits(opts.VideoQuality) {
			q := strings.TrimSpace(opts.VideoQuality)
			cmd = append(cmd, fmt.Sprintf("--ytdl-format=bestvideo[height<=%s]+bestaudio/best[height<=%s]/best", q, q))
		}

		cmd = append(cmd, "--cache=yes")
		if s := strings.TrimSpace(opts.BufferSecs); isDigits(s) {
			cmd = append(cmd, fmt.Sprintf("--cache-secs=%s", s))
		}
		cmd = append(cmd, "--demuxer-readahead-secs=60", "--demuxer-seekable-cache=yes")
		if s := strings.TrimSpace(opts.StreamBufSize); isDigits(s) {
			cmd = append(cmd, fmt.Sprintf("--stream-buffer-size=%sMiB", s))
		}
		if s := strings.TrimSpace(opts.NetTimeout); isDigits(s) {
			cmd = append(cmd, fmt.Sprintf("--network-timeout=%s", s))
		}
		cmd = append(cmd, "--audio-buffer=2")

		hwdec := strings.ToLower(strings.TrimSpace(opts.HWDecoding))
		if hwdec == "" || hwdec == "auto" {
			hwdec = detectHWDec()
		}
		if hwdec != "no" && hwdec != "" {
			cmd = append(cmd, fmt.Sprintf("--hwdec=%s", hwdec))
		} else {
			cmd = append(cmd, "--vd-lavc-threads=0")
		}

	case "vlc":
		cmd = append(cmd, "--video-title", opts.CleanTitle)
		if opts.AudioOnly {
			cmd = append(cmd, "--no-video")
		}
	}
	return cmd
}

func detectHWDec() string {
	if util.CommandExists("vainfo") {
		c := exec.Command("vainfo")
		c.Stdout = nil
		c.Stderr = nil
		if err := c.Run(); err == nil {
			return "vaapi"
		}
	}
	if _, err := os.Stat("/dev/dri/renderD128"); err == nil {
		return "nvdec"
	}
	if util.CommandExists("vulkaninfo") {
		return "vulkan"
	}
	return "no"
}

func Run(cmd []string) (int, error) {
	c, err := buildExecCommand(cmd)
	if err != nil {
		return 1, err
	}
	err = c.Run()
	if c.ProcessState != nil {
		return c.ProcessState.ExitCode(), err
	}
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func buildExecCommand(cmd []string) (*exec.Cmd, error) {
	if len(cmd) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c, nil
}

func IsInterrupted(code int, err error) bool {
	if err == nil {
		return false
	}
	if code == 130 {
		return true
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	return ok && status.Signaled() && status.Signal() == syscall.SIGINT
}

func isDigits(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
