# 001: Git 内容源（internal/source）

- 状态：已实现
- 关联：roadmap M2、ADR-004（系统 git）

## 问题

构建需要从 Git 仓库（远端 URL 或本地路径）获得一个**确定的内容集合**：文件清单、文件内容、每个文件的最后修改 commit。内容必须来自 git 树而非工作目录，保证构建只由 `repo + ref` 决定。

## 行为

1. **打开源**：
   - 远端 URL（ssh / https）：bare clone 到缓存目录 `os.UserCacheDir()/repolens/repos/<sha256(url) 前 16 位>`；已存在则 `git fetch`。所有 git 命令通过 `exec.Command` 调用系统 git，认证完全复用用户环境（ADR-004）。
   - 本地路径：直接以该路径为 git 目录，不复制。
2. **解析 ref**：默认远端 HEAD（`git ls-remote --symref` 或 clone 后的 HEAD）；`--ref` 可指定分支 / tag / commit。解析为 commit hash 后全程使用 hash。
3. **物化树**：`git archive <hash> | tar -x` 到构建临时目录（`os.MkdirTemp`）。后续读文件走文件系统。symlink 不跟随、不输出（archive 默认行为即可，需测试确认）。
4. **文件清单**：遍历物化目录生成 `[]File`（路径为 repo 相对、斜杠分隔、按字典序）。
5. **git 元数据**：一次 `git log --name-status --format=...` 从 `<hash>` 向前扫描，建立 path → 最后修改 commit（hash、时间、subject）映射。**禁止逐文件 git log**。重命名（R 状态）按新路径记录。扫描有 `--max-count` 上限（默认 10000，可未来配置化），超出的老文件允许无元数据（`LastCommit == nil`）。
6. **worktree 模式**（仅 serve 用）：跳过 2–5 中的 git 树逻辑，直接以本地工作目录为物化树，git 元数据尽力而为（无 git 仓库时全部为 nil）。

## 接口契约

```go
package source

type Spec struct {
    Repo     string // URL 或本地路径
    Ref      string // 空 = HEAD
    Worktree bool
}

type Commit struct {
    Hash    string
    Time    time.Time
    Subject string
}

type File struct {
    Path       string // repo 相对路径，斜杠分隔
    Size       int64
    LastCommit *Commit // 可能为 nil
}

type Tree struct {
    Root       string // 物化后的文件系统根目录
    CommitHash string // worktree 模式为空
    Files      []File // 字典序
}

// Open 物化内容集合。调用方负责在构建结束后 Cleanup。
func Open(ctx context.Context, spec Spec) (*Tree, error)
func (t *Tree) Cleanup() error
```

## 边界与非目标

- 不做 submodule、不做 git-lfs 实体拉取（LFS 指针文件按普通文件处理，v1 不解析）；
- 不做 shallow clone 优化（bare clone + fetch 已够，未来可加 `--filter`）；
- 认证失败、ref 不存在等错误原样透出 git stderr，包一层 `source: ` 前缀，不吞不猜。

## 验收

- 对远端仓库和本地路径均能 `Open` 出一致的 `Tree`；二次 `Open` 远端走 fetch 不重新 clone；
- `Files` 与 `git ls-tree -r --name-only <hash>` 输出一致（不含目录项；symlink 依行为 3 排除，此为与 ls-tree 的预期差异）；
- 任选文件的 `LastCommit.Hash` 与 `git log -1 --format=%H <hash> -- <path>` 一致；
- 单元测试用 `t.TempDir()` 内动态创建的 git 仓库，不依赖网络；
- `gofmt` / `go vet` / `go test` 通过。
