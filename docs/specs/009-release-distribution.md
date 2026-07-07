# 009: 发布流水线与分发（GoReleaser / Homebrew / Windows）

- 状态：已确认
- 关联：roadmap M7、v1 迭代清单（version 显示 dev）

## 问题

当前安装唯一路径是 `go install`，要求用户有 Go 环境——对目标用户（把文档仓给非技术读者看的维护者）门槛过高。且 `go install` 安装的二进制 `repolens version` 显示 `dev`。需要：多平台预编译二进制、主流包管理器分发、明确的升级路径。

## 行为

1. **GoReleaser 流水线**：推送 `v*` tag 触发 GitHub Actions，自动产出：
   - darwin / linux / windows × amd64 / arm64 共 6 个构建，`-ldflags` 注入版本；
   - 归档（mac/linux 为 tar.gz，windows 为 zip，内含单 exe）＋ `checksums.txt`；
   - GitHub Release 自动生成（changelog 取 CHANGELOG.md 对应节）。
2. **版本显示修复**：`cli.Version` 保持 ldflags 注入为主；为空时回退 `debug.ReadBuildInfo().Main.Version`（覆盖 `go install @vX.Y.Z` 场景），仍无则 `dev`。`index.json` 的 `generator` 同步受益。
3. **Homebrew**：GoReleaser 自动维护 `ConteMan/homebrew-tap` 的 formula；用户侧 `brew install conteman/tap/repolens`、`brew upgrade repolens`。
4. **Windows**：Release 页直接下载 zip（解压即用的单 exe）；提供 Scoop manifest（`scoop bucket add` + `scoop install repolens`）。winget 提交后置（需微软审核流程，v1.x 观望）。
5. **升级方案**：
   - 包管理器安装的：交给 `brew upgrade` / `scoop update`，`repolens upgrade` 检测到该来源时提示对应命令而不自更新；
   - 直接下载 / go install 的：`repolens upgrade` 查 GitHub Releases API，下载校验（checksums）后原子替换自身；
   - **隐私立场**：联网查版本仅发生在用户显式运行 `upgrade` / `upgrade --check` 时，无任何后台或随命令附带的版本上报；产物零外部请求约束不变。

## 边界与非目标

- 不做 apt/rpm/AUR/nix（社区有需求再说）；
- 不做自动后台更新与更新提醒推送；
- winget 观望，不阻塞本 spec 验收；
- 不改变 `go install` 路径的可用性。

## 验收

- tag 一个预发布版本（如 v1.0.1-rc.1）走通全流水线：6 构建产出、checksums 正确、Release 自动建；
- mac（brew）、windows（zip 解压）、linux（tar.gz）三平台实测 `repolens version` 显示正确版本号；
- `go install @vX.Y.Z` 后 `repolens version` 不再显示 `dev`；
- `repolens upgrade` 在直接下载安装下完成一次真实自更新；brew 安装下正确提示改用 `brew upgrade`；
- CI 质量门禁全绿。
