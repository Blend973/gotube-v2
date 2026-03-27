package player

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type PlayerOpts struct {
	Player     string
	VideoURL   string
	CleanTitle string
	AudioOnly  bool
}

func BuildCommand(opts PlayerOpts) []string {
	cmd := []string{opts.Player, opts.VideoURL}

	switch strings.ToLower(strings.TrimSpace(opts.Player)) {
	case "vlc":
		cmd = append(cmd, "--video-title", opts.CleanTitle)
		if opts.AudioOnly {
			cmd = append(cmd, "--no-video")
		}
	}
	return cmd
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
