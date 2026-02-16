package player

import (
	"errors"
	"os"
	"testing"
)

func TestBuildCommandMPVQuality(t *testing.T) {
	cmd := BuildCommand("mpv", "http://x", "title", false, "720")
	if len(cmd) < 3 {
		t.Fatalf("unexpected command: %#v", cmd)
	}
	if cmd[2] != "--ytdl-format=bestvideo[height<=720]+bestaudio/best[height<=720]/best" {
		t.Fatalf("unexpected format arg: %q", cmd[2])
	}
}

func TestBuildCommandAudioOnly(t *testing.T) {
	cmd := BuildCommand("mpv", "http://x", "title", true, "720")
	if len(cmd) < 4 || cmd[2] != "--no-video" || cmd[3] != "--force-window=no" {
		t.Fatalf("unexpected command: %#v", cmd)
	}
}

func TestBuildExecCommandBindsTerminalStreams(t *testing.T) {
	c, err := buildExecCommand([]string{"player-bin", "ok"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Stdin != os.Stdin {
		t.Fatalf("stdin not bound to terminal stdin")
	}
	if c.Stdout != os.Stdout {
		t.Fatalf("stdout not bound to terminal stdout")
	}
	if c.Stderr != os.Stderr {
		t.Fatalf("stderr not bound to terminal stderr")
	}
}

func TestBuildExecCommandEmpty(t *testing.T) {
	_, err := buildExecCommand(nil)
	if err == nil {
		t.Fatalf("expected error for empty command")
	}
}

func TestIsInterruptedByExitCode(t *testing.T) {
	if !IsInterrupted(130, errors.New("interrupted")) {
		t.Fatalf("expected interrupted status for code 130")
	}
}

func TestIsInterruptedWithoutError(t *testing.T) {
	if IsInterrupted(130, nil) {
		t.Fatalf("did not expect interrupted status when there is no error")
	}
}
