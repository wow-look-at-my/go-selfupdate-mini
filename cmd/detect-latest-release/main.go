package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	selfupdate "github.com/wow-look-at-my/lightweight-go-selfupdate/pkg/selfupdate"
)

func main() {
	var help, verbose bool
	var forceOS, forceArch string
	flag.BoolVar(&help, "h", false, "Show help")
	flag.BoolVar(&verbose, "v", false, "Display debugging information")
	flag.StringVar(&forceOS, "o", "", "OS name (windows, darwin, linux, etc)")
	flag.StringVar(&forceArch, "a", "", "CPU architecture (amd64, arm64, etc)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: detect-latest-release [flags] owner/repo\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if help || flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	if verbose {
		selfupdate.SetLogger(log.New(os.Stdout, "", 0))
	}

	cfg := selfupdate.Config{}
	if forceOS != "" {
		cfg.Platform.OS = forceOS
	}
	if forceArch != "" {
		cfg.Platform.Arch = forceArch
	}

	updater, err := selfupdate.NewUpdater(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	latest, found, err := updater.DetectLatest(context.Background(), selfupdate.ParseSlug(flag.Arg(0)))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !found {
		fmt.Println("No release found")
		return
	}
	fmt.Printf("Latest version: %s\n", latest.Version.Version)
	fmt.Printf("Download URL:   %s\n", latest.Asset.URL)
	fmt.Printf("Release URL:    %s\n", latest.URL)
	fmt.Printf("Release Notes:\n%s\n", latest.ReleaseNotes)
}
