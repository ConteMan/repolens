# 桌面版发布

macOS 桌面版和 CLI 独立编号。桌面版以 `cmd/repolens-desktop/mygo.json` 中的版本生成 `desktop-vX.Y.Z` 标签；CLI 使用根目录的 CHANGELOG、`vX.Y.Z` 标签和 GoReleaser。GitHub 的 Latest 保留给 CLI，MyGo 按 `desktop-v` 前缀检查桌面更新。

| 用途 | Homebrew Cask | 更新 |
| --- | --- | --- |
| 命令行 | `brew install --cask conteman/tap/repolens` | `brew upgrade --cask repolens` |
| macOS 应用 | `brew install --cask conteman/tap/repolens-desktop` | 应用设置内更新，或 `brew upgrade --cask --greedy repolens-desktop` |

桌面 Cask 安装 `Repolens.app`，标记 `auto_updates true`。两个 Cask 可以同时安装。苹果芯片和 Intel Mac 共用 Universal DMG。首发的最低系统版本为 macOS 13，与 Go 编译出的 Mach-O 部署目标一致。

## 打包

维护者在 macOS 上更新桌面版本和 [桌面更新日志](../cmd/repolens-desktop/CHANGELOG.md)。完成设计回归、仓库质量门禁，提交发布源代码后运行：

```sh
python3 scripts/desktop-release.py
```

构建使用 go.mod 固定的 MyGo 版本，更新签名验证需要 OpenSSL 3 或更高版本。可用同一脚本的 `--verify-only` 参数复核已公证的本地构建。Apple Developer ID Application 私钥在系统钥匙串中，公证复用已配置的 `cubbix-notary` 凭据。正常运行和本地调试不需要发布证书。本地开发包可以使用 `go tool mygo build -sign - -skip-notarize -skip-dmg ./cmd/repolens-desktop`。

应用更新使用专属 Ed25519 密钥。默认文件为 `~/Library/Application Support/Repolens/update-keys/mygo-update.key`，权限为 0600；可用 `REPOLENS_UPDATE_KEY_PATH` 指向备份恢复后的文件。私钥只由发布构建读取。已安装的客户端依赖仓内公钥验证后续更新。

## 发布与验收

1. 核对构建输出的版本、标签目标提交，以及 `bin/desktop-release/<版本>/publish` 中的 DMG、更新包、更新清单和 SHA256SUMS。脚本检查签名、公证票据、Universal 架构和签名更新清单。
2. 将已验收的提交推送到主分支，创建指向该提交的 `desktop-vX.Y.Z` 标签。用 `gh release create --draft --verify-tag --latest=false` 和本次桌面更新说明创建草稿，上传 publish 内的文件，核对后用 `gh release edit desktop-vX.Y.Z --draft=false --latest=false` 发布。CLI 的 release 工作流仅监听 `v*`。
3. 在 `ConteMan/homebrew-tap` 中将脚本生成的 `repolens-desktop.rb` 提交到 `Casks/`。先发布 GitHub 资产再更新 Cask。若 tap 更新失败，可以单独重试，复用同一次构建的 DMG 摘要。
4. 从公开 URL 下载并校验 SHA256SUMS。检查 MyGo 清单的版本与归档 URL、桌面 Cask 的版本及 DMG SHA-256，再运行 `brew fetch --cask conteman/tap/repolens-desktop`。确认 `/releases/latest` 仍返回 CLI 标签。
