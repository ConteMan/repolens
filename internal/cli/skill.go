package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ConteMan/repolens/internal/skill"
	"github.com/spf13/cobra"
)

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Install and update bundled agent skills",
	}
	cmd.AddCommand(newSkillListCmd(), newSkillInstallCmd(), newSkillUpdateCmd())
	return cmd
}

func newSkillListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List bundled and installed skills",
		Args:  cobra.NoArgs,
		RunE:  runSkillList,
	}
}

func newSkillInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <name>",
		Short: "Install a bundled skill into agent skill directories",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillInstall(cmd, args[0])
		},
	}
	cmd.Flags().String("target", "", "comma-separated target keys (claude,codex,cursor,copilot,agents)")
	cmd.Flags().Bool("global", false, "install into user-level directories")
	cmd.Flags().String("dir", ".", "project directory used for detection and project-scope install")
	cmd.Flags().Bool("force", false, "overwrite locally modified or foreign copies")
	cmd.Flags().Bool("dry-run", false, "report actions without writing files")
	return cmd
}

func newSkillUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update installed skill copies to the current binary version",
		Args:  cobra.NoArgs,
		RunE:  runSkillUpdate,
	}
	cmd.Flags().String("dir", ".", "project directory")
	cmd.Flags().Bool("force", false, "overwrite locally modified copies")
	cmd.Flags().Bool("dry-run", false, "report actions without writing files")
	return cmd
}

func runSkillList(cmd *cobra.Command, _ []string) error {
	builtins, err := skill.Builtin()
	if err != nil {
		return err
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	version := ResolveVersion()
	installed, err := skill.Scan(root, version)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "内置:")
	fmt.Fprintln(out)
	for _, s := range builtins {
		fmt.Fprintf(out, "  %s (%s)  版本 %s\n", s.Name, s.Alias, version)
		fmt.Fprintf(out, "    %s\n", firstSentence(s.Description))
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "已安装:")
	fmt.Fprintln(out)
	if len(installed) == 0 {
		fmt.Fprintln(out, "  未发现已安装的副本。使用 `repolens skill install <name>` 安装。")
		return nil
	}
	for _, item := range installed {
		fmt.Fprintf(out, "  %s  %s  %s\n", item.Path, displayVersion(item.Version), displayState(item, version))
	}
	return nil
}

func runSkillInstall(cmd *cobra.Command, name string) error {
	targetFlag, err := cmd.Flags().GetString("target")
	if err != nil {
		return err
	}
	global, err := cmd.Flags().GetBool("global")
	if err != nil {
		return err
	}
	dirFlag, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}

	s, ok, err := skill.Lookup(name)
	if err != nil {
		return err
	}
	if !ok {
		return unknownSkillError(name)
	}

	root, err := filepath.Abs(dirFlag)
	if err != nil {
		return err
	}
	scope := skill.ScopeProject
	if global {
		scope = skill.ScopeGlobal
	}

	targets, fallback, err := resolveInstallTargets(root, scope, targetFlag)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if fallback {
		fmt.Fprintln(out, "未探测到已知 Agent 产品目录，已回退到中立路径 agents（.agents/skills/）。可用 --target 指定目标：claude,codex,cursor,copilot,agents。")
	}

	opts := skill.Options{Force: force, DryRun: dryRun}
	version := ResolveVersion()
	for _, t := range targets {
		res, err := skill.Install(s, t, version, opts)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, formatSkillResult(res))
	}
	fmt.Fprintln(out, "请重启 Agent 会话以使 skill 目录重新被扫描。")
	return nil
}

func runSkillUpdate(cmd *cobra.Command, _ []string) error {
	dirFlag, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}
	root, err := filepath.Abs(dirFlag)
	if err != nil {
		return err
	}
	results, err := skill.Update(root, ResolveVersion(), skill.Options{Force: force, DryRun: dryRun})
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if len(results) == 0 {
		fmt.Fprintln(out, "未发现带 provenance 的已安装副本。")
		return nil
	}
	for _, res := range results {
		fmt.Fprintln(out, formatSkillResult(res))
	}
	fmt.Fprintln(out, "请重启 Agent 会话以使 skill 目录重新被扫描。")
	return nil
}

func resolveInstallTargets(root string, scope skill.Scope, targetFlag string) ([]skill.Target, bool, error) {
	if strings.TrimSpace(targetFlag) != "" {
		targets, err := skill.ResolveTargets(root, scope, splitCSV(targetFlag))
		return targets, false, err
	}
	detected, err := skill.Detect(root, scope)
	if err != nil {
		return nil, false, err
	}
	keys := make([]string, 0, len(detected))
	for _, t := range detected {
		keys = append(keys, t.Key)
	}
	fallback := false
	if len(keys) == 0 {
		keys = []string{"agents"}
		fallback = true
	}
	targets, err := skill.ResolveTargets(root, scope, keys)
	return targets, fallback, err
}

func unknownSkillError(name string) error {
	all, err := skill.Builtin()
	if err != nil {
		return fmt.Errorf("unknown skill %q", name)
	}
	parts := make([]string, 0, len(all))
	for _, s := range all {
		parts = append(parts, s.Name+" ("+s.Alias+")")
	}
	return fmt.Errorf("unknown skill %q (available: %s)", name, strings.Join(parts, ", "))
}

func splitCSV(s string) []string {
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func formatSkillResult(res skill.Result) string {
	verb := "跳过"
	switch res.Action {
	case skill.ActionCreated:
		verb = "新建"
	case skill.ActionUpdated:
		verb = "更新"
	}
	extra := ""
	if res.Action == skill.ActionSkipped && res.State == skill.StateCurrent && res.Warning == "" {
		extra = "  已是最新"
	}
	if res.Warning != "" {
		extra = "  " + warningZH(res.Warning)
	}
	return verb + "  " + res.Path + extra
}

func warningZH(w string) string {
	switch w {
	case skill.WarnModified:
		return "正文已被本地修改，使用 --force 覆盖"
	case skill.WarnForeign:
		return "存在非 repolens 安装的同名 skill，使用 --force 覆盖"
	case skill.WarnUnbundled:
		return "记录的 skill 不在内置列表中，已跳过且未删除"
	default:
		return w
	}
}

func displayVersion(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

func displayState(item skill.Installed, current string) string {
	switch item.State {
	case skill.StateCurrent:
		return "最新"
	case skill.StateOutdated:
		return fmt.Sprintf("过期（%s → %s）", item.Version, current)
	case skill.StateModified:
		return "本地已修改"
	case skill.StateForeign:
		return "非 repolens 安装"
	default:
		return item.State.String()
	}
}

func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "。"); i >= 0 {
		return s[:i+len("。")]
	}
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i+1]
	}
	return s
}
