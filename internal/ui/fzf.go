package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type PreviewOptions struct {
	Mode       string
	Dispatcher string
}

func Prompt(text, value, historyText string) string {
	if _, err := exec.LookPath("gum"); err == nil {
		args := []string{"input", "--header", "", "--prompt", text + ": ", "--value", value}
		cmd := exec.Command("gum", args...)
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}

	if strings.TrimSpace(historyText) != "" {
		_, _ = fmt.Fprintln(os.Stderr, historyText)
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s: ", text)
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(line)
}

func Launcher(options []string, prompt string, preview *PreviewOptions) string {
	joined := strings.Join(options, "\n") + "\n"
	args := []string{
		"--info=hidden",
		"--layout=reverse",
		"--height=100%",
		"--prompt=" + prompt + ": ",
		"--header-first",
		"--header=",
		"--exact",
		"--cycle",
		"--ansi",
	}
	withPreview := preview != nil && preview.Mode != "" && preview.Dispatcher != ""
	if withPreview {
		args = append(args,
			"--preview-window=left,35%,wrap",
			"--bind=right:accept",
			"--expect=shift-left,shift-right",
			"--tabstop=1",
			fmt.Sprintf("--preview=bash '%s' '%s' {}", preview.Dispatcher, preview.Mode),
		)
	}

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(joined)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}
	if withPreview && len(lines) >= 2 {
		return strings.TrimSpace(lines[1])
	}
	return strings.TrimSpace(lines[0])
}
