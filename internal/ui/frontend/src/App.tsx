import { AlertDialog } from "@base-ui/react/alert-dialog";
import { useEffect, useMemo, useState } from "react";
import { api, APIError } from "./api";
import { Field, NativeSelect, Section, Status, TextArea, TextInput, TriState } from "./components";
import { emptyFileOptions, normalizeSettings, type BuildResponse, type PrepareResponse, type RepositorySettings, type Rule } from "./types";

type Notice = { kind: "info" | "success" | "warning" | "error"; message: string } | null;

const optionalText = (value: string) => (value.trim() ? value.trim() : null);
const optionalNumber = (value: string) => (value === "" ? null : Number(value));
const formatVars = (vars: Record<string, string> | null) =>
  vars ? Object.entries(vars).map(([key, value]) => `${key}: ${value}`).join("\n") : "";
const parseVars = (value: string) => {
  const entries = value
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const separator = line.indexOf(":");
      return separator > 0 ? [line.slice(0, separator).trim(), line.slice(separator + 1).trim()] : null;
    })
    .filter((entry): entry is [string, string] => Boolean(entry?.[0]));
  return entries.length ? Object.fromEntries(entries) : null;
};

function RuleEditor({
  rule,
  index,
  count,
  onChange,
  onMove,
  onRemove,
}: {
  rule: Rule;
  index: number;
  count: number;
  onChange: (rule: Rule) => void;
  onMove: (offset: number) => void;
  onRemove: () => void;
}) {
  return (
    <article className="rule-card">
      <header className="rule-header">
        <div>
          <span className="eyebrow">规则 {index + 1}</span>
          <strong>{rule.match || "尚未设置匹配模式"}</strong>
        </div>
        <div className="button-row compact">
          <button type="button" className="button button-ghost" disabled={index === 0} onClick={() => onMove(-1)}>上移</button>
          <button type="button" className="button button-ghost" disabled={index === count - 1} onClick={() => onMove(1)}>下移</button>
          <button type="button" className="button button-danger" onClick={onRemove}>删除</button>
        </div>
      </header>
      <div className="field-grid three">
        <Field label="匹配模式">
          <TextInput value={rule.match ?? ""} placeholder="docs/**" onChange={(e) => onChange({ ...rule, match: optionalText(e.target.value) })} />
        </Field>
        <Field label="生成阅读页"><TriState value={rule.render} onChange={(render) => onChange({ ...rule, render })} /></Field>
        <Field label="最大文件大小（字节）">
          <TextInput type="number" min="0" value={rule.max_file_size ?? ""} onChange={(e) => onChange({ ...rule, max_file_size: optionalNumber(e.target.value) })} />
        </Field>
        <Field label="Markdown 数学公式"><TriState value={rule.markdown.math} onChange={(math) => onChange({ ...rule, markdown: { ...rule.markdown, math } })} /></Field>
        <Field label="Markdown 目录"><TriState value={rule.markdown.toc} onChange={(toc) => onChange({ ...rule, markdown: { ...rule.markdown, toc } })} /></Field>
        <Field label="目录最少标题数">
          <TextInput type="number" min="0" value={rule.markdown.toc_min_headings ?? ""} onChange={(e) => onChange({ ...rule, markdown: { ...rule.markdown, toc_min_headings: optionalNumber(e.target.value) } })} />
        </Field>
        <Field label="标题锚点"><TriState value={rule.markdown.anchors} onChange={(anchors) => onChange({ ...rule, markdown: { ...rule.markdown, anchors } })} /></Field>
        <Field label="Mermaid"><TriState value={rule.markdown.mermaid} onChange={(mermaid) => onChange({ ...rule, markdown: { ...rule.markdown, mermaid } })} /></Field>
        <Field label="Frontmatter 标题"><TriState value={rule.markdown.frontmatter_title} onChange={(frontmatter_title) => onChange({ ...rule, markdown: { ...rule.markdown, frontmatter_title } })} /></Field>
        <Field label="HTML 展示方式">
          <NativeSelect value={rule.html.view ?? ""} onChange={(e) => onChange({ ...rule, html: { view: optionalText(e.target.value) } })}>
            <option value="">使用默认值</option><option value="embed">嵌入</option><option value="direct">直接链接</option><option value="source">源码</option>
          </NativeSelect>
        </Field>
        <Field label="代码行号"><TriState value={rule.code.line_numbers} onChange={(line_numbers) => onChange({ ...rule, code: { ...rule.code, line_numbers } })} /></Field>
        <Field label="代码主题"><TextInput value={rule.code.theme ?? ""} onChange={(e) => onChange({ ...rule, code: { ...rule.code, theme: optionalText(e.target.value) } })} /></Field>
      </div>
    </article>
  );
}

export function App() {
  const [path, setPath] = useState("");
  const [activePath, setActivePath] = useState("");
  const [revision, setRevision] = useState("");
  const [settings, setSettings] = useState<RepositorySettings | null>(null);
  const [savedSettings, setSavedSettings] = useState<RepositorySettings | null>(null);
  const [notice, setNotice] = useState<Notice>(null);
  const [busy, setBusy] = useState(false);
  const [pending, setPending] = useState<PrepareResponse | null>(null);
  const [build, setBuild] = useState<BuildResponse | null>(null);
  const [lastSuccess, setLastSuccess] = useState<BuildResponse | null>(null);

  const dirty = useMemo(
    () => Boolean(settings && savedSettings && JSON.stringify(settings) !== JSON.stringify(savedSettings)),
    [settings, savedSettings],
  );

  const showError = (error: unknown, fallback: string) => {
    const message = error instanceof APIError ? error.message : fallback;
    setNotice({ kind: "error", message });
  };

  const openProject = async () => {
    if (!path.trim()) return setNotice({ kind: "error", message: "请输入本地 Git 工作树的绝对路径。" });
    setBusy(true); setNotice({ kind: "info", message: "正在读取仓库配置…" }); setBuild(null);
    try {
      const response = await api.open(path.trim());
      const next = normalizeSettings(response.settings);
      setActivePath(path.trim()); setRevision(response.revision); setSettings(next); setSavedSettings(next); setLastSuccess(null);
      setNotice({ kind: "success", message: "配置已加载。修改后请先校验并预览 diff。" });
    } catch (error) { showError(error, "无法打开项目。"); } finally { setBusy(false); }
  };

  const validate = async () => {
    if (!settings) return;
    setBusy(true); setNotice({ kind: "info", message: "正在校验配置…" });
    try {
      const response = await api.validate(activePath, settings, revision);
      setSettings(normalizeSettings(response.settings));
      setNotice({ kind: "success", message: "配置校验通过。" });
    } catch (error) { showError(error, "配置校验失败。"); } finally { setBusy(false); }
  };

  const prepare = async () => {
    if (!settings) return;
    setBusy(true); setNotice({ kind: "info", message: "正在生成 YAML diff…" });
    try {
      const response = await api.prepare(activePath, settings, revision);
      if (!response.diff) {
        setSettings(normalizeSettings(response.settings)); setSavedSettings(normalizeSettings(response.settings));
        setNotice({ kind: "success", message: "没有需要写入的配置变更。" });
      } else {
        setPending(response); setNotice(null);
      }
    } catch (error) { showError(error, "无法生成写入预览。"); } finally { setBusy(false); }
  };

  const commit = async () => {
    if (!pending) return;
    setBusy(true);
    try {
      const response = await api.commit(activePath, normalizeSettings(pending.settings), pending.revision);
      const next = normalizeSettings(response.settings);
      setRevision(response.revision); setSettings(next); setSavedSettings(next); setPending(null);
      setNotice({ kind: "success", message: "配置已原子写入，现在可以开始构建。" });
    } catch (error) { showError(error, "写入配置失败。"); } finally { setBusy(false); }
  };

  const startBuild = async () => {
    setBusy(true); setNotice({ kind: "info", message: "正在启动构建…" });
    try {
      const operation = await api.startBuild(activePath);
      setBuild(operation); setNotice(null);
    } catch (error) { showError(error, "无法启动构建。"); setBusy(false); }
  };

  useEffect(() => {
    if (!build?.id || build.stage === "completed" || build.stage === "failed") return;
    const timer = window.setTimeout(async () => {
      try {
        const next = await api.getBuild(build.id);
        setBuild(next);
        if (next.stage === "completed") { setLastSuccess(next); setNotice({ kind: "success", message: "构建完成。输出已保存在本机缓存目录。" }); setBusy(false); }
        if (next.stage === "failed") { setNotice({ kind: "error", message: next.error || "构建失败。" }); setBusy(false); }
      } catch (error) { showError(error, "读取构建状态失败。"); setBusy(false); }
    }, 500);
    return () => window.clearTimeout(timer);
  }, [build]);

  const updateRule = (index: number, rule: Rule) => setSettings((current) => current ? ({ ...current, rules: (current.rules ?? []).map((item, i) => i === index ? rule : item) }) : current);
  const moveRule = (index: number, offset: number) => setSettings((current) => {
    if (!current) return current;
    const rules = [...(current.rules ?? [])]; const target = index + offset;
    if (target < 0 || target >= rules.length) return current;
    [rules[index], rules[target]] = [rules[target], rules[index]];
    return { ...current, rules };
  });

  return (
    <div className="app-shell">
      <header className="topbar">
        <div><strong>repolens ui</strong><span>本地仓库配置与构建</span></div>
        <code title={activePath}>{activePath || "未打开项目"}</code>
      </header>
      <main className="page">
        <section className="open-panel">
          <div><span className="eyebrow">本地工作树</span><h1>安全地配置并构建仓库</h1><p>输入绝对路径。仓库内容只在本机读取，构建结果写入仓库外缓存。</p></div>
          <div className="path-row"><TextInput aria-label="仓库绝对路径" value={path} onChange={(e) => setPath(e.target.value)} placeholder="/Users/name/Projects/repository" disabled={busy} /><button className="button button-primary" onClick={openProject} disabled={busy}>打开项目</button></div>
        </section>

        {notice ? <Status kind={notice.kind}>{notice.message}</Status> : null}

        {settings ? (
          <>
            <div className="workspace-heading"><div><span className="eyebrow">仓库配置</span><h2>编辑 `.repolens.yml`</h2></div><span className={`save-state ${dirty ? "dirty" : ""}`}>{dirty ? "有未保存修改" : "已保存"}</span></div>
            <Status kind="warning">仅编辑仓库信任域；`source`、`output`、`access` 不会出现在页面或写入 payload。</Status>
            <div className="layout-grid">
              <div className="main-column">
                <Section title="站点" description="页面标题、语言与默认首页。"><div className="field-grid three">
                  <Field label="标题"><TextInput value={settings.site.title ?? ""} onChange={(e) => setSettings({ ...settings, site: { ...settings.site, title: optionalText(e.target.value) } })} /></Field>
                  <Field label="语言"><TextInput value={settings.site.language ?? ""} placeholder="zh-CN" onChange={(e) => setSettings({ ...settings, site: { ...settings.site, language: optionalText(e.target.value) } })} /></Field>
                  <Field label="首页文件"><TextInput value={settings.site.home ?? ""} placeholder="README.md" onChange={(e) => setSettings({ ...settings, site: { ...settings.site, home: optionalText(e.target.value) } })} /></Field>
                </div></Section>
                <Section title="忽略路径" description="每行一个 Git glob；未编辑的空列表保持未设置。"><TextArea rows={5} value={(settings.ignore ?? []).join("\n")} onChange={(e) => setSettings({ ...settings, ignore: e.target.value.split("\n").map((line) => line.trim()).filter(Boolean) })} /></Section>
                <Section title="全局渲染" description="作为所有路径规则之前的第 0 条默认规则。"><div className="field-grid three">
                  <Field label="生成阅读页"><TriState value={settings.render.render} onChange={(render) => setSettings({ ...settings, render: { ...settings.render, render } })} /></Field>
                  <Field label="目录"><TriState value={settings.render.markdown.toc} onChange={(toc) => setSettings({ ...settings, render: { ...settings.render, markdown: { ...settings.render.markdown, toc } } })} /></Field>
                  <Field label="目录最少标题数"><TextInput type="number" min="0" value={settings.render.markdown.toc_min_headings ?? ""} onChange={(e) => setSettings({ ...settings, render: { ...settings.render, markdown: { ...settings.render.markdown, toc_min_headings: optionalNumber(e.target.value) } } })} /></Field>
                  <Field label="标题锚点"><TriState value={settings.render.markdown.anchors} onChange={(anchors) => setSettings({ ...settings, render: { ...settings.render, markdown: { ...settings.render.markdown, anchors } } })} /></Field>
                  <Field label="Mermaid"><TriState value={settings.render.markdown.mermaid} onChange={(mermaid) => setSettings({ ...settings, render: { ...settings.render, markdown: { ...settings.render.markdown, mermaid } } })} /></Field>
                  <Field label="数学公式"><TriState value={settings.render.markdown.math} onChange={(math) => setSettings({ ...settings, render: { ...settings.render, markdown: { ...settings.render.markdown, math } } })} /></Field>
                  <Field label="Frontmatter 标题"><TriState value={settings.render.markdown.frontmatter_title} onChange={(frontmatter_title) => setSettings({ ...settings, render: { ...settings.render, markdown: { ...settings.render.markdown, frontmatter_title } } })} /></Field>
                  <Field label="HTML 展示"><NativeSelect value={settings.render.html.view ?? ""} onChange={(e) => setSettings({ ...settings, render: { ...settings.render, html: { view: optionalText(e.target.value) } } })}><option value="">使用默认值</option><option value="embed">嵌入</option><option value="direct">直接链接</option><option value="source">源码</option></NativeSelect></Field>
                  <Field label="代码行号"><TriState value={settings.render.code.line_numbers} onChange={(line_numbers) => setSettings({ ...settings, render: { ...settings.render, code: { ...settings.render.code, line_numbers } } })} /></Field>
                  <Field label="代码主题"><TextInput value={settings.render.code.theme ?? ""} onChange={(e) => setSettings({ ...settings, render: { ...settings.render, code: { ...settings.render.code, theme: optionalText(e.target.value) } } })} /></Field>
                  <Field label="最大文件大小（字节）"><TextInput type="number" min="0" value={settings.render.max_file_size ?? ""} onChange={(e) => setSettings({ ...settings, render: { ...settings.render, max_file_size: optionalNumber(e.target.value) } })} /></Field>
                </div></Section>
                <Section title="路径规则" description="按顺序全部匹配，后写覆盖先写；未编辑的空列表保持未设置。"><div className="rule-list">{(settings.rules ?? []).map((rule, index) => <RuleEditor key={index} rule={rule} index={index} count={settings.rules?.length ?? 0} onChange={(next) => updateRule(index, next)} onMove={(offset) => moveRule(index, offset)} onRemove={() => setSettings({ ...settings, rules: (settings.rules ?? []).filter((_, i) => i !== index) })} />)}</div><button type="button" className="button button-secondary" onClick={() => setSettings({ ...settings, rules: [...(settings.rules ?? []), { ...emptyFileOptions(), match: null }] })}>新增规则</button></Section>
              </div>
              <aside className="side-column">
                <Section title="主题"><div className="field-stack">
                  <Field label="变量" hint="每行 `名称: 值`"><TextArea rows={6} value={formatVars(settings.theme.vars)} onChange={(e) => setSettings({ ...settings, theme: { ...settings.theme, vars: parseVars(e.target.value) } })} /></Field>
                  <Field label="附加 CSS"><TextInput value={settings.theme.css ?? ""} onChange={(e) => setSettings({ ...settings, theme: { ...settings.theme, css: optionalText(e.target.value) } })} /></Field>
                  <Field label="模板目录"><TextInput value={settings.theme.templates ?? ""} onChange={(e) => setSettings({ ...settings, theme: { ...settings.theme, templates: optionalText(e.target.value) } })} /></Field>
                </div></Section>
                <Section title="浏览"><div className="field-stack">
                  <Field label="文件树位置"><NativeSelect value={settings.view.tree_position ?? ""} onChange={(e) => setSettings({ ...settings, view: { ...settings.view, tree_position: optionalText(e.target.value) } })}><option value="">使用默认值</option><option value="left">左侧</option><option value="right">右侧</option></NativeSelect></Field>
                  <Field label="展开层级"><TextInput type="number" min="0" value={settings.view.tree_expand_depth ?? ""} onChange={(e) => setSettings({ ...settings, view: { ...settings.view, tree_expand_depth: optionalNumber(e.target.value) } })} /></Field>
                  <Field label="目录面板"><NativeSelect value={settings.view.toc_panel ?? ""} onChange={(e) => setSettings({ ...settings, view: { ...settings.view, toc_panel: optionalText(e.target.value) } })}><option value="">使用默认值</option><option value="floating">浮动</option><option value="inline">内联</option></NativeSelect></Field>
                  <Field label="站内搜索"><TriState value={settings.view.search} onChange={(search) => setSettings({ ...settings, view: { ...settings.view, search } })} /></Field>
                </div></Section>
                <Section title="Agent 视图"><div className="field-stack">
                  <Field label="生成 llms.txt"><TriState value={settings.agent.llms_txt} onChange={(llms_txt) => setSettings({ ...settings, agent: { ...settings.agent, llms_txt } })} /></Field>
                  <Field label="生成 llms-full.txt"><TriState value={settings.agent.llms_full.enabled} onChange={(enabled) => setSettings({ ...settings, agent: { ...settings.agent, llms_full: { ...settings.agent.llms_full, enabled } } })} /></Field>
                  <Field label="llms-full 最大字节数"><TextInput type="number" min="0" value={settings.agent.llms_full.max_size ?? ""} onChange={(e) => setSettings({ ...settings, agent: { ...settings.agent, llms_full: { ...settings.agent.llms_full, max_size: optionalNumber(e.target.value) } } })} /></Field>
                  <Field label="生成 index.json"><TriState value={settings.agent.index_json} onChange={(index_json) => setSettings({ ...settings, agent: { ...settings.agent, index_json } })} /></Field>
                </div></Section>
                <Section title="构建结果"><div className="build-state"><strong>{build ? build.stage : "尚未构建"}</strong>{build?.stats ? <dl><div><dt>文件</dt><dd>{build.stats.Files}</dd></div><div><dt>页面</dt><dd>{build.stats.Pages}</dd></div><div><dt>耗时</dt><dd>{Math.round(build.stats.Duration / 1_000_000)}ms</dd></div></dl> : null}{build?.output_path ? <code>{build.output_path}</code> : null}{build?.warnings?.length ? <ul>{build.warnings.map((warning) => <li key={warning}>{warning}</li>)}</ul> : null}{build?.stage === "failed" && lastSuccess?.output_path ? <Status kind="success">最近一次成功构建仍可用：<code>{lastSuccess.output_path}</code></Status> : null}</div></Section>
              </aside>
            </div>
            <div className="sticky-actions"><span>{dirty ? "请校验并预览写入" : "配置与磁盘一致"}</span><div className="button-row"><button className="button button-secondary" disabled={busy} onClick={validate}>校验配置</button><button className="button button-primary" disabled={busy || !dirty} onClick={prepare}>预览写入 diff</button><button className="button button-success" disabled={busy || dirty} onClick={startBuild}>开始构建</button></div></div>
          </>
        ) : null}
      </main>

      <AlertDialog.Root open={Boolean(pending)} onOpenChange={(open) => { if (!open && !busy) setPending(null); }}>
        <AlertDialog.Portal>
          <AlertDialog.Backdrop className="dialog-backdrop" />
          <AlertDialog.Viewport className="dialog-viewport">
            <AlertDialog.Popup className="dialog-popup">
              <AlertDialog.Title className="dialog-title">确认写入 `.repolens.yml`</AlertDialog.Title>
              <AlertDialog.Description className="dialog-description">请检查目标路径与完整 diff。写入会规范化 YAML，不保留注释、空行、原键顺序或未知字段；文件已在外部修改时，提交会因 revision 冲突被拒绝。</AlertDialog.Description>
              {notice?.kind === "error" ? <Status kind="error">{notice.message}</Status> : null}
              <code className="target-path">{activePath}/.repolens.yml</code>
              <pre className="diff-view">{pending?.diff}</pre>
              <div className="button-row end"><AlertDialog.Close className="button button-secondary" disabled={busy}>返回编辑</AlertDialog.Close><button className="button button-primary" disabled={busy} onClick={commit}>确认原子写入</button></div>
            </AlertDialog.Popup>
          </AlertDialog.Viewport>
        </AlertDialog.Portal>
      </AlertDialog.Root>
    </div>
  );
}
