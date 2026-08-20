# 019: Skill 安装与更新

- 状态：已确认（2026-08-20 维护者确认方向）
- 关联：roadmap M12、Issue #80、[spec 018](018-glossary-skill.md)（本 spec 取代其部分条款）、[ADR-007](../decisions/ADR-007-optional-repo-conventions.md)

## 问题

spec 018 交付了术语标注 skill，但把分发完全留给使用者：从 GitHub 复制 `SKILL.md` 到自己仓库的某个位置。实测这条路径要求使用者自己知道 raw URL、自己选 tag、自己建目录、自己判断放在哪个 Agent 认得的路径下，且升级 repolens 之后没有任何机制提示副本已经过期——skill 与实现的一致由维护者保证（spec 018 行为 5），但**使用者手里那份副本与二进制版本是否一致，没人保证**。

同时外部事实发生了变化，spec 018 写下「不为特定 Agent 产品提供适配层」时它们都还不存在：

- Agent Skills 于 2025-12-18 被开源为公开标准（[agentskills.io](https://agentskills.io/specification)），到 2026 年中已有约 40 个产品实现，`SKILL.md` 的目录形态与 front matter 字段是跨产品统一的；
- 该标准**只定义格式，不定义安装位置**。各家仍各扫各的目录，并出现了 `.agents/skills/` 这一被多家识别的中立路径；
- 分发层出现了成熟先例：GitHub `gh skill`（2026-04-16）把来源与版本信息写进 front matter 的 `metadata`，用内容比对做更新检测。

因此「适配特定产品」的性质变了：适配的不再是某一家的私有约定，而是一个有公开规范和多家实现的目录约定。repolens 内嵌 skill 后可以做到别人做不到的一件事：**更新完全离线**——`brew upgrade repolens` 之后本地重写副本即可，不需要网络、不需要 registry。

## 行为

### 1. skill 随二进制分发

`skills/` 下的 `SKILL.md` 通过 `go:embed` 进入二进制。`skills/` 仍是顶层源码目录，仍是使用者可直接浏览的事实源，spec 018 的一致性检查（行为 7）不变。

skill 不携带独立版本号：**skill 的版本就是二进制的版本**（`cli.ResolveVersion()` 的返回值）。这与 spec 018 行为 5「一致性是维护者的责任」是同一条约定的两面。

### 2. 目录名符合规范

Agent Skills 规范要求 front matter 的 `name` 必须等于父目录名。现有 `skills/glossary/` 与 `name: repolens-glossary` 不符，改名为 `skills/repolens-glossary/`。

命令参数同时接受完整名 `repolens-glossary` 与短名 `glossary`（去掉 `repolens-` 前缀），二者等价。

### 3. 已知安装目标

| key | 项目级目录 | 个人级目录 | 探测信号（项目级） |
|---|---|---|---|
| `claude` | `.claude/skills/` | `~/.claude/skills/` | `.claude/` 存在 |
| `codex` | `.codex/skills/` | `~/.codex/skills/` | `.codex/` 存在 |
| `cursor` | `.cursor/skills/` | `~/.cursor/skills/` | `.cursor/` 存在 |
| `copilot` | `.github/skills/` | `~/.config/github-copilot/skills/` | `.github/skills/` 或 `.github/copilot-instructions.md` 存在 |
| `agents` | `.agents/skills/` | `~/.agents/skills/` | `.agents/` 存在 |

`copilot` 的探测信号不能只看 `.github/` 是否存在——几乎每个仓库都有该目录（CI 配置），据此判定会给所有人装一份用不到的副本。

个人级目录的 `~` 按 `os.UserHomeDir()` 解析；解析失败时个人级目标整体不可用，报错而非静默跳过。

### 4. `repolens skill list`

无参数，输出两段：

- **内置**：名称、短名、版本、`description` 首句；
- **已安装**：扫描全部已知目标（项目级 ＋ 个人级），逐条给出路径、记录的版本、状态（`最新` / `过期（1.6.3 → 1.7.0）` / `本地已修改`）。

没有任何已安装副本时，第二段输出一行提示与 `repolens skill install` 的用法，不报错。

### 5. `repolens skill install <name>`

参数为 skill 名（完整名或短名）。行为：

1. **确定目标**：给了 `--target` 就用给定的（逗号分隔多个 key，未知 key 报错并列出合法值）；没给就按行为 3 的信号探测，装到全部探测到的目标；一个信号都没探到时装到 `agents`（中立路径），并在输出中说明这是兜底选择以及如何用 `--target` 指定。
2. **确定作用域**：默认项目级，写入 `--dir` 指定的目录（默认当前工作目录）；`--global` 改为个人级。
3. **写入** `<target>/<skill-name>/SKILL.md`，注入行为 6 的 provenance。目录不存在则创建。
4. **报告**：逐个目标一行，写明绝对路径与动作（新建 / 更新 / 跳过），末尾提示重启 Agent 会话使其重新扫描 skill 目录。

冲突处理：

| 目标文件状态 | 默认行为 |
|---|---|
| 不存在 | 写入 |
| 存在，有 provenance，版本相同且未被本地修改 | 跳过，报告「已是最新」 |
| 存在，有 provenance，版本不同且未被本地修改 | 覆盖 |
| 存在，有 provenance，正文已被本地修改 | **跳过并告警**，提示 `--force` |
| 存在，无 provenance（使用者自有的同名 skill） | **跳过并告警**，提示 `--force` |

`--force` 覆盖上表的两种告警情形。任何情况下只写 `SKILL.md` 这一个文件，不删除、不改动目标 skill 目录下的其他文件——使用者可能在同目录补充了自己的 `references/`。写入采用「临时文件 ＋ rename」的原子替换。

flags：`--target`、`--global`、`--dir`、`--force`、`--dry-run`（只报告不落盘）。

### 6. Provenance

安装时在 front matter 末尾追加 `metadata` 块：

```yaml
metadata:
  repolens.skill: repolens-glossary
  repolens.version: "1.7.0"
  repolens.digest: "a1b2c3d4e5f6"
```

- 三个键都是字符串值——Agent Skills 规范限定 `metadata` 为 string→string 映射；
- `repolens.version` 取 `cli.ResolveVersion()`；
- `repolens.digest` 是**正文**（front matter 结束标记之后的全部字节）SHA-256 的十六进制前 12 位。只覆盖正文而不覆盖整个文件，是为了让 provenance 自身的存在不影响摘要计算，避免自指；
- 这些键随文件走，副本被复制到别的仓库也不丢失来源与版本。

为使注入是确定性的纯文本拼接而非 YAML 重序列化（后者会打乱正文格式），**内置 SKILL.md 的 front matter 不得包含 `metadata` 字段**，由行为 8 的检查保证。读取时按 YAML 宽松解析。

### 7. `repolens skill update`

扫描全部已知目标，对每个带 provenance 的副本：

- 记录版本与当前二进制版本相同、正文未修改 → 跳过；
- 版本不同、正文未修改 → 重写为内置版本；
- 正文已被本地修改 → 跳过并告警，`--force` 覆盖；
- 记录的 skill 名不在内置列表中（如未来某个 skill 被移除）→ 跳过并告警，不删除。

当前二进制版本为 `dev`，或副本记录的版本为 `dev` 时，一律视为过期并重写——开发态不做版本比较。

flags：`--dir`、`--force`、`--dry-run`。无参数（不按 skill 名过滤）。

### 8. 一致性检查

在 spec 018 行为 7 的检查基础上追加：

- 每个内置 skill 的 front matter `name` 等于其目录名，且符合规范的命名约束（1–64 字符，小写字母数字与连字符，不以连字符开头结尾，无连续连字符）；
- 内置 SKILL.md 的 front matter 不含 `metadata` 字段；
- `go:embed` 出的 skill 集合与 `skills/` 下实际存在的 skill 目录一一对应——新增 skill 目录却未被嵌入应当失败；
- 安装产物可被重新解析：注入 provenance 后的文件仍是合法的 YAML front matter ＋ 正文，且解析回来的三个 `repolens.*` 键与写入值相等；
- `example/` 不进入安装产物。

## 接口契约

新增顶层包 `skills`（仅承载内嵌数据，`go:embed` 要求 `.go` 文件与被嵌文件同处一棵目录树，故不能放在 `internal/` 下）：

```go
// Package skills embeds the user-facing agent skills shipped with repolens.
package skills

//go:embed */SKILL.md
var FS embed.FS
```

嵌入模式 `*/SKILL.md` 只匹配一层目录，`example/` 及其内容天然被排除。

新增 `internal/skill`：

```go
// Skill is a skill shipped inside the binary.
type Skill struct {
	Name        string // directory name, equal to the front matter name
	Alias       string // short name, Name without the "repolens-" prefix
	Description string
	Content     []byte // SKILL.md as embedded, without provenance
}

func Builtin() ([]Skill, error)
func Lookup(name string) (Skill, bool, error) // accepts Name or Alias

type Scope int // ScopeProject | ScopeGlobal

// Target is a directory an agent product scans for skills.
type Target struct {
	Key      string
	Dir      string // absolute
	Scope    Scope
	Detected bool
}

func Targets(root string, scope Scope) ([]Target, error)
func Detect(root string) ([]Target, error) // project scope, Detected only

type State int // StateAbsent | StateCurrent | StateOutdated | StateModified | StateForeign

// Installed is one copy of a skill found on disk.
type Installed struct {
	Path    string
	Target  Target
	Name    string
	Version string
	State   State
}

func Scan(root string) ([]Installed, error)

type Options struct {
	Force  bool
	DryRun bool
}

type Action int // ActionCreated | ActionUpdated | ActionSkipped

type Result struct {
	Path    string
	Action  Action
	State   State // state found before writing
	Warning string
}

func Install(s Skill, t Target, version string, opts Options) (Result, error)
func Update(root string, opts Options) ([]Result, error)
```

`internal/cli` 新增 `newSkillCmd()`，挂载 `list` / `install` / `update` 三个子命令，注册进 `newRootCmd()`。CLI 层只负责 flag 解析与输出格式化，判断逻辑全部在 `internal/skill`。

## 边界与非目标

- **不联网**。不做远程 registry、不与 GitHub 比对、不下载任何东西；能安装的只有当前二进制内嵌的 skill。这是本方案相对 `gh skill` 的取舍，也是它的优势所在；
- **不做 symlink**。默认复制。symlink 在 Windows 与 Git 上的行为差异不值得为省几 KB 承担；
- **不做 uninstall**。删除一个目录，使用者自己做即可，不值得为它引入「删掉了使用者自己加的文件」的风险；
- **不生成 Agent Plugins 的 `plugin.json`**。[Agent Plugins 1.0](https://agent-plugins.org/) 是 skill 之上的打包格式，repolens 只有一个 skill、没有 MCP server，暂不需要；生态稳定后再议；
- **不代改使用者的入口文件**。不往 `AGENTS.md` / `CLAUDE.md` / `.cursor/rules` 写任何内容，只在命令输出中提示；
- **不改变 SKILL.md 正文的 Agent 中立立场**（spec 018 行为 2 保留）。正文照旧不出现产品名与加载机制，适配只发生在安装路径这一层；
- **不给 skill 独立版本号**，不支持安装历史版本；
- 不新增 Go 依赖：YAML 解析复用 `github.com/goccy/go-yaml`。

### 对 spec 018 的取代关系

| spec 018 条款 | 处置 |
|---|---|
| 行为 1（skill 位于 `skills/<name>/`） | 修订：目录名必须等于 front matter `name`，`skills/glossary/` → `skills/repolens-glossary/` |
| 行为 8（不 `go:embed`、不随二进制分发） | **取代**：skill 内嵌进二进制并随发布分发 |
| 非目标「不为特定 Agent 产品提供适配层、安装脚本或打包格式」 | **部分取代**：提供安装命令与目录适配；仍不提供打包格式 |
| 其余条款（Agent 中立、职责边界、自包含、一致性责任、固定样例、一致性检查） | 全部保留 |

spec 018 的状态维持「已实现」，在其文首标注被本 spec 修订的条款。ADR 未被推翻——ADR-007 约束的是仓库可选约定的准入，与 skill 的分发方式无关。

## 验收

- `skills/repolens-glossary/` 目录改名完成，`SKILL.md` 内容与 spec 018 交付时逐字节相同（仅路径变化）；
- `repolens skill list` 在无已安装副本、有最新副本、有过期副本、有本地修改副本四种情形下输出正确状态；
- 在一个既无 `.claude/` 也无其他信号的干净仓库中 `repolens skill install glossary`，产物落在 `.agents/skills/repolens-glossary/SKILL.md`，且输出说明了兜底原因；
- 在同时存在 `.claude/` 与 `.codex/` 的仓库中不带 `--target` 安装，两处各得到一份，输出列出两条绝对路径；
- `--target` 传入未知 key 时报错并列出合法值，不产生任何写入；
- `--dry-run` 在上述所有情形下不产生任何文件系统写入；
- 重复执行同一条 install 命令是幂等的：第二次全部报告「已是最新」，文件字节不变；
- 手工修改已安装副本的正文后，install 与 update 均跳过并告警，`--force` 才覆盖；
- 目标位置存在一个无 provenance 的同名 skill 时不被覆盖，`--force` 才覆盖；
- 安装产物同目录下预置的 `references/note.md` 在 install 与 update 后仍然存在且内容不变；
- 模拟版本升级（构造记录版本为旧值的副本）后 `repolens skill update` 重写该副本，`repolens.version` 更新且 `repolens.digest` 与新正文匹配；
- 安装产物通过 Agent Skills 规范的字段约束检查（name 命名规则、description 非空且 ≤1024 字符、metadata 全为字符串值）；
- 行为 8 的全部检查项以 Go 测试实现并随 CI 执行；
- 文档同步：`skills/README.md` 的「接入方式」改以 `repolens skill install` 为首选路径并保留手工方式；README.md 与 README.en.md 中指向 `skills/glossary/SKILL.md` 的链接随改名更新（两份同一 PR 内联动）；CHANGELOG 由发布 PR 承载，本 PR 不改；
- `gofmt -l .`、`go vet ./...`、`go test ./...`、`go build ./...` 全部通过。
