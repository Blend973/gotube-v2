package download

import "testing"

func TestBuildCommandAudio(t *testing.T) {
	cfg := map[string]string{
		"VIDEO_QUALITY":      "720",
		"DOWNLOAD_DIRECTORY": "/tmp/videos",
	}
	cmd := BuildCommand("http://x", cfg, true)
	if len(cmd) < 6 {
		t.Fatalf("unexpected command: %#v", cmd)
	}
	if cmd[0] != "yt-dlp" || cmd[3] == "" {
		t.Fatalf("unexpected command start: %#v", cmd)
	}
}

func TestBuildCommandVideo(t *testing.T) {
	cfg := map[string]string{
		"VIDEO_QUALITY":      "480",
		"DOWNLOAD_DIRECTORY": "/tmp/videos",
	}
	cmd := BuildCommand("http://x", cfg, false)
	found := false
	for _, a := range cmd {
		if a == "-f" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected -f in command: %#v", cmd)
	}
}
