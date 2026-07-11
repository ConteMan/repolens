# M6 真实规模数据样本

- 主仓库：`/Users/mei/Workspaces/regulated-platform/architecture-and-config-migration-scenarios/repolens-documentation-monorepo`
- 路径池：`/Users/mei/Library/CloudStorage/OneDrive-ExampleOrg/Engineering/Platform/Customer-Enablement/2026-q3/partner-integration-reference-site`、`/Volumes/Engineering Archive/Legacy Migration/2022-2026/transaction-processing-platform-and-disaster-recovery-handbook`、`/Users/mei/Projects/open-source/localization/zh-Hans/developer-experience-and-configuration-samples`。
- `site.title`：`Regulated Platform Architecture and Configuration Migration Reference`；`site.home`：`docs/architecture/2026-q3/long-running-migration-and-rollout-guide.md`。
- `rules`：`docs/architecture-and-governance/**/*.md` 的 `markdown.math: true`、`docs/partner-integrations/legacy-embedded-reports/**/*.html` 的 `html.view: direct`、`internal/customer-data-export/**` 的 `render: false`；`theme.vars.accent: "#0969da"`、`theme.css: docs/assets/regulated-platform-overrides.css`。
- 读取警告：仓库内 `source`、`output`、`access` 与 `unknown_future_option`；它们不得进入写回。
- 失败：`rules[1].match: "["`、`render.max_file_size: "six MB"`、`view.toc_panel: docked`、YAML 缩进错误、`theme.New` 找不到 CSS。成功结果为 `Files: 438`、`Pages: 721`、`Duration: 3.842s`；缓存路径为 `/Users/mei/Library/Caches/repolens/ui/builds/9d6bc2d6e52d0e1e421b4d0a4cf7af832f268bc1e8b9f8a4b4a129b8cda3d5f1`。
