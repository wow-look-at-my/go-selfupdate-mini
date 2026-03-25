package selfupdate

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// CobraConfig configures the cobra commands created by NewUpdateCommand and
// NewVersionCommand.
type CobraConfig struct {
	// Version is the current version of the application (e.g. "1.2.3").
	// Required.
	Version string

	// Repository identifies the GitHub repository that hosts releases.
	// Required.
	Repository Repository

	// Updater to use. If nil, DefaultUpdater() is used.
	Updater *Updater
}

func (c *CobraConfig) updater() *Updater {
	if c.Updater != nil {
		return c.Updater
	}
	return DefaultUpdater()
}

// NewUpdateCommand returns a *cobra.Command that checks for updates and, if a
// newer version is available, downloads and installs it.
//
// Usage in your root command:
//
//	root.AddCommand(selfupdate.NewUpdateCommand(&selfupdate.CobraConfig{
//	    Version:    version,
//	    Repository: selfupdate.ParseSlug("owner/repo"),
//	}))
func NewUpdateCommand(cfg *CobraConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update to the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			up := cfg.updater()
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			status, err := up.CheckUpdate(ctx, cfg.Version, cfg.Repository)
			if err != nil {
				return fmt.Errorf("checking for updates: %w", err)
			}

			if !status.UpdateAvailable {
				fmt.Fprintf(cmd.OutOrStdout(), "Already up to date (version %s).\n", cfg.Version)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updating from %s to %s...\n",
				cfg.Version, status.LatestRelease.Version.Version)

			rel, err := up.UpdateSelf(ctx, cfg.Version, cfg.Repository)
			if err != nil {
				return fmt.Errorf("applying update: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updated to version %s.\n", rel.Version.Version)
			return nil
		},
	}
}

// NewVersionCommand returns a *cobra.Command that prints the current version
// and checks whether an update is available.
//
// Usage:
//
//	root.AddCommand(selfupdate.NewVersionCommand(&selfupdate.CobraConfig{
//	    Version:    version,
//	    Repository: selfupdate.ParseSlug("owner/repo"),
//	}))
func NewVersionCommand(cfg *CobraConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and check for updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			name := cmd.Root().Name()
			if name == "" {
				name, _ = os.Executable()
			}
			fmt.Fprintf(w, "%s version %s\n", name, cfg.Version)

			up := cfg.updater()
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			status, err := up.CheckUpdate(ctx, cfg.Version, cfg.Repository)
			if err != nil {
				// Non-fatal: still print version even if update check fails.
				fmt.Fprintf(w, "Update check failed: %v\n", err)
				return nil
			}

			if status.UpdateAvailable {
				days := status.DaysOutOfDate()
				fmt.Fprintf(w, "Update available: %s", status.LatestRelease.Version.Version)
				if days > 0 {
					fmt.Fprintf(w, " (released %d day(s) ago)", days)
				}
				fmt.Fprintln(w)
				fmt.Fprintf(w, "Run '%s update' to update.\n", name)
			} else {
				fmt.Fprintln(w, "Up to date.")
			}

			return nil
		},
	}
}

// AddVersionFlag adds a --version flag to the root command that prints
// the version string and exits. This complements the version subcommand.
func AddVersionFlag(root *cobra.Command, version string) {
	root.Version = version
	root.SetVersionTemplate(fmt.Sprintf("{{.Name}} version %s\n", version))
}
