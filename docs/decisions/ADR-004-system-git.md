# ADR-004: Git 层 shell out 系统 git 而非 go-git

- 状态：已接受
- 日期：2026-07-04

## 背景

需要 clone / fetch / ls-tree / log 四类 git 能力。候选：纯 Go 的 go-git，或调用系统 git CLI。

## 决策

Shell out 到系统 git，以薄接口封装（`internal/source`）。

## 理由

- **决定性理由是私有仓库认证**：用户已配置的 SSH key、credential helper、token 缓存走系统 git 全部直接可用；go-git 需要自建认证配置，恰好在最重要的私有仓库场景制造摩擦；
- 大仓库性能、shallow clone、边角 git 行为，CLI 更可靠；
- 目标用户是 git 用户，环境装有 git 等于零门槛。

## 后果

- 运行环境要求 PATH 上有 git，文档需声明；
- 接口保持薄，未来如需纯 Go 分发可换 go-git 而不伤架构；
- 内容集合来自 `git ls-tree`（非工作目录），构建可复现；git 元数据一次 `git log --name-status` 全量建映射，禁止逐文件调用。
