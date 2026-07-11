# AGENTS.md — repolens Agent 工作指引

repolens 是一个 Go 编写的单二进制 CLI 工具：把任意 Git 仓库原样渲染成可部署到任何静态托管的浏览站点。MIT 开源，AI Agent 主导开发，文档先行。

## 语言约定

- 与维护者的对话、思考过程、任务清单：**中文**。
- 代码、注释、commit message、Issue / PR 正文与模板：**英文**。
- **主文档双语，中文为默认**：README.md / CONTRIBUTING.md 为中文，README.en.md / CONTRIBUTING.en.md 为英文镜像，两份必须在同一 PR 内联动更新，顶部保留语言切换链接。
- `docs/` 下的设计文档、ADR、spec：**中文**。

## 入项阅读路径（按序）

1. [docs/design/architecture.md](docs/design/architecture.md) — 定位、双层输出、URL 方案、浏览层交互、Agent 视图、访问控制
2. [docs/design/config.md](docs/design/config.md) — 配置模型：双信任域、级联规则
3. [docs/decisions/](docs/decisions/README.md) — 已定且不可轻易推翻的架构决策（ADR）
4. [docs/roadmap.md](docs/roadmap.md) — v1 范围（明确的 In / Out）与里程碑

## 硬规则

1. **文档先行**：改变架构、配置 schema、URL 约定、公开 CLI 行为之前，先更新对应 design 文档或新增 ADR，同一 PR 内文档与代码联动。
2. **ADR 不可静默推翻**：要推翻已有 ADR，新增一条 ADR 声明取代关系，说明理由。
3. **Commit 遵循 Conventional Commits**（见 CONTRIBUTING.md），一个 commit 一个逻辑变更。
4. **新增依赖需要论证**：Go 直接依赖控制在个位数，候选库必须主流且在维护（参照 ADR-003 / ADR-006）。Node 工具链仅允许用于 `repolens ui` 的开发构建；最终发布仍须是无需 Node 的单二进制。
5. **质量门禁**：提交前 `gofmt -l .` 无输出、`go vet ./...`、`go test ./...`、`go build ./...` 全部通过；CI 红灯的 PR 不合并。
6. **范围纪律**：roadmap 中标记 Out of v1 的能力（搜索、多仓库聚合、主题市场等）不做，除非 roadmap 先修订。
7. **零外部请求约束**：生成站点的任何页面不得引用外部 CDN / 字体 / 脚本，所有资源 embed 进二进制并输出到站点内。

## 协作模式：主控 Agent 与实现 Agent

本项目采用**主控 ＋ 实现**的分工：主控 Agent（与维护者对话的会话）负责设计决策、spec 撰写、review 与集成把关；实现 Agent 以**单份 spec 为工作单元**独立推进。

实现 Agent 必须遵守（详见 [docs/specs/README.md](docs/specs/README.md)）：

1. 只做 spec 范围内的事，接口契约是包间合同，调整必须回写 spec（同一 PR）；
2. spec 与 design / ADR 冲突或有遗漏时，停下来报告，不自行拍板；
3. 完成定义 = spec 验收项全过 ＋ 质量门禁绿 ＋ spec 状态改"已实现"；
4. 一份 spec 一个 PR。

## 常用命令

```sh
go build ./...        # 构建
go test ./...         # 测试
gofmt -l .            # 格式检查（应无输出）
go vet ./...          # 静态检查
```

## 仓库结构速览

- `cmd/repolens/` — main 入口
- `internal/cli/` — cobra 命令定义
- `internal/config/` — 配置加载、双信任域合并、规则级联
- `internal/source/` — git 操作（shell out 到系统 git）
- `internal/render/` — goldmark / chroma 渲染管线
- `internal/site/` — 站点组装：镜像层 + 浏览层 + Agent 视图
- `internal/theme/` — 内置模板与静态资源（go:embed）
- `internal/server/` — 本地预览 serve + watch
- `internal/ui/` — React + TypeScript + Base UI 本地管理界面，静态产物 go:embed
- `docs/` — 设计文档 / ADR / spec / roadmap（也是 repolens 自己的 dogfood 语料）
