// Package cli defines the repolens command-line interface.
package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the repolens version, overridable at build time via
// -ldflags "-X github.com/ConteMan/repolens/internal/cli.Version=v1.2.3".
var Version = "dev"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "repolens",
		Short:         "Turn any Git repository into a browsable static site",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newBuildCmd(), newServeCmd(), newVersionCmd())
	return root
}

// Execute runs the root command and reports any error to stderr.
func Execute() error {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(root.ErrOrStderr(), "Error:", err)
		return err
	}
	return nil
}

var errNotImplemented = errors.New("not implemented yet, see docs/roadmap.md")

func newBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [repo-url|path]",
		Short: "Build a static site from a Git repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errNotImplemented
		},
	}
	cmd.Flags().String("config", "", "path to an external config file (trusted domain)")
	cmd.Flags().String("ref", "", "branch, tag or commit to build (default: remote HEAD)")
	cmd.Flags().StringP("output", "o", "dist", "output directory")
	return cmd
}

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve [path]",
		Short: "Preview a repository locally with live rebuild",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errNotImplemented
		},
	}
	cmd.Flags().String("config", "", "path to an external config file (trusted domain)")
	cmd.Flags().String("addr", "127.0.0.1:8788", "listen address")
	cmd.Flags().Bool("worktree", false, "render the working tree instead of the git tree")
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the repolens version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "repolens", Version)
		},
	}
}
