# ADR-003: 技术栈：Go 单二进制与依赖选型标准

- 状态：已接受（本地管理界面前端部分已被 ADR-006 取代）
- 日期：2026-07-04

## 背景

维护者偏好 Go；非商业开源项目要求长期低成本维护，依赖必须主流且在维护。

## 决策

Go 单二进制，模板与静态资源全部 go:embed。选型锚点为 Hugo 验证过的生态：

| 组件 | 选型 |
|---|---|
| Markdown | yuin/goldmark ＋ abhinav 系扩展（toc / anchor / frontmatter / mermaid） |
| 代码高亮 | alecthomas/chroma（构建期） |
| CLI | spf13/cobra |
| YAML | goccy/go-yaml |
| 文件监听 | fsnotify/fsnotify |
| 模板 | 标准库 html/template |
| 生成站点增强层 | 手写原生 JS / CSS |
| 本地管理界面 | React + TypeScript + Base UI + Vite（见 ADR-006） |

## 理由

- goldmark / chroma 即 Hugo 渲染层，是"主流且在维护"的最强背书；goldmark 的 AST ＋ 扩展架构与"规则 = 管线"的设计对口，链接改写实现为 ASTTransformer；
- YAML 库选 goccy/go-yaml 而非 gopkg.in/yaml.v3：后者官方仓库 2025 年已归档；
- 生成站点增强层不引入框架与打包器；本地管理界面的需求已超出本决策当时“数百行薄增强层”的前提，按 ADR-006 独立采用 Node 构建工具链；
- 只参照 docu.md 交互形态、不复用其代码（GPLv3），项目可自由采用 MIT。

## 后果

- 直接依赖控制在个位数；新增依赖需按本 ADR 标准论证；
- 全文搜索（v2）预定 pagefind（独立二进制、静态后处理），不影响现有文件名与标题级搜索设计；
- 客户端加密（可选功能）按 PageCrypt 思路自实现（构建期加密 ＋ WebCrypto 解密），不引依赖。
