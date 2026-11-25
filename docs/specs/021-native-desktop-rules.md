# 021: 原生桌面端打开仓库与保存规则

- 状态：已实现
- 关联：[Roadmap](../roadmap.md)、[ADR-004](../decisions/ADR-004-system-git.md)、[ADR-005](../decisions/ADR-005-config-trust-domains.md)、[ADR-006](../decisions/ADR-006-react-base-ui-admin.md)、[Spec 013](013-config-ui.md)、[原生桌面设计评审](../design/ui/reviews/desktop-native/README.md)

## 问题

桌面用户需要通过系统目录选择器打开本地工作树并编辑渲染规则。第一阶段用一次完整的打开、编辑、确认、保存和重开验证 MyGo 与现有 Go 配置合同可以配合工作。

## 行为

1. 从空窗口或已有仓库打开系统目录选择器。选择本地 Git 工作树根目录后读取根目录的 `.repolens.yml`。缺少配置文件时进入空规则列表。普通目录、仓库子目录、裸仓库和无法读取的配置应给出可恢复错误。取消选择时返回当前页面。切换仓库失败时保留当前仓库和草稿。有草稿时，切换成功前先明确确认是否丢弃。
2. 在「路径与规则」页编辑有序 `rules`，支持新增、修改、删除和调序。编辑器使用 `RepositoryRuleSettings` 已支持的 `match`、`render`、`markdown`、`html`、`code`、`max_file_size`。可选布尔值保留继承、显式开启、显式关闭三态。列表顺序决定级联顺序，后匹配规则覆盖先匹配规则的已设置字段。规则展示显式值与继承状态，具体文件路径的有效值由构建管线按路径计算。
3. 「保存设置」先校验，定位首个错误并保留草稿。校验通过后展示目标文件与差异，确认前只在内存中准备候选。取消确认后继续编辑。确认保存后显示实际目标与成功状态，并以已保存的规则和最新 revision 为新的编辑基准。写入失败或 revision 冲突时保留草稿，显示原因和恢复入口。

## 接口契约

| 边界 | 合同 |
|---|---|
| 原生界面 | 采用 `github.com/egoist/mygo v0.10.0`。窗口与控件使用 MyGo/`github.com/egoist/mygo/ui`，目录选择使用 `mygo.Dialog.Open` 和 `OpenDialogOptions{Directory: true}`。桌面与 CLI 共用根模块及 Go 1.27.4。 |
| 工作树定位 | 沿用系统 Git，验证目录处于非裸工作树内并与 `git rev-parse --show-toplevel` 返回的根目录一致，比较时处理路径别名。Spec 013 的 Web 打开入口目前只校验绝对目录路径。 |
| 读取与草稿 | 使用 `config.LoadRepositoryDocument` 获取仓库显式设置和 revision。草稿只替换 `RepositorySettings.Rules`，其他受控字段取自同一快照。规则列表的缺省和显式空值沿用仓库文档模型。 |
| 校验与预览 | 复用 `config.ValidateRepositorySettings`、`RepositoryDocument.YAML` 和 Spec 013 的可信字段差异脱敏。`RepositoryDocument.Replace` 会重建所有受控设置，桌面端应由 `internal/config` 提供仅替换 `rules` 的更新入口，复用内部规则替换逻辑。差异以当前工作区 revision 为基准。候选保留已有 `source`、`output`、`access` 及其他配置的语义。`rules` 内的 YAML 注释和未知字段沿用现有文档接口的保留约定。 |
| 持久化 | 确认绑定仓库、草稿与准备差异时的 revision。确认后重新读取并比较 revision，再由 `RepositoryDocument.Write` 执行写前检查和同目录原子替换。`config.ErrRevisionConflict` 转为可恢复的界面状态。同一确认只接受一次在途写入。 |

## 边界与待决策项

技术选型见 [ADR-008](../decisions/ADR-008-native-desktop.md)，首期在 macOS 验收。本阶段视觉基线见桌面评审目录，涵盖 CMP-01、REPO-01 至 REPO-03、SITE-04、SITE-07 至 SITE-13。仅展示首期已接入的操作。

站点其他设置、`ignore` 编辑、外部可信配置、构建与预览、远程仓库、最近仓库、偏好持久化、自动更新、签名公证与发布均留待后续规格。本阶段的唯一写入目标是选定工作树根目录下的 `.repolens.yml`。

## 验收

- macOS 实机通过系统目录选择器打开 Git fixture。覆盖首次打开失败、切换失败、取消、含空格的工作树路径和无配置文件的仓库。失败或取消后验证原有草稿与导航位置。
- 使用两个重叠匹配规则验证新增、修改、删除和调序。确认前磁盘内容一致，确认后重开文件核对顺序、继承与显式 `false`，并用 `config.Load` 验证级联结果。包含其他站点设置及可信字段的 fixture 在保存后保留原有语义，差异预览隐藏可信字段。
- 验证非法 glob、取消保存、外部修改导致的 revision 冲突与写入失败。冲突或失败时保留草稿与磁盘上的现有内容。
- 实现 PR 按冻结的设计节点记录页面、状态、窗口尺寸与截图。通过 `gofmt -l .`、`go vet ./...`、`go test ./...` 和 `go build ./...`，并按 ADR 确定的方式验证 macOS 桌面产物。

首次 macOS 选择器与 Linux 质量门禁验收记录见 [CI 45594905844](https://github.com/ConteMan/repolens/actions/runs/45594905844)。
