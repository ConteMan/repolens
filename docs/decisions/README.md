# 架构决策记录（ADR）

记录已定且不可轻易推翻的决策。推翻已有 ADR 必须新增一条 ADR 声明取代关系并说明理由，禁止静默修改。

新增决策复制 [template.md](template.md)，编号递增。

| 编号 | 标题 | 状态 |
|---|---|---|
| [ADR-001](ADR-001-two-layer-output.md) | 双层输出：镜像层 ＋ 浏览层 | 已接受 |
| [ADR-002](ADR-002-prerendered-mpa.md) | 浏览层采用预渲染 MPA ＋ 薄增强层 | 已接受 |
| [ADR-003](ADR-003-tech-stack.md) | 技术栈：Go 单二进制与依赖选型标准 | 已接受 |
| [ADR-004](ADR-004-system-git.md) | Git 层 shell out 系统 git 而非 go-git | 已接受 |
| [ADR-005](ADR-005-config-trust-domains.md) | 配置双信任域与级联规则语义 | 已接受 |
| [ADR-006](ADR-006-react-base-ui-admin.md) | 本地管理界面采用 React、TypeScript 与 Base UI | 已接受 |
