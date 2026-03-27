package player

import (
	"errors"
	"os"
	"testing"
)

func TestBuildCommandMPVPlain(t *testing.T) {
	cmd := BuildCommand(PlayerOpts{
		Player:     "mpv",
		VideoURL:   "http://x",
		CleanTitle: "title",
		AudioOnly:  false,
	})
	if len(cmd) != 2 {
		t.Fatalf("unexpected command: %#v", cmd)
	}
	if cmd[0] != "mpv" || cmd[1] != "http://x" {
		t.Fatalf("unexpected command: %#v", cmd)
	}
}

func TestBuildCommandVLCAudioOnly(t *testing.T) {
	cmd := BuildCommand(PlayerOpts{
		Player:     "vlc",
		VideoURL:   "http://x",
		CleanTitle: "title",
		AudioOnly:  true,
	})
	if len(cmd) != 5 {
		t.Fatalf("unexpected command: %#v", cmd)
	}
	if cmd[2] != "--video-title" || cmd[3] != "title" || cmd[4] != "--no-video" {
		t.Fatalf("unexpected command: %#v", cmd)
	}
}

func TestBuildCommandVLCVideoTitle(t *testing.T) {
	cmd := BuildCommand(PlayerOpts{
		Player:     "vlc",
		VideoURL:   "http://x",
		CleanTitle: "title",
	})
	if len(cmd) != 4 {
		t.Fatalf("unexpected command: %#v", cmd)
	}
	if cmd[2] != "--video-title" || cmd[3] != "title" {
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
