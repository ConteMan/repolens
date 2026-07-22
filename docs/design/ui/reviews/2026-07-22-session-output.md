# 2026-07-22 UI 会话级输出设计基线评审

## 结论

通过，作为 Spec 014 与 Issue #46 的实现基线。

本次在 `docs/design/ui/prototypes/repolens-ui.pen` 增补 3 个顶层画板，覆盖桌面自定义输出、已有 repolens 产物的明确覆盖确认和 390px 路径校验失败。新增状态复用既有 Header、Sidebar、Button、Field 与 Build Panel 组件；没有把会话级输出伪装成仓库 `output` 配置，也没有加入目录浏览器、部署或托管入口。

## 冻结合同

- Build 页以“缓存目录 / 自定义目录”分段控件选择本次会话输出；默认仍为缓存目录。
- 自定义绝对路径靠近字段展示结构化错误；错误时禁用构建，并可一键恢复缓存模式。
- 带 `.repolens-build` 哨兵的旧产物必须弹出明确确认；确认文案说明同级临时构建、发布前所有权复检和原子替换。
- 未知非空目录、文件路径和与仓库存在危险包含关系的目录不提供强制覆盖入口。
- 切换项目后恢复缓存模式；路径不进入 `.repolens.yml`、YAML diff 或浏览器持久化。

## 节点与快照

| 画板 | 节点 ID | 视口与状态 | 快照 |
|---|---|---|---|
| `08 Build / Custom Output` | `OF3Ud` | 1440px，自定义绝对目录可构建并展示最终路径 | [PNG](2026-07-22-session-output-assets/08-build-custom-output-OF3Ud.png) |
| `09 Build / Confirm Overwrite` | `k7Wma8` | 1440px，检测到 repolens 哨兵后的覆盖确认 | [PNG](2026-07-22-session-output-assets/09-build-confirm-overwrite-k7Wma8.png) |
| `10 Narrow / Custom Output` | `SNX4i` | 390px，仓库内输出路径被拒绝并绑定字段错误 | [PNG](2026-07-22-session-output-assets/10-narrow-custom-output-SNX4i.png) |

## 验证记录

- 三个新增顶层节点分别执行 `snapshot_layout(problemsOnly: true)`，均返回无布局问题。
- 顶层坐标复核通过：`07` 与 `08`、`08` 与 `09`、`09` 与 `10` 均保留 120px 画布间距，没有画板重叠。
- 三张 2x PNG 均非空；长绝对路径、错误码、覆盖说明和按钮文案没有裁切或页面级横向溢出。
- 桌面画板保留 Build 导航、结果统计与会话边界；窄屏画板使用同一信息层级，不依赖桌面侧栏。
- 使用 Pen 桌面窗口执行 File → Save，标题由 `Edited` 恢复为干净状态；关闭目标窗口并从 Recent 重开后，MCP 仍能定位 11 个顶层节点与 `OF3Ud`、`k7Wma8`、`SNX4i`。
- `.pen` 是唯一可编辑视觉事实源；PNG 只用于评审与 PR 差异浏览。

## 实现映射

1. `POST /api/build` 接受可选 `output_path` 与 `confirm_overwrite`，错误映射回输出路径字段或覆盖确认对话框。
2. Build 页面维护会话级输出模式与路径；切换项目重置，构建进行中禁用输出控件。
3. 后端在同级临时目录生成并在发布前复检目标所有权；失败不破坏最近成功产物。
4. 以本文件节点和 PNG 做 1440px / 390px 回归，并验证键盘焦点进入和退出覆盖确认对话框。
