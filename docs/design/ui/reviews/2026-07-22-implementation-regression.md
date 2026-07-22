# 2026-07-22 UI 实现回归

## 结论

Issue #30 的 React、TypeScript 与 Base UI 实现已覆盖冻结的方向 B 工作区、1440px / 1024px 桌面布局和 390px 窄屏路径。配置分区持续展示有效配置来源与最近成功构建上下文；现有配置 API、仓库信任域、revision 乐观并发与原子写入合同保持不变。本轮未发现需要另建 `design-gap` Issue 的剩余差异。

本结论来自实现代码审查、真实临时 Git 仓库的浏览器回归和冻结快照的启发式视觉对照，不替代后续 3～5 名目标用户的任务验证。

## 节点与实现证据

| 冻结画板 | 节点 ID | 实现覆盖 | 导出证据 |
|---|---|---|---|
| `02 Project Open` | `mQGrN` | 绝对路径输入、打开、加载、warning 与失败反馈 | [PNG](2026-07-22-final-baseline-assets/02-project-open-mQGrN.png) |
| `03 Config Edit` | `oMxTz` | 分区导航、字段来源、dirty、字段错误与有序规则 | [PNG](2026-07-22-final-baseline-assets/03-config-edit-oMxTz.png) |
| `04 Diff and Write` | `bXQ2L` | diff、取消、写入成功与 revision 冲突后重新读取 | [PNG](2026-07-22-final-baseline-assets/04-diff-write-bXQ2L.png) |
| `05 Build` | `I9boV` | 运行、成功、warning、失败及保留最近成功结果 | [PNG](2026-07-22-final-baseline-assets/05-build-I9boV.png) |
| `06 Narrow / Project and Config` | `mvZOJ` | 390px 横向分区导航与 Rule 独立编辑页 | [PNG](2026-07-22-final-baseline-assets/06-narrow-project-config-mvZOJ.png) |
| `07 Narrow / Diff and Build` | `cgNoV` | 390px diff 内部横向滚动、冲突恢复与 Build 状态 | [PNG](2026-07-22-final-baseline-assets/07-narrow-diff-build-cgNoV.png) |

Foundations `RRkxu` 与 Core Components `sGnOn` 已映射为语义颜色、间距、圆角、焦点样式，以及项目自有 Button、Badge、Alert、Header、Sidebar 和 Sticky Actions；所有资源仍由单二进制同源内嵌，不加载外部字体、脚本或 CDN。

## 自动化回归

Playwright 使用每条测试独立创建的真实临时 Git 仓库，覆盖：

1. 相对路径打开失败时保留输入，并把邻近错误、ARIA 关联和焦点绑定到路径字段；
2. 打开仓库、编辑、校验并生成 diff；
3. 三类字段错误、ARIA 关联、跨分区定位与焦点恢复；
4. revision 冲突后重新读取，且不覆盖外部修改；
5. 390px 下打开 warning、纵向 Build Stats、构建成功、构建失败并保留最近成功输出；
6. 1440px 与 1024px 持续上下文可见且页面无横向溢出；
7. 390px Rule 子页、diff 内部横向滚动、Escape 焦点恢复与 sticky action 可达。

`pnpm check` 与 `pnpm test:e2e` 均通过。完整 Go 质量门禁与 CI 结果记录在关联 PR。

## 未决验证

- 真实目标用户任务验证尚未执行，应在后续独立 Issue 中跟踪；它不改变本次冻结设计与实现的一致性结论。
