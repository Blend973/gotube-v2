package history

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Read(file string) ([]string, error) {
	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line != "" {
			out = append(out, line)
		}
	}
	return out, s.Err()
}

func LastN(lines []string, n int) []string {
	if n <= 0 || len(lines) == 0 {
		return nil
	}
	if len(lines) <= n {
		copyLines := make([]string, len(lines))
		copy(copyLines, lines)
		return copyLines
	}
	copyLines := make([]string, n)
	copy(copyLines, lines[len(lines)-n:])
	return copyLines
}

func FormatLatest(lines []string, n int) string {
	latest := LastN(lines, n)
	if len(latest) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Search history:\n")
	for i := len(latest) - 1; i >= 0; i-- {
		index := len(latest) - i
		b.WriteString(fmt.Sprintf("%d. %s\n", index, latest[i]))
	}
	b.WriteString("(Enter !<n> to select from history. Example: !1)\n")
	return b.String()
}

func ResolveBangSelection(input string, lines []string) string {
	if !strings.HasPrefix(input, "!") || len(input) < 2 {
		return input
	}
	var idx int
	_, err := fmt.Sscanf(input, "!%d", &idx)
	if err != nil || idx <= 0 || idx > 10 || len(lines) == 0 {
		return input
	}
	if idx > len(lines) {
		return input
	}
	return lines[len(lines)-idx]
}

func AppendUnique(file string, searchTerm string) error {
	searchTerm = strings.TrimSpace(searchTerm)
	if searchTerm == "" {
		return nil
	}
	lines, err := Read(file)
	if err != nil {
		return err
	}
	filtered := make([]string, 0, len(lines)+1)
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && strings.TrimSpace(line) != searchTerm {
			filtered = append(filtered, line)
		}
	}
	filtered = append(filtered, searchTerm)

	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, line := range filtered {
		if _, err := fmt.Fprintln(f, line); err != nil {
			return err
		}
	}
	return nil
}
