// Package cli defines the repolens command-line interface.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/ConteMan/repolens/internal/server"
	"github.com/ConteMan/repolens/internal/site"
	"github.com/ConteMan/repolens/internal/source"
	"github.com/ConteMan/repolens/internal/theme"
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

func newBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [repo-url|path]",
		Short: "Build a static site from a Git repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			ref, err := cmd.Flags().GetString("ref")
			if err != nil {
				return err
			}
			output, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}

			flags := config.Flags{Ref: ref}
			if len(args) > 0 {
				flags.Repo = args[0]
			}
			if cmd.Flags().Changed("output") {
				flags.OutputDir = output
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			initialCfg, _, err := config.Load("", configPath, flags)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Opening source %s...\n", initialCfg.Source.Repo)
			tree, err := source.Open(ctx, source.Spec{
				Repo: initialCfg.Source.Repo,
				Ref:  initialCfg.Source.Ref,
			})
			if err != nil {
				return err
			}
			defer tree.Cleanup()

			cfg, warnings, err := config.Load(tree.Root, configPath, flags)
			if err != nil {
				return err
			}
			overrideDir := resolveSourcePath(tree.Root, cfg.Theme.Templates)
			customCSS := resolveSourcePath(tree.Root, cfg.Theme.CSS)
			renderer, err := theme.New(overrideDir, customCSS, cfg.Theme.Vars)
			if err != nil {
				return err
			}

			outDir := cfg.Output.Dir
			fmt.Fprintf(cmd.OutOrStdout(), "Building site into %s...\n", outDir)
			stats, err := site.NewBuilder(cfg, renderer).Build(ctx, tree, outDir)
			if err != nil {
				return err
			}
			warnings = append(warnings, stats.Warnings...)
			fmt.Fprintf(cmd.OutOrStdout(), "Built %d files and %d pages in %s.\n", stats.Files, stats.Pages, time.Since(start).Round(time.Millisecond))
			for _, warning := range warnings {
				fmt.Fprintf(cmd.OutOrStdout(), "Warning: %s\n", warning.Msg)
			}
			return nil
		},
	}
	cmd.Flags().String("config", "", "path to an external config file (trusted domain)")
	cmd.Flags().String("ref", "", "branch, tag or commit to build (default: remote HEAD)")
	cmd.Flags().StringP("output", "o", "dist", "output directory")
	return cmd
}

func resolveSourcePath(root, p string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(root, filepath.FromSlash(p))
}

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve [path]",
		Short: "Preview a repository locally with live rebuild",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			if configPath != "" {
				configPath, err = filepath.Abs(configPath)
				if err != nil {
					return err
				}
			}
			addr, err := cmd.Flags().GetString("addr")
			if err != nil {
				return err
			}
			worktree, err := cmd.Flags().GetBool("worktree")
			if err != nil {
				return err
			}

			repo := "."
			if len(args) > 0 {
				repo = args[0]
			}
			repoDir, err := filepath.Abs(repo)
			if err != nil {
				return err
			}
			info, err := os.Stat(repoDir)
			if err != nil {
				return err
			}
			if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", repo)
			}

			previousDir, err := os.Getwd()
			if err != nil {
				return err
			}
			if err := os.Chdir(repoDir); err != nil {
				return err
			}
			defer func() {
				_ = os.Chdir(previousDir)
			}()

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			flags := config.Flags{Repo: repoDir}
			rebuild := func(ctx context.Context) (string, error) {
				start := time.Now()
				initialCfg, _, err := config.Load("", configPath, flags)
				if err != nil {
					return "", err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Opening source %s...\n", initialCfg.Source.Repo)
				tree, err := source.Open(ctx, source.Spec{
					Repo:     initialCfg.Source.Repo,
					Ref:      initialCfg.Source.Ref,
					Worktree: worktree,
				})
				if err != nil {
					return "", err
				}
				defer tree.Cleanup()

				cfg, warnings, err := config.Load(tree.Root, configPath, flags)
				if err != nil {
					return "", err
				}
				overrideDir := resolveSourcePath(tree.Root, cfg.Theme.Templates)
				customCSS := resolveSourcePath(tree.Root, cfg.Theme.CSS)
				renderer, err := theme.New(overrideDir, customCSS, cfg.Theme.Vars)
				if err != nil {
					return "", err
				}

				outDir, err := os.MkdirTemp("", "repolens-serve-*")
				if err != nil {
					return "", err
				}
				// site.Build 拒绝清空无哨兵的已存在目录：删掉空壳让它自建。
				if err := os.Remove(outDir); err != nil {
					return "", err
				}
				keepOutDir := false
				defer func() {
					if !keepOutDir {
						_ = os.RemoveAll(outDir)
					}
				}()

				fmt.Fprintf(cmd.OutOrStdout(), "Building preview into %s...\n", outDir)
				stats, err := site.NewBuilder(cfg, renderer).Build(ctx, tree, outDir)
				if err != nil {
					return "", err
				}
				warnings = append(warnings, stats.Warnings...)
				fmt.Fprintf(cmd.OutOrStdout(), "Built %d files and %d pages in %s.\n", stats.Files, stats.Pages, time.Since(start).Round(time.Millisecond))
				for _, warning := range warnings {
					fmt.Fprintf(cmd.OutOrStdout(), "Warning: %s\n", warning.Msg)
				}
				keepOutDir = true
				return outDir, nil
			}

			return server.Run(ctx, server.Options{Addr: addr, Worktree: worktree}, rebuild)
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
