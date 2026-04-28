package selfupdate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// RegisterCommands registers the version, update, and install commands on the
// root command and sets the --version flag. This is the only supported way to
// integrate selfupdate into your CLI app -- call once and everything is wired up.
//
// The current version is resolved from [WithVersion] if supplied, otherwise it
// is auto-detected via [CurrentVersion] (which honours ldflags injection into
// the package-level [EmbeddedVersion] variable, the main module version from
// build info, or a VCS revision fallback).
//
// Usage in main:
//
//	selfupdate.RegisterCommands(rootCmd, selfupdate.ParseSlug("owner/repo"))
//
// Or with an explicit version override:
//
//	selfupdate.RegisterCommands(rootCmd, repo, selfupdate.WithVersion("1.0.0"))
func RegisterCommands(rootCmd *cobra.Command, repository Repository, opts ...CommandOption) {
	cfg := applyOptions(opts)
	rootCmd.Version = cfg.currentVersion
	rootCmd.AddCommand(newVersionCommand(repository, opts...))
	rootCmd.AddCommand(newUpdateCommand(repository, opts...))
	rootCmd.AddCommand(newInstallCommand(repository, opts...))
}

// commandConfig holds shared configuration for cobra commands.
type commandConfig struct {
	config         *Config
	currentVersion string
}

// CommandOption configures the cobra commands registered by RegisterCommands.
type CommandOption func(*commandConfig)

// WithConfig sets a custom Config for the underlying Updater.
func WithConfig(cfg Config) CommandOption {
	return func(c *commandConfig) {
		c.config = &cfg
	}
}

// WithVersion overrides the current binary version reported by the version
// command and used by the update command's "is there a newer release?" check.
// When this option is omitted, the version is auto-detected via [CurrentVersion].
func WithVersion(v string) CommandOption {
	return func(c *commandConfig) {
		c.currentVersion = v
	}
}

func applyOptions(opts []CommandOption) commandConfig {
	var cfg commandConfig
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.currentVersion == "" {
		cfg.currentVersion = CurrentVersion()
	}
	return cfg
}

func newUpdaterFromConfig(cfg commandConfig) (*Updater, error) {
	if cfg.config != nil {
		return NewUpdater(*cfg.config)
	}
	return NewUpdater(Config{})
}

func newInstallCommand(repository Repository, opts ...CommandOption) *cobra.Command {
	var version string

	cmd := &cobra.Command{
		Use:   "install [path]",
		Short: "Install the binary from a GitHub release",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := applyOptions(opts)
			up, err := newUpdaterFromConfig(cfg)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			var rel *Release
			if version != "" {
				rel, err = detectVersion(ctx, up, repository, version)
				if err != nil {
					return err
				}
			} else {
				r, found, err := up.DetectLatest(ctx, repository)
				if err != nil {
					return err
				}
				if !found {
					return fmt.Errorf("no release found")
				}
				rel = r
			}

			cmdPath, err := installPath(repository, args)
			if err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(cmdPath), 0o755); err != nil {
				return fmt.Errorf("create install directory: %w", err)
			}

			if err := up.UpdateTo(ctx, rel, cmdPath); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Installed %s to %s\n", rel.Version.Version, cmdPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", "", "install a specific version instead of latest")
	return cmd
}

func newUpdateCommand(repository Repository, opts ...CommandOption) *cobra.Command {
	var version string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the binary to the latest version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := applyOptions(opts)
			up, err := newUpdaterFromConfig(cfg)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			if version != "" {
				return updateToVersion(ctx, cmd, up, repository, version)
			}

			rel, err := up.UpdateSelf(ctx, cfg.currentVersion, repository)
			if err != nil {
				return err
			}
			if rel.Version.Version == cfg.currentVersion {
				fmt.Fprintln(cmd.OutOrStdout(), "Already up-to-date.")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Updated to %s\n", rel.Version.Version)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", "", "update to a specific version instead of latest")
	return cmd
}

func newVersionCommand(repository Repository, opts ...CommandOption) *cobra.Command {
	var bare bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := applyOptions(opts)
			out := cmd.OutOrStdout()
			if bare {
				fmt.Fprintln(out, cfg.currentVersion)
				return nil
			}

			up, err := newUpdaterFromConfig(cfg)
			if err != nil {
				return err
			}

			latest, found, err := up.DetectLatest(cmd.Context(), repository)
			if err != nil || !found {
				fmt.Fprintf(out, "version: %s\n", cfg.currentVersion)
				return nil
			}

			age := humanizeAge(time.Since(latest.PublishedAt))
			if latest.Version.Version == cfg.currentVersion {
				fmt.Fprintf(out, "version: %s (latest, released %s)\n", cfg.currentVersion, age)
			} else {
				fmt.Fprintf(out, "version: %s\n", cfg.currentVersion)
				fmt.Fprintf(out, "latest:  %s (released %s)\n", latest.Version.Version, age)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&bare, "bare", false, "print only the version string")
	return cmd
}

// humanizeAge converts a duration into a human-readable relative string such
// as "3 days ago", "2 weeks ago", or "1 month ago".
func humanizeAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	switch {
	case days < 1:
		return "today"
	case days == 1:
		return "1 day ago"
	case days < 14:
		return fmt.Sprintf("%d days ago", days)
	case days < 30:
		return fmt.Sprintf("%d weeks ago", days/7)
	default:
		months := days / 30
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}

func updateToVersion(ctx context.Context, cmd *cobra.Command, up *Updater, repository Repository, version string) error {
	rel, err := detectVersion(ctx, up, repository, version)
	if err != nil {
		return err
	}

	cmdPath, err := getExecutablePath()
	if err != nil {
		return err
	}

	if err := up.UpdateTo(ctx, rel, cmdPath); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Updated to %s\n", rel.Version.Version)
	return nil
}

// detectVersion tries to find a release matching the given version string.
// It tries the version as-is first, then with a "v" prefix, so callers can
// pass either "1.0.0" or "v1.0.0".
func detectVersion(ctx context.Context, up *Updater, repository Repository, version string) (*Release, error) {
	rel, found, err := up.DetectVersion(ctx, repository, version)
	if err != nil {
		return nil, err
	}
	if found {
		return rel, nil
	}
	// Retry with "v" prefix if the user passed a bare version like "1.0.0".
	if !strings.HasPrefix(version, "v") {
		rel, found, err = up.DetectVersion(ctx, repository, "v"+version)
		if err != nil {
			return nil, err
		}
		if found {
			return rel, nil
		}
	}
	return nil, fmt.Errorf("version %s not found", version)
}

// installPath determines the destination path for the install command.
// An explicit args[0] wins; otherwise default to $HOME/.local/bin/<repo>.
func installPath(repository Repository, args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	_, repo, err := repository.GetSlug()
	if err != nil {
		return "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".local", "bin", repo), nil
}
