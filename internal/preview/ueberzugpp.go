package preview

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"yt/internal/util"
)

type ueberzugppDaemon struct {
	cmd   *exec.Cmd
	fifo  string
	keep  *os.File
	stdin *os.File
}

var daemonMu sync.Mutex
var daemonState *ueberzugppDaemon

func EnsureUeberzugppDaemon(cliName string, cfg map[string]string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	daemonMu.Lock()
	defer daemonMu.Unlock()

	if !shouldUseUeberzugpp(cfg) {
		stopUeberzugppLocked()
		return nil
	}
	if daemonState != nil && daemonState.cmd != nil && daemonState.cmd.Process != nil {
		if processAlive(daemonState.cmd.Process) {
			return nil
		}
	}
	stopUeberzugppLocked()

	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if !dirWritable(runtimeDir) {
		runtimeDir = os.TempDir()
	}

	fifo := filepath.Join(runtimeDir, fmt.Sprintf("%s-ueberzugpp-%d.fifo", cliName, os.Getpid()))
	_ = os.Remove(fifo)
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		return err
	}

	keep, err := os.OpenFile(fifo, os.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		_ = os.Remove(fifo)
		return err
	}
	stdinFile, err := os.OpenFile(fifo, os.O_RDONLY, 0)
	if err != nil {
		_ = keep.Close()
		_ = os.Remove(fifo)
		return err
	}

	cmd := exec.Command("ueberzugpp", "layer", "--silent")
	cmd.Stdin = stdinFile
	devNull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if devNull != nil {
		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}
	if err := cmd.Start(); err != nil {
		if devNull != nil {
			_ = devNull.Close()
		}
		_ = stdinFile.Close()
		_ = keep.Close()
		_ = os.Remove(fifo)
		return err
	}
	if devNull != nil {
		_ = devNull.Close()
	}

	daemonState = &ueberzugppDaemon{cmd: cmd, fifo: fifo, keep: keep, stdin: stdinFile}
	os.Setenv("UEBERZUGPP_FIFO", fifo)
	os.Setenv("UEBERZUGPP_PID", strconv.Itoa(cmd.Process.Pid))
	os.Setenv("UEBERZUGPP_MANAGED", "1")
	return nil
}

func StopUeberzugppDaemon() {
	daemonMu.Lock()
	defer daemonMu.Unlock()
	stopUeberzugppLocked()
}

func ClearPreviewImage(cfg map[string]string) {
	if cfg == nil {
		return
	}
	if !strings.EqualFold(cfg["ENABLE_PREVIEW"], "true") {
		return
	}
	renderer := strings.ToLower(strings.TrimSpace(cfg["IMAGE_RENDERER"]))
	switch renderer {
	case "ueberzugpp":
		clearUeberzugppImage()
	case "icat":
		clearKittyImage()
	}
}

func stopUeberzugppLocked() {
	if daemonState == nil {
		return
	}
	if cmd := daemonState.cmd; cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan struct{}, 1)
		go func() {
			_ = cmd.Wait()
			done <- struct{}{}
		}()
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}
	if daemonState.stdin != nil {
		_ = daemonState.stdin.Close()
	}
	if daemonState.keep != nil {
		_ = daemonState.keep.Close()
	}
	if daemonState.fifo != "" {
		_ = os.Remove(daemonState.fifo)
	}
	os.Unsetenv("UEBERZUGPP_FIFO")
	os.Unsetenv("UEBERZUGPP_PID")
	os.Unsetenv("UEBERZUGPP_MANAGED")
	daemonState = nil
}

func shouldUseUeberzugpp(cfg map[string]string) bool {
	if cfg == nil {
		return false
	}
	if !strings.EqualFold(cfg["ENABLE_PREVIEW"], "true") {
		return false
	}
	if !strings.EqualFold(cfg["IMAGE_RENDERER"], "ueberzugpp") {
		return false
	}
	return util.CommandExists("ueberzugpp")
}

func clearUeberzugppImage() {
	fifo := os.Getenv("UEBERZUGPP_FIFO")
	if fifo == "" {
		return
	}
	info, err := os.Stat(fifo)
	if err != nil || (info.Mode()&os.ModeNamedPipe) == 0 {
		return
	}
	f, err := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return
	}
	_, _ = f.WriteString("{\"action\":\"remove\",\"identifier\":\"ytbpreview\"}\n")
	_ = f.Close()
}

func clearKittyImage() {
	if util.CommandExists("kitten") {
		_ = exec.Command("kitten", "icat", "--clear", "--stdin=no").Run()
		return
	}
	if util.CommandExists("icat") {
		_ = exec.Command("icat", "--clear", "--stdin=no").Run()
		return
	}
	if util.CommandExists("kitty") {
		_ = exec.Command("kitty", "icat", "--clear", "--stdin=no").Run()
	}
}

func processAlive(p *os.Process) bool {
	if p == nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return p.Signal(syscall.Signal(0)) == nil
}

func dirWritable(dir string) bool {
	if dir == "" {
		return false
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	f, err := os.CreateTemp(dir, "ytb-")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}
