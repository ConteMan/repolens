# CLI 与桌面版统一发布

从 1.10.0 开始，同一 Git 提交和 `vX.Y.Z` 标签产出一个 GitHub Release。Release 包含 CLI 的六个平台归档、macOS Universal DMG、Windows x64 安装程序、两份签名更新归档和对应清单。CLI 读取 GitHub Latest；桌面端使用 MyGo 检查带 `v` 前缀且包含对应平台清单的正式版本。

| 平台 | 安装 | 升级 |
| --- | --- | --- |
| CLI | Homebrew `conteman/tap/repolens` 或 Scoop `repolens` | 使用原包管理器或独立二进制的 `repolens upgrade` |
| macOS 桌面 | `brew install --cask conteman/tap/repolens-desktop`，或从 Release 下载 DMG | 应用设置，或 `brew upgrade --cask --greedy repolens-desktop` |
| Windows x64 桌面 | 从同一 Release 下载 `repolens-setup-X.Y.Z-amd64.exe` | 应用设置 |

macOS 支持 13 及以上版本。历史桌面版 0.1.0 使用 `desktop-v` 更新渠道，首次迁移通过 Homebrew 或 DMG 安装 1.10.0。Windows 安装程序当前没有 Authenticode 发行者签名，首次运行可能出现 SmartScreen 提示；应用内更新在安装前验证 Ed25519 签名。

## 准备候选

维护者同步修改 `cmd/repolens-desktop/mygo.json`、[桌面更新说明](../cmd/repolens-desktop/CHANGELOG.md) 与根目录 [CHANGELOG](../CHANGELOG.md)。MyGo 在构建时将桌面更新说明写入应用内更新清单。CI 在 Windows 上检查原生工作区并生成候选安装包。完成两端生成结果对比及 Windows runner 的安装、启动、构建、卸载验收，再打正式标签。1.10.0 的 Windows 11 实机体验回归按维护者确认安排在发布后。

在具备 Developer ID 和 Apple 公证凭据的 Mac 上从发布提交构建。需要 NSIS 的 `makensis` 和 OpenSSL 3，可通过 Homebrew 安装 `nsis` 与 `openssl@3`。两个平台的更新使用同一把 Repolens 专用 Ed25519 密钥。默认路径为 `~/Library/Application Support/Repolens/update-keys/mygo-update.key`，权限 0600；可用 `REPOLENS_UPDATE_KEY_PATH` 指定备份恢复后的文件。私钥留在发布机。Apple 公证使用钥匙串中的 `cubbix-notary`。

```sh
python3 scripts/desktop-release.py
python3 scripts/desktop-release.py --verify-only
```

脚本检查 macOS 架构、版本、签名、Gatekeeper 和公证票据，验证两份更新清单的目标地址、版本、归档大小与签名。已验证的资产位于 `bin/desktop-release/X.Y.Z/publish`，桌面 Cask 在上一级目录。

## 创建同一份 Release

1. 在已验收的主分支提交上打 `vX.Y.Z` 标签并推送。Release workflow 生成 CLI 归档、校验和及 Homebrew/Scoop 清单，保存为工作流产物。该工作流只需要仓库读权限。
2. 用 `gh run download RUN_ID -n release-vX.Y.Z-COMMIT -D /tmp/repolens-vX.Y.Z` 下载对应标签和提交的产物。在 `cli` 子目录运行 `shasum -a 256 -c checksums.txt`。归档覆盖 Darwin、Linux、Windows 各自的 amd64 与 arm64。在桌面 publish 目录运行 `shasum -a 256 -c SHA256SUMS`。两端校验文件名称不同，可在同一 Release 中使用。
3. 从根更新日志提取本版说明到 RELEASE_NOTES，在标签对应的干净工作区运行 `python3 scripts/publish-release.py /tmp/repolens-vX.Y.Z RELEASE_NOTES`。脚本核对两个构建来源、校验清单和更新签名，建立一个草稿并逐项比较 GitHub 资产摘要。核对草稿后，以相同参数加 `--publish` 一次公开全部文件。脚本可续传原草稿。
4. 将 CLI 工作流产物里的 `Casks/repolens.rb`、`bucket/repolens.json` 及本机生成的 `repolens-desktop.rb` 一起提交到 `ConteMan/homebrew-tap` 对应目录。包管理器清单只指向已公开的资源；tap 推送失败可独立重试。
5. 从公开 Release 下载产物，重验两份校验和，确认 GitHub Latest、两个 Homebrew 包、Scoop 和两个 MyGo 清单都指向同一版本。
