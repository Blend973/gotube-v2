package preview

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"yt/internal/config"
	"yt/internal/util"
	"yt/internal/ytdlp"
)

func CreateBashHelpers(paths config.Paths, imageRenderer string) error {
	helperTemplate := `#!/usr/bin/env bash
export CLI_PREVIEW_IMAGES_CACHE_DIR="__IMG_CACHE__"
export CLI_PREVIEW_SCRIPTS_DIR="__SCRIPT_CACHE__"
export IMAGE_RENDERER="__IMAGE_RENDERER__"

generate_sha256() {
  local input
  if [ -n "$1" ]; then input="$1"; else input=$(cat); fi
  if command -v sha256sum &>/dev/null; then echo -n "$input" | sha256sum | awk '{print $1}'
  elif command -v shasum &>/dev/null; then echo -n "$input" | shasum -a 256 | awk '{print $1}'
  else echo -n "$input" | base64 | tr '/+' '_-' | tr -d '\n'; fi
}

fzf_preview() {
  file=$1
  dim=${FZF_PREVIEW_COLUMNS}x${FZF_PREVIEW_LINES}
  if [ "$dim" = x ]; then dim=$(stty size </dev/tty | awk '{print $2 "x" $1}'); fi

  if ! [ "$IMAGE_RENDERER" = "icat" ] && [ -z "$KITTY_WINDOW_ID" ]; then
     dim=${FZF_PREVIEW_COLUMNS}x$((FZF_PREVIEW_LINES - 1))
  fi

  if [ "$IMAGE_RENDERER" = "icat" ] || [ -n "$KITTY_WINDOW_ID" ]; then
    if command -v kitten >/dev/null 2>&1; then
      kitten icat --clear --transfer-mode=memory --unicode-placeholder --stdin=no --place="$dim@0x0" "$file" | sed "\$d" | sed "$(printf "\$s/\$/\033[m/")"
    elif command -v icat >/dev/null 2>&1; then
      icat --clear --transfer-mode=memory --unicode-placeholder --stdin=no --place="$dim@0x0" "$file" | sed "\$d" | sed "$(printf "\$s/\$/\033[m/")"
    else
      kitty icat --clear --transfer-mode=memory --unicode-placeholder --stdin=no --place="$dim@0x0" "$file" | sed "\$d" | sed "$(printf "\$s/\$/\033[m/")"
    fi
  elif command -v chafa >/dev/null 2>&1; then
    chafa -s "$dim" "$file"; echo
  elif command -v imgcat >/dev/null; then
    imgcat -W "${dim%%x*}" -H "${dim##*x}" "$file"
  else
    echo "No image renderer found"
  fi
}
export -f generate_sha256
export -f fzf_preview
`
	helperContent := strings.NewReplacer(
		"__IMG_CACHE__", paths.PreviewImagesCacheDir,
		"__SCRIPT_CACHE__", paths.PreviewScriptsDir,
		"__IMAGE_RENDERER__", imageRenderer,
	).Replace(helperTemplate)

	if err := os.WriteFile(paths.HelperScript, []byte(helperContent), 0o755); err != nil {
		return err
	}

	previewTemplate := `#!/usr/bin/env bash
source "__HELPER__"
MODE="$1"; shift; SELECTION="$*"

if [ "$MODE" = "video" ]; then
  title="$SELECTION"
  clean_title=$(echo "$title" | sed -E 's/^[0-9]+ //g')
  id=$(generate_sha256 "$clean_title")
  if [ -f "__SCRIPT_CACHE__"/${id}.txt ]; then
    . "__SCRIPT_CACHE__"/${id}.txt
  else
    echo "Loading Preview..."
  fi
fi
`
	previewContent := strings.NewReplacer(
		"__HELPER__", paths.HelperScript,
		"__SCRIPT_CACHE__", paths.PreviewScriptsDir,
	).Replace(previewTemplate)

	return os.WriteFile(paths.PreviewDispatcher, []byte(previewContent), 0o755)
}

func CleanupCache(paths config.Paths) {
	now := time.Now().Unix()
	cutoff := now - 86400
	for _, d := range []string{paths.PreviewImagesCacheDir, paths.PreviewScriptsDir} {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			full := filepath.Join(d, e.Name())
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Unix() < cutoff {
				_ = os.Remove(full)
			}
		}
	}
}

func GenerateTextPreview(data *ytdlp.Result, paths config.Paths, currentTime int64) error {
	if data == nil {
		return nil
	}
	for _, video := range data.Entries {
		if video == nil {
			continue
		}
		rawTitle := strings.ReplaceAll(video.Title, "\n", " ")
		cleanTitle := trimNumericPrefix(rawTitle)
		filenameHash := util.SHA256(cleanTitle)

		thumbURL := ""
		if len(video.Thumbnails) > 0 {
			thumbURL = video.Thumbnails[len(video.Thumbnails)-1].URL
		}
		previewImageHash := util.SHA256(thumbURL)
		viewCount := humanViewCount(video.ViewCount)
		liveStatus := humanLiveStatus(video.LiveStatus)
		duration := humanDuration(video.Duration)
		uploaded := humanTimestamp(currentTime, video.Timestamp)
		desc := strings.ReplaceAll(strings.ReplaceAll(video.Description, "\n", " "), "\r", " ")
		if strings.TrimSpace(desc) == "" {
			desc = "null"
		}

		content := fmt.Sprintf(`
if [ -f %q/%s.jpg ];then fzf_preview %q/%s.jpg 2>/dev/null;
else echo loading preview image...;
fi
ll=1
while [ $ll -le $FZF_PREVIEW_COLUMNS ];do echo -n -e "─" ;(( ll++ ));done;
echo
echo %s
ll=1
while [ $ll -le $FZF_PREVIEW_COLUMNS ];do echo -n -e "─" ;(( ll++ ));done;
echo "Channel: %s"
echo "Duration: %s"
echo "View Count: %s views"
echo "Live Status: %s"
echo "Uploaded: %s"
ll=1
while [ $ll -le $FZF_PREVIEW_COLUMNS ];do echo -n -e "─" ;(( ll++ ));done;
echo
! [ %s = "null" ] && echo -n %s;
`, paths.PreviewImagesCacheDir, previewImageHash, paths.PreviewImagesCacheDir, previewImageHash,
			shellQuote(cleanTitle), shellQuote(video.Channel), duration, viewCount, liveStatus, uploaded,
			shellQuote(desc), shellQuote(desc))

		if err := os.WriteFile(filepath.Join(paths.PreviewScriptsDir, filenameHash+".txt"), []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func DownloadPreviewImages(data *ytdlp.Result, paths config.Paths, prefix string) error {
	if data == nil {
		return nil
	}
	if err := GenerateTextPreview(data, paths, time.Now().Unix()); err != nil {
		return err
	}

	previewsFile := filepath.Join(paths.PreviewImagesCacheDir, "previews.txt")
	_ = os.Remove(previewsFile)

	entries := make([][2]string, 0)
	for _, video := range data.Entries {
		if video == nil || len(video.Thumbnails) == 0 {
			continue
		}
		url := video.Thumbnails[len(video.Thumbnails)-1].URL
		filename := util.SHA256(url)
		if _, err := os.Stat(filepath.Join(paths.PreviewImagesCacheDir, filename+".jpg")); err == nil {
			continue
		}
		entries = append(entries, [2]string{url, filename})
	}
	if len(entries) == 0 {
		return nil
	}

	f, err := os.Create(previewsFile)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if _, err := fmt.Fprintf(f, "url = \"%s%s\"\n", prefix, e[0]); err != nil {
			_ = f.Close()
			return err
		}
		if _, err := fmt.Fprintf(f, "output = \"%s/%s.jpg\"\n", paths.PreviewImagesCacheDir, e[1]); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}

	cmd := exec.Command("curl", "-s", "-K", previewsFile)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func trimNumericPrefix(v string) string {
	v = strings.TrimSpace(v)
	for i := 0; i < len(v); i++ {
		if v[i] < '0' || v[i] > '9' {
			if v[i] == ' ' {
				return strings.TrimSpace(v[i+1:])
			}
			break
		}
	}
	return v
}

func humanViewCount(v interface{}) string {
	n, ok := asInt64(v)
	if !ok {
		return "Unknown"
	}
	return formatComma(n)
}

func humanLiveStatus(status string) string {
	switch status {
	case "is_live":
		return "Online"
	case "was_live":
		return "Offline"
	default:
		return "False"
	}
}

func humanDuration(v interface{}) string {
	n, ok := asFloat64(v)
	if !ok || n <= 0 {
		return "Unknown"
	}
	if n >= 3600 {
		return fmt.Sprintf("%d hours", int(n/3600))
	}
	if n >= 60 {
		return fmt.Sprintf("%d mins", int(n/60))
	}
	return fmt.Sprintf("%d secs", int(n))
}

func humanTimestamp(now int64, v interface{}) string {
	ts, ok := asInt64(v)
	if !ok || ts <= 0 {
		return ""
	}
	diff := now - ts
	switch {
	case diff < 60:
		return "just now"
	case diff < 3600:
		return fmt.Sprintf("%d minutes ago", diff/60)
	case diff < 86400:
		return fmt.Sprintf("%d hours ago", diff/3600)
	case diff < 604800:
		return fmt.Sprintf("%d days ago", diff/86400)
	case diff < 2635200:
		return fmt.Sprintf("%d weeks ago", diff/604800)
	case diff < 31622400:
		return fmt.Sprintf("%d months ago", diff/2635200)
	default:
		return fmt.Sprintf("%d years ago", diff/31622400)
	}
}

func asInt64(v interface{}) (int64, bool) {
	switch t := v.(type) {
	case int:
		return int64(t), true
	case int64:
		return t, true
	case float64:
		return int64(t), true
	case jsonNumber:
		n, err := strconv.ParseInt(string(t), 10, 64)
		return n, err == nil
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func asFloat64(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return n, err == nil
	default:
		return 0, false
	}
}

type jsonNumber string

func formatComma(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
