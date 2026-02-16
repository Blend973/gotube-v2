package download

import (
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func BuildCommand(videoURL string, config map[string]string, audioOnly bool) []string {
	folder := "videos"
	args := []string{}
	if audioOnly {
		folder = "audio"
		args = append(args, "-x", "-f", "bestaudio", "--audio-format", "mp3")
	} else {
		q := strings.TrimSpace(config["VIDEO_QUALITY"])
		if isDigits(q) {
			args = append(args, "-f", "bestvideo[height<="+q+"]+bestaudio/best[height<="+q+"]/best")
		}
	}

	outTmpl := filepath.Join(config["DOWNLOAD_DIRECTORY"], folder, "individual", "%(channel)s", "%(title)s.%(ext)s")
	cmd := []string{"yt-dlp", videoURL, "--output", outTmpl}
	cmd = append(cmd, args...)
	if pb := strings.TrimSpace(config["PREFERRED_BROWSER"]); pb != "" {
		cmd = append(cmd, strings.Fields(pb)...)
	}
	return cmd
}

func StartDetached(cmdArgs []string) error {
	if len(cmdArgs) == 0 {
		return nil
	}
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func isDigits(s string) bool {
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
