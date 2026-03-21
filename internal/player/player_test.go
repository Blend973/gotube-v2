package player

import (
	"errors"
	"os"
	"testing"
)

func TestBuildCommandMPVQuality(t *testing.T) {
	cmd := BuildCommand(PlayerOpts{
		Player:        "mpv",
		VideoURL:      "http://x",
		CleanTitle:    "title",
		AudioOnly:     false,
		VideoQuality:  "720",
		BufferSecs:    "120",
		NetTimeout:    "10",
		StreamBufSize: "64",
		HWDecoding:    "no",
	})
	if len(cmd) < 3 {
		t.Fatalf("unexpected command: %#v", cmd)
	}
	if cmd[2] != "--ytdl-format=bestvideo[height<=720]+bestaudio/best[height<=720]/best" {
		t.Fatalf("unexpected format arg: %q", cmd[2])
	}
}

func TestBuildCommandAudioOnly(t *testing.T) {
	cmd := BuildCommand(PlayerOpts{
		Player:        "mpv",
		VideoURL:      "http://x",
		CleanTitle:    "title",
		AudioOnly:     true,
		VideoQuality:  "720",
		BufferSecs:    "120",
		NetTimeout:    "10",
		StreamBufSize: "64",
		HWDecoding:    "no",
	})
	hasNoVideo := false
	hasForceWindow := false
	for _, arg := range cmd {
		if arg == "--no-video" {
			hasNoVideo = true
		}
		if arg == "--force-window=no" {
			hasForceWindow = true
		}
	}
	if !hasNoVideo || !hasForceWindow {
		t.Fatalf("unexpected command: %#v", cmd)
	}
}

func TestBuildCommandCachingFlags(t *testing.T) {
	cmd := BuildCommand(PlayerOpts{
		Player:        "mpv",
		VideoURL:      "http://x",
		CleanTitle:    "title",
		BufferSecs:    "90",
		NetTimeout:    "15",
		StreamBufSize: "32",
		HWDecoding:    "no",
	})
	expected := map[string]bool{
		"--cache=yes":                  false,
		"--cache-secs=90":              false,
		"--demuxer-readahead-secs=60":  false,
		"--demuxer-seekable-cache=yes": false,
		"--stream-buffer-size=32MiB":   false,
		"--network-timeout=15":         false,
		"--audio-buffer=2":             false,
	}
	for _, arg := range cmd {
		if _, ok := expected[arg]; ok {
			expected[arg] = true
		}
	}
	for flag, found := range expected {
		if !found {
			t.Errorf("missing flag: %s", flag)
		}
	}
}

func TestBuildCommandHWDecExplicit(t *testing.T) {
	cmd := BuildCommand(PlayerOpts{
		Player:     "mpv",
		VideoURL:   "http://x",
		CleanTitle: "title",
		HWDecoding: "vaapi",
	})
	hasHWDec := false
	for _, arg := range cmd {
		if arg == "--hwdec=vaapi" {
			hasHWDec = true
		}
	}
	if !hasHWDec {
		t.Errorf("expected --hwdec=vaapi in command: %#v", cmd)
	}
}

func TestBuildCommandHWDecNo(t *testing.T) {
	cmd := BuildCommand(PlayerOpts{
		Player:     "mpv",
		VideoURL:   "http://x",
		CleanTitle: "title",
		HWDecoding: "no",
	})
	hasThreads := false
	for _, arg := range cmd {
		if arg == "--vd-lavc-threads=0" {
			hasThreads = true
		}
		if arg == "--hwdec=no" {
			t.Errorf("should not pass --hwdec=no explicitly")
		}
	}
	if !hasThreads {
		t.Errorf("expected --vd-lavc-threads=0 when hwdec=no: %#v", cmd)
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
