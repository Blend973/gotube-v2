package deps

import (
	"fmt"
	"yt/internal/util"
)

func CheckDependencies() error {
	required := []string{"yt-dlp", "fzf", "curl"}
	missing := make([]string, 0)
	for _, name := range required {
		if !util.CommandExists(name) {
			missing = append(missing, name)
		}
	}

	hasMPV := util.CommandExists("mpv")
	hasVLC := util.CommandExists("vlc")
	if !hasMPV && !hasVLC {
		missing = append(missing, "mpv OR vlc")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing dependencies: %v", missing)
	}
	return nil
}
