# 002: 配置加载与规则级联（internal/config）

- 状态：已实现
- 关联：roadmap M2、ADR-005（双信任域）、[design/config.md](../design/config.md)

## 问题

实现 design/config.md 定义的配置模型：双信任域合并、白名单校验、按路径的级联规则求值。所有下游包只面对求值后的结果，不自行解析配置。

## 行为

1. **Schema 定义**：按 design/config.md 的 schema 定义 Go 结构体（YAML tag），所有字段有零值可用的默认值。`max_file_size` / `max_size` 支持 `5MB` 风格的人类可读写法（自定义 UnmarshalYAML）。
2. **加载顺序与优先级**：内置默认值 ← 仓库内 `.repolens.yml`（树根，若存在）← 外部 `--config` 文件 ← CLI flag。逐字段覆盖（非整段替换）；`rules` 为例外：仓库内与外部的规则**拼接**（仓库内在前、外部在后，外部因此天然后置覆盖）。
3. **信任域白名单**：仓库内配置只允许 `site` / `ignore` / `render` / `rules` / `theme` / `view` / `agent`。出现 `source` / `output` / `access` 时忽略该段并产生 Warning（不报错）。
4. **规则级联求值**：`OptionsFor(path)` 从全局 `render` 出发，按序对每条 `match` 命中的规则做字段级覆盖，返回最终 `FileOptions`。glob 语义为 doublestar（`**` 跨目录），匹配对象是 repo 相对斜杠路径。依赖 `github.com/bmatcuk/doublestar/v4`（主流且在维护，符合 ADR-003 标准，本 spec 即为其论证记录）。
5. **ignore 求值**：`Ignored(path)` 按 `ignore` glob 列表判断；`.git/**` 与 `.repolens.yml` 恒被忽略。
6. **校验与告警**：加密路径与 agent 输出开启时的交集告警（design/config.md 的 lint 规则）；未知字段告警（yaml 严格模式）；Warning 统一返回给调用方，由 CLI 决定输出方式。

## 接口契约

```go
package config

type Config struct {
    Source Source; Output Output; Access Access
    Site Site; Ignore []string; Render FileOptions
    Rules []Rule; Theme Theme; View View; Agent Agent
}

type Rule struct {
    Match  string
    Render *bool // nil = 未设置；false = 只镜像不出浏览页
    // 其余字段与 FileOptions 同构，指针类型表达"未设置"
    Markdown *MarkdownOptions; HTML *HTMLOptions; Code *CodeOptions
    MaxFileSize *int64
}

type FileOptions struct {
    Render      bool // false = 只镜像
    Markdown    MarkdownOptions
    HTML        HTMLOptions // View: "embed" | "direct" | "source"
    Code        CodeOptions
    MaxFileSize int64
}

type Warning struct{ Msg string }

// Flags 为 CLI 可信域覆盖；空字符串表示未设置。
type Flags struct {
    Repo string
    Ref string
    OutputDir string
    BasePath string
}

// Load 合并全部来源。repoRoot 为物化树根（找 .repolens.yml），
// externalPath 可为空，flags 由 CLI 层构造。
func Load(repoRoot, externalPath string, flags Flags) (*Config, []Warning, error)

func (c *Config) OptionsFor(path string) FileOptions
func (c *Config) Ignored(path string) bool
```

## 边界与非目标

- 不做配置热加载（serve 的重建即重新 Load）；
- 不实现 `access.encrypt` 的加密本身（v1 Out），但 schema 字段与互斥告警要有；
- 环境变量只在 `password_env` 这类"引用名"字段读取，不做全局 env 展开。

## 验收

- 表驱动测试覆盖：默认值、仓库内白名单拦截（含 Warning）、外部覆盖仓库内、CLI 覆盖外部、rules 拼接顺序、级联覆盖（后规则覆盖前规则、规则覆盖全局）、`render: false`、`Ignored` 恒排除项、大小写法解析、加密×agent 互斥告警；
- `OptionsFor` 对同一 path 幂等且无副作用；
- `gofmt` / `go vet` / `go test` 通过。
