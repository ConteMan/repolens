# ADR-006: 本地管理界面采用 React、TypeScript 与 Base UI

- 状态：已接受
- 日期：2026-07-11
- 取代：ADR-003 中“所有前端均手写原生 JS/CSS、永不引入 Node 工具链”的部分决策

## 背景

`repolens ui` 已从少量增强脚本发展为完整本地应用：结构化配置表单、有序规则编辑、三态布尔值、校验与字段错误、YAML diff、revision 冲突、确认写入、异步构建轮询和多种失败状态。继续在单个 HTML 文件中手写 DOM、状态和可访问交互，维护与测试成本已高于引入专用前端工具链的成本。

生成站点的浏览层仍是预渲染 MPA，其约束与本地管理界面不同；两者不应被同一技术选择强行绑定。

## 决策

`repolens ui` 采用 React、TypeScript、Vite 与 Base UI：

- React/TypeScript 负责状态、页面和业务组件；
- Base UI 提供无样式、可组合且可访问的交互原语；
- 样式由 repolens 自有 CSS variables 与普通 CSS 定义，不依赖外部 CDN；
- pnpm 是唯一 Node 包管理器，lockfile 必须提交；
- Vite 产物输出到 `internal/ui/dist/`，由 Go `go:embed` 编入单二进制；
- 最终用户运行、构建或发布站点时不需要 Node；Node 仅用于仓库开发和发布构建；
- Go API、loopback、CSRF、信任域和缓存构建合同保持不变。

ADR-002 继续有效：repolens 生成的静态浏览站仍采用预渲染 MPA 与原生薄增强层，不加载 React。

## 理由

- Base UI 专注键盘、焦点、弹层与 ARIA 行为，适合配置工具中的 Select、Dialog、Tabs、Collapsible 和错误恢复；
- React 组件与状态模型能把规则编辑、表单草稿、diff 确认和构建 operation 从手工 DOM 操作中分离；
- TypeScript 可让前端 payload 与 Go 的仓库配置 DTO 更容易保持一致；
- Vite 能产出纯静态、可 embed 的哈希资源，Node 不进入产品运行时；
- Vue 同样可满足状态管理，但 Base UI 是 React 专用库；选择 Base UI 后 React 是最直接且维护面最小的组合。

## 后果

- 开发环境新增 Node 24 与 pnpm 10；CI 在 Go 门禁前执行 `pnpm install --frozen-lockfile`、TypeScript 严格检查、前端单元测试、Playwright 浏览器冒烟和生产构建；
- `internal/ui/dist/` 作为 Go 构建输入提交仓库，保证仅安装 Go 的用户仍可 `go build`；CI 必须验证 dist 与源码同步；
- 前端依赖不计入 Go 直接依赖个位数约束，但新增或升级仍需论证并接受依赖审计；
- 禁止运行时外部请求，字体、脚本和样式必须进入 embed 产物；
- 原生 `internal/ui/assets/index.html` 在迁移完成后删除，不保留双实现。
