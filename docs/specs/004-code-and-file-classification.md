# 004: 文件分类与代码渲染（internal/render）

- 状态：已确认
- 关联：roadmap M2、ADR-003

## 问题

内容集合中的每个文件需要被分类，决定它在浏览层的形态；代码/文本文件需要构建期高亮。

## 行为

1. **分类**：`Classify(path string, size int64, sniff []byte) Kind`，Kind ∈ `Markdown` / `HTML` / `Code`（含纯文本）/ `Image` / `Binary`。判定顺序：
   - 扩展名 `.md` / `.markdown` → Markdown；`.html` / `.htm` → HTML；
   - 常见图片扩展（png/jpg/jpeg/gif/svg/webp/avif/ico）→ Image；
   - 其余按 chroma lexer 注册表 ＋ 文本嗅探（`sniff` 前 512B 无 NUL 且可判为文本）→ Code；否则 → Binary。
   - 超过 `MaxFileSize` 的文件一律降级为 Binary（不读内容）。
2. **代码高亮**：chroma（`github.com/alecthomas/chroma/v2`）HTML formatter：
   - `WithClasses(true)`——输出 class 而非内联样式，样式表由 theme 层用 `chroma.NewStyle`/`styles.Get(opts.Theme)` 生成一次 CSS（提供 `StylesCSS(theme string) (string, error)` 助手）；
   - 行号由 `line_numbers` 控制，行号可链接（`#L42` 锚点，formatter 的 LinkableLineNumbers）；
   - lexer 按文件名匹配，失败退化为 plaintext；
   - 与 spec 003 的 Markdown 代码块共用同一 chroma 配置，保证站内高亮观感一致。
3. **Image / Binary**：不产出正文 HTML；site 层据 Kind 生成图片预览页（`<img>` 指向镜像路径）或下载页（文件名、大小、镜像链接）。

## 接口契约

```go
package render

type Kind int
const (KindMarkdown Kind = iota; KindHTML; KindCode; KindImage; KindBinary)

func Classify(path string, size int64, sniff []byte, maxFileSize int64) Kind

type CodeOptions struct{ LineNumbers bool; Theme string }

type CodeResult struct {
    HTML  template.HTML
    Lang  string // 识别出的语言名，plaintext 为 "text"
    Lines int
}

func Code(src []byte, filename string, opts CodeOptions) (CodeResult, error)
func StylesCSS(theme string) (string, error) // chroma 样式表，theme 层调用
```

## 边界与非目标

- 不做语言的启发式内容识别（shebang 之外不猜），扩展名优先；
- 不做超大文件的分页/截断渲染（超限即 Binary，v1 简单化）；
- SVG 归为 Image（镜像层原样、预览页 `<img>` 引用），不内联进页面。

## 验收

- 分类表驱动测试：五类各有正反例，含"无扩展名文本文件"“NUL 二进制”“超限降级”；
- 高亮 golden 测试：Go / JS / YAML 三个样例，class 输出、行号锚点存在；未知扩展退化 plaintext 不报错；
- `StylesCSS` 对内置主题名返回非空 CSS，未知主题返回错误；
- `gofmt` / `go vet` / `go test` 通过。
