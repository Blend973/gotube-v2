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

fzf_image_clear() {
  if [ "$IMAGE_RENDERER" = "ueberzugpp" ] && [ -n "$UEBERZUGPP_FIFO" ] && [ -p "$UEBERZUGPP_FIFO" ]; then
    if [ -z "$UEBERZUGPP_FD" ]; then
      exec 3> "$UEBERZUGPP_FIFO"
      UEBERZUGPP_FD=3
      export UEBERZUGPP_FD
    fi
    printf '{"action":"remove","identifier":"ytbpreview"}\n' >&${UEBERZUGPP_FD} 2>/dev/null
  elif [ "$IMAGE_RENDERER" = "icat" ] || [ -n "$KITTY_WINDOW_ID" ]; then
    if command -v kitten >/dev/null 2>&1; then
      kitten icat --clear --stdin=no >/dev/null 2>&1
    elif command -v icat >/dev/null 2>&1; then
      icat --clear --stdin=no >/dev/null 2>&1
    elif command -v kitty >/dev/null 2>&1; then
      kitty icat --clear --stdin=no >/dev/null 2>&1
    fi
  fi
}

fzf_preview() {
  file=$1
  dim=${FZF_PREVIEW_COLUMNS}x${FZF_PREVIEW_LINES}
  if [ "$dim" = x ]; then dim=$(stty size </dev/tty | awk '{print $2 "x" $1}'); fi

  if ! [ "$IMAGE_RENDERER" = "icat" ] && [ -z "$KITTY_WINDOW_ID" ]; then
     dim=${FZF_PREVIEW_COLUMNS}x$((FZF_PREVIEW_LINES - 1))
  fi

  ueberzugpp_cleanup() {
    if [ -n "$UEBERZUGPP_FD" ]; then
      printf '{"action":"remove","identifier":"ytbpreview"}\n' >&${UEBERZUGPP_FD} 2>/dev/null
      eval "exec ${UEBERZUGPP_FD}>&-" 2>/dev/null
    fi
    if [ "$UEBERZUGPP_MANAGED" = "1" ]; then
      unset UEBERZUGPP_FD
      return 0
    fi
    if [ -n "$UEBERZUGPP_PID" ]; then
      kill "$UEBERZUGPP_PID" 2>/dev/null
    fi
    if [ -n "$UEBERZUGPP_FIFO" ]; then
      rm -f "$UEBERZUGPP_FIFO" 2>/dev/null
    fi
    unset UEBERZUGPP_FIFO UEBERZUGPP_PID UEBERZUGPP_FD
  }

  ueberzugpp_init() {
    if [ -n "$UEBERZUGPP_FIFO" ] && [ -p "$UEBERZUGPP_FIFO" ] && [ -n "$UEBERZUGPP_PID" ]; then
      if kill -0 "$UEBERZUGPP_PID" 2>/dev/null; then
        if [ -z "$UEBERZUGPP_FD" ]; then
          exec 3> "$UEBERZUGPP_FIFO"
          UEBERZUGPP_FD=3
          export UEBERZUGPP_FD
        fi
        return 0
      fi
    fi

    ueberzugpp_cleanup >/dev/null 2>&1 || true

    local tmpdir="${TMPDIR:-/tmp}"
    local reqdir="${XDG_RUNTIME_DIR:-$tmpdir}"
    local runtime_dir=""

    if [ -d "$reqdir" ] && [ -w "$reqdir" ]; then
      runtime_dir="$reqdir"
    else
      runtime_dir="$tmpdir"
    fi

    UEBERZUGPP_FIFO="$(mktemp -u "$runtime_dir/ytb-ueberzugpp-XXXXXX")"
    mkfifo "$UEBERZUGPP_FIFO"
    ueberzugpp layer --silent < "$UEBERZUGPP_FIFO" >/dev/null 2>&1 &
    UEBERZUGPP_PID=$!
    exec 3> "$UEBERZUGPP_FIFO"
    UEBERZUGPP_FD=3
    export UEBERZUGPP_FIFO UEBERZUGPP_PID UEBERZUGPP_FD
    if [ -z "$UEBERZUGPP_TRAP_SET" ]; then
      trap 'ueberzugpp_cleanup' EXIT HUP INT QUIT TERM
      UEBERZUGPP_TRAP_SET=1
      export UEBERZUGPP_TRAP_SET
    fi
  }

  ueberzugpp_preview() {
    local img="$1"
    local debug_log="${TMPDIR:-/tmp}/yt-browser-ueberzugpp-debug.log"
    if ! [ -f "$img" ]; then
      echo "Image: $(basename "$img")"
      return 0
    fi

    ueberzugpp_init || return 0

    local preview_cols="${FZF_PREVIEW_COLUMNS}"
    local preview_lines="${FZF_PREVIEW_LINES}"
    local preview_left="${FZF_PREVIEW_LEFT:-0}"
    local preview_top="${FZF_PREVIEW_TOP:-0}"

    if [ -z "$preview_cols" ] || [ "$preview_cols" -le 0 ]; then
      preview_cols=$(tput cols)
    fi
    if [ -z "$preview_lines" ] || [ "$preview_lines" -le 0 ]; then
      preview_lines=$(tput lines)
    fi

    local x=$((preview_left + 1))
    local y=$((preview_top + 1))
    local max_width=$((preview_cols - 2))
    local text_reserved=8
    local max_height=$((preview_lines / 2))

    if [ $max_height -gt $((preview_lines - text_reserved)) ]; then
      max_height=$((preview_lines - text_reserved))
    fi
    if [ $max_height -lt 4 ]; then
      max_height=4
    fi
    if [ $max_height -gt $((preview_lines - 2)) ]; then
      max_height=$((preview_lines - 2))
    fi
    if [ $max_width -lt 1 ]; then
      max_width=1
    fi
    if [ $max_height -lt 1 ]; then
      max_height=1
    fi

    if [ -n "$YT_X_DEBUG" ]; then
      (
        echo "time=$(date -Iseconds)"
        echo "img=$img"
        echo "cols=$preview_cols lines=$preview_lines left=$preview_left top=$preview_top"
        echo "x=$x y=$y max_w=$max_width max_h=$max_height"
        echo "term=$TERM tty=$(tty 2>/dev/null)"
        echo "fifo=$UEBERZUGPP_FIFO pid=$UEBERZUGPP_PID"
      ) >> "$debug_log"
    fi

    printf '{"action":"add","identifier":"ytbpreview","x":%s,"y":%s,"max_width":%s,"max_height":%s,"path":"%s"}\n' \
      "$x" "$y" "$max_width" "$max_height" "$img" >&${UEBERZUGPP_FD} 2>/dev/null

    local i=0
    while [ $i -lt $max_height ]; do
      echo
      i=$((i + 1))
    done
  }

  if [ "$IMAGE_RENDERER" = "ueberzugpp" ] && command -v ueberzugpp >/dev/null 2>&1; then
    ueberzugpp_preview "$file"
  elif [ "$IMAGE_RENDERER" = "icat" ] || [ -n "$KITTY_WINDOW_ID" ]; then
    if command -v kitten >/dev/null 2>&1; then
      kitten icat --clear --transfer-mode=memory --unicode-placeholder --stdin=no --place="$dim@0x0" "$file" | sed "\$d" | sed "$(printf "\$s/\$/\033[m/")"
    elif command -v icat >/dev/null 2>&1; then
      icat --clear --transfer-mode=memory --unicode-placeholder --stdin=no --place="$dim@0x0" "$file" | sed "\$d" | sed "$(printf "\$s/\$/\033[m/")"
    else
      kitty icat --clear --transfer-mode=memory --unicode-placeholder --stdin=no --place="$dim@0x0" "$file" | sed "\$d" | sed "$(printf "\$s/\$/\033[m/")"
    fi
  elif command -v chafa >/dev/null 2>&1; then
    chafa -f kitty -s "$dim" "$file"; echo
  elif command -v imgcat >/dev/null; then
    imgcat -W "${dim%%x*}" -H "${dim##*x}" "$file"
  else
    echo "No image renderer found"
  fi
}
export -f generate_sha256
export -f fzf_image_clear
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
    fzf_image_clear 2>/dev/null
    echo "Loading Preview Information..."
    echo "Title: $clean_title"
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
		previewImageHash := ""
		if strings.TrimSpace(thumbURL) != "" {
			previewImageHash = util.SHA256(thumbURL)
		}
		viewCount := humanViewCount(video.ViewCount)
		liveStatus := humanLiveStatus(video.LiveStatus)
		duration := humanDuration(video.Duration)
		uploaded := humanTimestamp(currentTime, video.Timestamp)
		desc := strings.ReplaceAll(strings.ReplaceAll(video.Description, "\n", " "), "\r", " ")
		if strings.TrimSpace(desc) == "" {
			desc = "null"
		}

		content := fmt.Sprintf(`
if [ -n %q ] && [ -f %q/%s.jpg ]; then
  fzf_preview %q/%s.jpg 2>/dev/null;
elif [ -n %q ]; then
  fzf_image_clear 2>/dev/null;
  echo "Image loading...";
else
  fzf_image_clear 2>/dev/null;
  echo "No preview image";
fi
ll=1
while [ $ll -le $FZF_PREVIEW_COLUMNS ];do echo -n -e "─" ;(( ll++ ));done;
echo
printf "Title: %%s\n" %s
ll=1
while [ $ll -le $FZF_PREVIEW_COLUMNS ];do echo -n -e "─" ;(( ll++ ));done;
printf "Channel: %%s\n" %s
echo "Duration: %s"
echo "Views:    %s"
echo "Live:     %s"
echo "Uploaded: %s"
ll=1
while [ $ll -le $FZF_PREVIEW_COLUMNS ];do echo -n -e "─" ;(( ll++ ));done;
echo
if [ %s != "null" ]; then printf "%%s" %s; fi
`, previewImageHash, paths.PreviewImagesCacheDir, previewImageHash, paths.PreviewImagesCacheDir, previewImageHash,
			previewImageHash, shellQuote(cleanTitle), shellQuote(video.Channel), duration, viewCount, liveStatus, uploaded,
			shellQuote(desc), shellQuote(desc))

		if err := os.WriteFile(filepath.Join(paths.PreviewScriptsDir, filenameHash+".txt"), []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func DownloadPreviewImages(data *ytdlp.Result, paths config.Paths) error {
	if data == nil {
		return nil
	}
	if err := GenerateTextPreview(data, paths, time.Now().Unix()); err != nil {
		return err
	}

	previewsFile := filepath.Join(paths.PreviewImagesCacheDir, "previews.txt")
	_ = os.Remove(previewsFile)

	entries := make([][2]string, 0, len(data.Entries))
	for _, video := range data.Entries {
		if video == nil || len(video.Thumbnails) == 0 {
			continue
		}
		url := video.Thumbnails[len(video.Thumbnails)-1].URL
		if strings.TrimSpace(url) == "" {
			continue
		}
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
		if _, err := fmt.Fprintf(f, "url = \"%s\"\n", e[0]); err != nil {
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
