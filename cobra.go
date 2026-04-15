package selfupdate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// commandConfig holds shared configuration for cobra commands.
type commandConfig struct {
	config *Config
}

// CommandOption configures the cobra commands returned by NewInstallCommand and NewUpdateCommand.
type CommandOption func(*commandConfig)

// WithConfig sets a custom Config for the underlying Updater.
func WithConfig(cfg Config) CommandOption {
	return func(c *commandConfig) {
		c.config = &cfg
	}
}

func applyOptions(opts []CommandOption) commandConfig {
	var cfg commandConfig
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

func newUpdaterFromConfig(cfg commandConfig) (*Updater, error) {
	if cfg.config != nil {
		return NewUpdater(*cfg.config)
	}
	return NewUpdater(Config{})
}

// NewInstallCommand returns a *cobra.Command that downloads a release from the
// repository and installs it to a given path.
//
// Usage: <program> install [path]
//
// If path is omitted, the binary is installed to $HOME/.local/bin/<repo>
// (the XDG user-local convention; writable without sudo).
// Use --version to install a specific version instead of the latest.
func NewInstallCommand(repository Repository, opts ...CommandOption) *cobra.Command {
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

// NewUpdateCommand returns a *cobra.Command that updates the running binary
// in-place to the latest (or a specific) version.
//
// Usage: <program> update
//
// Use --version to update to a specific version instead of the latest.
func NewUpdateCommand(repository Repository, currentVersion string, opts ...CommandOption) *cobra.Command {
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

			rel, err := up.UpdateSelf(ctx, currentVersion, repository)
			if err != nil {
				return err
			}
			if rel.Version.Version == currentVersion {
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
