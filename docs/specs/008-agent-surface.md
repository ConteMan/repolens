# 008: Agent 视图（internal/site）

- 状态：已实现
- 关联：roadmap M4、[design/architecture.md](../design/architecture.md) Agent 视图一节

## 问题

让 AI Agent 无需渲染 HTML 即可发现并抓取站点全部内容：约定说明、内容清单、机器可读索引、逐页 raw 指引。

## 行为

1. **llms.txt**（`agent.llms_txt`，默认开）：站点根输出，格式遵循 llms.txt 惯例（H1 站点名 ＋ 引用块简介 ＋ 分节链接列表）：
   - 头部说明两条路径约定：原始文件在 `/<repo-path>`，浏览页在 `/view/<repo-path>/`；
   - 内容清单按目录分节，列出全部 Markdown 文件（`- [标题](相对镜像路径): 首段摘要 ≤120 字符`），标题取 spec 003 的 Title；
   - 尾部指向 `index.json` 与（若开启）`llms-full.txt`。
2. **llms-full.txt**（`agent.llms_full`，默认开、`max_size` 默认 2MB）：全部 Markdown 与纯文本（v1 按扩展名界定：`.txt` / `.text` / `.log`——比 Classify 的 Code 口径窄，避免把代码与数据文件拼入，实现时确认 2026-07-05）文件原文拼接，每篇前加 `----- <repo-path> -----` 分隔头，按树序；写满 `max_size` 即截断并在文末标注 `[truncated]`。
3. **index.json**（`agent.index_json`，默认开）：
   ```json
   {
     "generator": "repolens <version>",
     "commit": "<hash>",
     "built_at": "<RFC3339>",
     "site": {"title": "..."},
     "files": [
       {"path": "docs/foo.md", "kind": "markdown", "size": 1234,
        "title": "Foo", "modified": "<RFC3339|null>",
        "raw": "docs/foo.md", "view": "view/docs/foo.md/"}
     ]
   }
   ```
   `files` 含全部非 ignore 文件（含二进制，title 仅 Markdown 有）；路径均为站点根相对，不带前导 `/`。补充语义（实现时确认，2026-07-05）：`render:false` 的文件 `view` 为 `null`；合并进目录页的 `index.html`（spec 005 冲突规则）`view` 指向目录页 URL；worktree 模式无 commit hash，`commit` 为 `null`。
4. **逐页指引**：每个 Markdown 浏览页 `<head>` 注入 `<link rel="alternate" type="text/markdown" href="<相对镜像路径>">`（spec 006 的 `HeadExtra` 通道）。
5. **互斥告警**：`access.encrypt.paths` 与本 spec 输出集合有交集时构建 Warning（spec 002 已定义，此处消费）。

## 边界与非目标

- 不做向量化、不做站点地图 sitemap.xml（noindex 定位下无意义）；
- llms-full 不含代码文件（清单里有、原文不拼，控制体积；Agent 需要时按 raw 路径取）。

## 验收

- 对 testdata 仓库构建后：llms.txt 结构、摘要截断、约定说明存在；llms-full 分隔头与截断标注；index.json 通过 `json.Valid` 且字段齐全、与文件系统一致；
- 三个开关分别关闭时对应产物不生成、其余不受影响；
- `rel=alternate` 在 Markdown 浏览页存在、指向正确相对路径；
- `gofmt` / `go vet` / `go test` 通过。
