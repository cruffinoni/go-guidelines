package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/cruffinoni/go-guidelines/internal/update"
)

// pathHelp returns the PATH guidance the installers print.
func pathHelp(result update.Result) string {
	binDir := filepath.Dir(result.Path)
	if result.PathUpdated {
		return fmt.Sprintf("Added %s to the user PATH.\nRestart your terminal to use gg-lint from other shells.\n", binDir)
	}
	if result.OnPath {
		return ""
	}
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("Add %s to your PATH to run gg-lint from other shells.\n", binDir)
	}
	return fmt.Sprintf("\nAdd %s to your PATH:\n  export PATH=\"%s:$PATH\"\n", binDir, binDir)
}

// newUpdateCommand builds the update command, which installs a GitHub release
// into the same directory as the install scripts.
func newUpdateCommand() *cobra.Command {
	tag := os.Getenv("GG_LINT_VERSION")
	binDir := os.Getenv("GG_LINT_BIN_DIR")
	noPathUpdate := os.Getenv("GG_LINT_NO_PATH_UPDATE") == "1"
	var quiet bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update gg-lint from a GitHub release.",
		Long: "Update gg-lint by downloading a GitHub release for the current operating system and CPU, " +
			"verifying its SHA-256 checksum, and installing it into the same directory as the install " +
			"scripts. Without --version, only the latest stable release is considered. Pass --version " +
			"to install a specific tag, including prereleases such as v1.2.3-rc.1.",
		Example: "  gg-lint update\n" +
			"  gg-lint update --version v1.2.3-rc.1\n" +
			"  gg-lint update --bin-dir \"$HOME/.local/bin\"",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			reporter := newUpdateReporter(cmd.ErrOrStderr(), quiet)
			result, err := update.Run(cmd.Context(), update.Options{
				CurrentVersion: version,
				Version:        tag,
				BinDir:         binDir,
				NoPathUpdate:   noPathUpdate,
				Reporter:       reporter,
			})
			if err != nil {
				reporter.Abort()
				return err
			}
			if result.PathWarning != "" {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "gg-lint: warning: %s\n", result.PathWarning); err != nil {
					return err
				}
			}
			if !result.Updated {
				_, err = fmt.Fprintf(out, "gg-lint %s is already up to date (latest stable is %s).\n", result.Previous, result.Version)
				return err
			}
			if tag == "" && result.Previous != result.Version {
				_, err = fmt.Fprintf(out, "Updated gg-lint %s to %s at %s.\n", result.Previous, result.Version, result.Path)
			} else {
				_, err = fmt.Fprintf(out, "Installed gg-lint %s at %s.\n", result.Version, result.Path)
			}
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(out, pathHelp(result))
			return err
		},
	}
	cmd.Flags().StringVar(&tag, "version", tag, "release tag to install, including prereleases (env GG_LINT_VERSION; default: latest stable)")
	cmd.Flags().StringVar(&binDir, "bin-dir", binDir, "install directory (env GG_LINT_BIN_DIR; default: the install scripts' directory)")
	cmd.Flags().BoolVar(&noPathUpdate, "no-path-update", noPathUpdate, "Windows only: do not add the install directory to the user PATH (env GG_LINT_NO_PATH_UPDATE=1)")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress step progress; print only the final result and errors")
	return cmd
}
