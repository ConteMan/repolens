# Agent Skills

面向 repolens **使用者**的 Agent 操作规范：帮助编码 Agent 在你自己的文档仓库里正确使用 repolens 的可选能力。

源文件在仓库 `skills/` 下，同时通过 `go:embed` 编入二进制，由 `repolens skill install` 安装到各 Agent 产品扫描的目录。skill 不参与站点构建。

| Skill | 用途 |
|---|---|
| [repolens-glossary](repolens-glossary/SKILL.md) | 建立和维护术语标注与解释（[spec 017](../docs/specs/017-glossary-terms.md)） |

## 格式：Agent 中立

每个 skill 是一个目录，主文件为 `SKILL.md`：YAML front matter 描述元信息，正文是纯 Markdown 操作规范。刻意不绑定任何一家 Agent 产品：

- 不出现具体工具名（不写"用 Read 工具"，写"读取文件"）；
- 不依赖特定的 skill 加载机制、目录约定或配置格式；
- 需要执行的命令一律写成 shell 命令，并给出无法执行命令时的替代路径。

任何能读文件、能改文件的编码 Agent 都可以直接使用。

## 接入方式

首选随当前二进制安装（离线；会写入来源与版本信息，升级后可 `repolens skill update`）：

```sh
repolens skill install glossary
```

不带 `--target` 时，命令按仓库里的 Agent 目录探测安装位置；都没探测到则写入中立路径 `.agents/skills/`。可用 `--target claude,codex` 显式指定，`--global` 安装到个人目录。

`repolens skill list` 查看内置与已安装副本；`repolens skill update` 把未改动过的过期副本更新到当前二进制版本。

也可以继续手工接入，本质都是让 Agent 读到 `SKILL.md`：

- **直接引用**：把 `SKILL.md` 复制到你仓库的任意位置，在对话中让 Agent 读取它；
- **仓库级约定**：在你仓库的 `AGENTS.md` / `CLAUDE.md` / `.cursor/rules` 等入口文件中，指向该文件的路径；
- **原生 skill 机制**：支持 skill 目录的工具可直接放置整个目录，front matter 的 `name` 与 `description` 即为触发元信息；
- **一次性使用**：把 `SKILL.md` 全文粘贴给 Agent。

## 维护

`SKILL.md` 是自洽的：使用者按它书写即可，无需查阅 repolens 的设计文档，正文也不引用它们。

skill 与实现的一致性由 repolens 维护者保证——渲染行为或格式变更时，`docs/specs/` 下的 spec、`SKILL.md` 与 `example/` 在同一 PR 内同步，并由 CI 对 `example/` 的构建检查兜底（见 [spec 018](../docs/specs/018-glossary-skill.md)）。安装副本与二进制版本是否一致，由 `repolens skill list` / `update` 保证（见 [spec 019](../docs/specs/019-skill-install.md)）。

按 skill 书写却得不到预期结果，属于 repolens 的 bug，请提 Issue。
