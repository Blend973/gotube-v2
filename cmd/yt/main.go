package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"yt/internal/app"
	"yt/internal/config"
	"yt/internal/deps"
	"yt/internal/util"
)

const cliVersion = "0.8.1"

func main() {
	cliName := os.Getenv("YT_X_APP_NAME")
	if cliName == "" {
		cliName = "gotube"
	}

	searchShort := flag.String("S", "", "search for a video")
	searchLong := flag.String("search", "", "search for a video")
	editShort := flag.Bool("e", false, "edit config file")
	editLong := flag.Bool("edit-config", false, "edit config file")
	versionShort := flag.Bool("v", false, "show version")
	versionLong := flag.Bool("version", false, "show version")
	flag.Parse()

	if *versionShort || *versionLong {
		fmt.Printf("%s v%s\n", cliName, cliVersion)
		return
	}

	if err := deps.CheckDependencies(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Please install them via your package manager.")
		os.Exit(1)
	}

	state, err := config.Load(cliName, cliVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if *editShort || *editLong {
		editor := state.Config["EDITOR"]
		if editor == "" {
			editor = "nano"
		}
		cmd := exec.Command(editor, state.Paths.ConfigFile)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
		return
	}

	a, err := app.New(state)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing app: %v\n", err)
		os.Exit(1)
	}

	searchTerm := *searchShort
	if searchTerm == "" {
		searchTerm = *searchLong
	}
	initialAction := ""
	if searchTerm != "" {
		initialAction = "Search"
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigCh {
			if sig == syscall.SIGINT && a.IsPlayerRunning() {
				continue
			}
			util.ClearScreen()
			os.Exit(0)
		}
	}()

	if err := a.Run(initialAction, searchTerm); err != nil {
		if err == app.ErrExit {
			util.ClearScreen()
			return
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
