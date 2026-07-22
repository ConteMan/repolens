import { AlertDialog } from "@base-ui/react/alert-dialog";
import { useEffect, useMemo, useRef, useState } from "react";
import { api, APIError } from "./api";
import { Alert, Badge, Button, Field, NativeSelect, Section, StickyActions, TextArea, TextInput, TriState, WorkspaceHeader, WorkspaceSidebar } from "./components";
import { emptyFileOptions, normalizeSettings, type BuildResponse, type PrepareResponse, type RepositorySettings, type Rule, type ValidationIssue } from "./types";

type Notice = { kind: "info" | "success" | "warning" | "error"; message: string } | null;
type SectionID = "overview" | "site" | "render" | "rules" | "theme" | "view" | "agent" | "build";

const optionalText = (value: string) => (value.trim() ? value.trim() : null);
const optionalNumber = (value: string) => (value === "" ? null : Number(value));
const formatVars = (vars: Record<string, string> | null) => vars ? Object.entries(vars).map(([key, value]) => `${key}: ${value}`).join("\n") : "";
const parseVars = (value: string) => {
  const entries = value.split("\n").map((line) => line.trim()).filter(Boolean).map((line) => {
    const separator = line.indexOf(":");
    return separator > 0 ? [line.slice(0, separator).trim(), line.slice(separator + 1).trim()] : null;
  }).filter((entry): entry is [string, string] => Boolean(entry?.[0]));
  return entries.length ? Object.fromEntries(entries) : null;
};
const formatEffectiveValue = (value: unknown): string => {
  if (value === null) return "null";
  if (typeof value === "string") return value || "（空）";
  if (typeof value === "boolean" || typeof value === "number") return String(value);
  return JSON.stringify(value);
};
const flattenEffectiveValues = (value: unknown, prefix = ""): Array<[string, unknown]> => {
  if (value === null) return [];
  if (Array.isArray(value)) return value.flatMap((item, index) => flattenEffectiveValues(item, `${prefix}[${index}]`));
  if (value && typeof value === "object") return Object.entries(value).flatMap(([key, item]) => flattenEffectiveValues(item, prefix ? `${prefix}.${key}` : key));
  return prefix ? [[prefix, value]] : [];
};
const normalizeIssuePath = (path: string) => path.replace(/^ignore\[\d+\]$/, "ignore");
const errorID = (path: string) => `error-${normalizeIssuePath(path).replace(/[^a-zA-Z0-9_-]/g, "-")}`;

function sectionForIssue(path: string): SectionID {
  if (path.startsWith("rules[")) return "rules";
  if (path.startsWith("render.")) return "render";
  if (path.startsWith("theme.")) return "theme";
  if (path.startsWith("view.")) return "view";
  if (path.startsWith("agent.")) return "agent";
  return "site";
}

function RuleEditor({ rule, index, count, mobileDetail = false, onChange, onMove, onRemove, onOpenMobile, fieldError }: {
  rule: Rule; index: number; count: number; mobileDetail?: boolean; onChange: (rule: Rule) => void; onMove: (offset: number) => void; onRemove: () => void; onOpenMobile: () => void; fieldError: (path: string) => string | undefined;
}) {
  const path = (field: string) => `rules[${index}].${field}`;
  return <article className={`rule-card${mobileDetail ? " rule-detail" : ""}`}>
    <header className="rule-header"><div><span className="eyebrow">规则 {index + 1}</span><strong>{rule.match || "尚未设置匹配模式"}</strong></div><div className="button-row"><Button className="mobile-rule-edit" onClick={onOpenMobile}>编辑</Button><Button variant="ghost" disabled={index === 0} onClick={() => onMove(-1)}>上移</Button><Button variant="ghost" disabled={index === count - 1} onClick={() => onMove(1)}>下移</Button><Button variant="danger" onClick={onRemove}>删除</Button></div></header>
    <div className="field-grid three">
      <Field label="匹配模式" error={fieldError(path("match"))} errorID={errorID(path("match"))}><TextInput name={path("match")} aria-invalid={Boolean(fieldError(path("match")))} aria-describedby={fieldError(path("match")) ? errorID(path("match")) : undefined} value={rule.match ?? ""} placeholder="docs/**" onChange={(e) => onChange({ ...rule, match: optionalText(e.target.value) })} /></Field>
      <Field label="生成阅读页"><TriState value={rule.render} onChange={(render) => onChange({ ...rule, render })} /></Field>
      <Field label="最大文件大小（字节）" error={fieldError(path("max_file_size"))} errorID={errorID(path("max_file_size"))}><TextInput name={path("max_file_size")} aria-invalid={Boolean(fieldError(path("max_file_size")))} aria-describedby={fieldError(path("max_file_size")) ? errorID(path("max_file_size")) : undefined} type="number" min="0" value={rule.max_file_size ?? ""} onChange={(e) => onChange({ ...rule, max_file_size: optionalNumber(e.target.value) })} /></Field>
      <Field label="Markdown 数学公式"><TriState value={rule.markdown.math} onChange={(math) => onChange({ ...rule, markdown: { ...rule.markdown, math } })} /></Field>
      <Field label="Markdown 目录"><TriState value={rule.markdown.toc} onChange={(toc) => onChange({ ...rule, markdown: { ...rule.markdown, toc } })} /></Field>
      <Field label="目录最少标题数" error={fieldError(path("markdown.toc_min_headings"))} errorID={errorID(path("markdown.toc_min_headings"))}><TextInput name={path("markdown.toc_min_headings")} aria-invalid={Boolean(fieldError(path("markdown.toc_min_headings")))} aria-describedby={fieldError(path("markdown.toc_min_headings")) ? errorID(path("markdown.toc_min_headings")) : undefined} type="number" min="0" value={rule.markdown.toc_min_headings ?? ""} onChange={(e) => onChange({ ...rule, markdown: { ...rule.markdown, toc_min_headings: optionalNumber(e.target.value) } })} /></Field>
      <Field label="标题锚点"><TriState value={rule.markdown.anchors} onChange={(anchors) => onChange({ ...rule, markdown: { ...rule.markdown, anchors } })} /></Field>
      <Field label="Mermaid"><TriState value={rule.markdown.mermaid} onChange={(mermaid) => onChange({ ...rule, markdown: { ...rule.markdown, mermaid } })} /></Field>
      <Field label="Frontmatter 标题"><TriState value={rule.markdown.frontmatter_title} onChange={(frontmatter_title) => onChange({ ...rule, markdown: { ...rule.markdown, frontmatter_title } })} /></Field>
      <Field label="HTML 展示方式" error={fieldError(path("html.view"))} errorID={errorID(path("html.view"))}><NativeSelect name={path("html.view")} aria-invalid={Boolean(fieldError(path("html.view")))} aria-describedby={fieldError(path("html.view")) ? errorID(path("html.view")) : undefined} value={rule.html.view ?? ""} onChange={(e) => onChange({ ...rule, html: { view: optionalText(e.target.value) } })}><option value="">使用默认值</option><option value="embed">嵌入</option><option value="direct">直接链接</option><option value="source">源码</option></NativeSelect></Field>
      <Field label="代码行号"><TriState value={rule.code.line_numbers} onChange={(line_numbers) => onChange({ ...rule, code: { ...rule.code, line_numbers } })} /></Field>
      <Field label="代码主题"><TextInput value={rule.code.theme ?? ""} onChange={(e) => onChange({ ...rule, code: { ...rule.code, theme: optionalText(e.target.value) } })} /></Field>
    </div>
  </article>;
}

export function App() {
  const [path, setPath] = useState("");
  const [pathError, setPathError] = useState<string | null>(null);
  const [activePath, setActivePath] = useState("");
  const [revision, setRevision] = useState("");
  const [settings, setSettings] = useState<RepositorySettings | null>(null);
  const [savedSettings, setSavedSettings] = useState<RepositorySettings | null>(null);
  const [notice, setNotice] = useState<Notice>(null);
  const [busy, setBusy] = useState(false);
  const [pending, setPending] = useState<PrepareResponse | null>(null);
  const [dialogError, setDialogError] = useState<APIError | null>(null);
  const [build, setBuild] = useState<BuildResponse | null>(null);
  const [lastSuccess, setLastSuccess] = useState<BuildResponse | null>(null);
  const [effective, setEffective] = useState<RepositorySettings | null>(null);
  const [sources, setSources] = useState<Record<string, "repository" | "default">>({});
  const [loadWarnings, setLoadWarnings] = useState<string[]>([]);
  const [validationIssues, setValidationIssues] = useState<ValidationIssue[]>([]);
  const [activeSection, setActiveSection] = useState<SectionID>("overview");
  const [mobileRuleIndex, setMobileRuleIndex] = useState<number | null>(null);
  const prepareButtonRef = useRef<HTMLButtonElement>(null);

  const dirty = useMemo(() => Boolean(settings && savedSettings && JSON.stringify(settings) !== JSON.stringify(savedSettings)), [settings, savedSettings]);
  const effectiveEntries = useMemo(() => effective ? flattenEffectiveValues(effective) : [], [effective]);
  const contextEntries = useMemo(() => {
    const prefix = activeSection === "site" ? "site." : activeSection === "rules" ? "rules[" : `${activeSection}.`;
    const matching = effectiveEntries.filter(([fieldPath]) => fieldPath.startsWith(prefix));
    return (matching.length ? matching : effectiveEntries).slice(0, 6);
  }, [activeSection, effectiveEntries]);
  const sourceFor = (fieldPath: string) => {
    let candidate = fieldPath;
    while (candidate) {
      if (sources[candidate]) return sources[candidate];
      candidate = candidate.replace(/\[\d+\]$/, "").replace(/(?:\.[^.]+|\[\d+\])$/, "");
    }
    return "default";
  };
  const label = (text: string, fieldPath: string) => <><span>{text}</span><span aria-hidden="true"><Badge kind={sourceFor(fieldPath) === "repository" ? "repository" : "default-value"}>{sourceFor(fieldPath) === "repository" ? "仓库配置" : "默认值"}</Badge></span></>;
  const focusField = (issuePath: string) => {
    const section = sectionForIssue(issuePath);
    setActiveSection(section);
    const ruleMatch = issuePath.match(/^rules\[(\d+)\]/);
    if (section === "rules" && ruleMatch && window.matchMedia("(max-width: 720px)").matches) {
      setMobileRuleIndex(Number(ruleMatch[1]));
    }
    window.requestAnimationFrame(() => window.requestAnimationFrame(() => {
    const normalizedPath = normalizeIssuePath(issuePath);
    const escaped = window.CSS?.escape ? window.CSS.escape(normalizedPath) : normalizedPath.replace(/"/g, '\\"');
    const target = document.querySelector<HTMLElement>(`[name="${escaped}"]`);
    target?.focus(); target?.scrollIntoView({ behavior: "smooth", block: "center" });
    }));
  };
  const showError = (error: unknown, fallback: string) => {
    const message = error instanceof APIError ? error.message : fallback;
    const issues = error instanceof APIError ? error.issues : [];
    setValidationIssues(issues); if (issues[0]?.path) focusField(issues[0].path); setNotice({ kind: "error", message });
  };
  const clearIssue = (issuePath: string) => setValidationIssues((current) => current.filter((issue) => normalizeIssuePath(issue.path) !== normalizeIssuePath(issuePath)));
  const clearIssuesWithPrefix = (prefix: string) => setValidationIssues((current) => current.filter((issue) => !issue.path.startsWith(prefix)));
  const fieldError = (issuePath: string) => validationIssues.find((issue) => normalizeIssuePath(issue.path) === normalizeIssuePath(issuePath))?.message;

  const applyOpenResponse = (response: Awaited<ReturnType<typeof api.open>>, nextPath: string, resetSession: boolean) => {
    const next = normalizeSettings(response.settings);
    setActivePath(nextPath); setRevision(response.revision); setSettings(next); setSavedSettings(next); setEffective(normalizeSettings(response.effective)); setSources(response.sources); setLoadWarnings(response.warnings); setValidationIssues([]);
    if (resetSession) { setBuild(null); setLastSuccess(null); }
  };
  const openProject = async () => {
    if (!path.trim()) {
      const message = "请输入本地 Git 工作树的绝对路径。";
      setPathError(message); setNotice({ kind: "error", message });
      window.requestAnimationFrame(() => document.querySelector<HTMLInputElement>("#project-path")?.focus());
      return;
    }
    setPathError(null);
    setBusy(true); setNotice({ kind: "info", message: "正在读取仓库配置…" }); setValidationIssues([]);
    try { applyOpenResponse(await api.open(path.trim()), path.trim(), true); setActiveSection("overview"); setMobileRuleIndex(null); setNotice({ kind: "success", message: "配置已加载。修改后请先校验并预览 diff。" }); }
    catch (error) {
      if (error instanceof APIError && error.code === "invalid_path") {
        setPathError(error.message);
        window.requestAnimationFrame(() => document.querySelector<HTMLInputElement>("#project-path")?.focus());
      }
      showError(error, "无法打开项目。");
    } finally { setBusy(false); }
  };
  const reloadProject = async () => {
    if (!activePath) return;
    setBusy(true); setDialogError(null);
    try { applyOpenResponse(await api.open(activePath), activePath, false); setPending(null); setNotice({ kind: "success", message: "已重新读取仓库配置。外部修改已作为当前快照加载。" }); }
    catch (error) { setDialogError(error instanceof APIError ? error : new APIError("无法重新读取仓库。", "open_failed", 0)); }
    finally { setBusy(false); }
  };
  const validate = async () => {
    if (!settings) return; setBusy(true); setNotice({ kind: "info", message: "正在校验配置…" }); setValidationIssues([]);
    try { const response = await api.validate(activePath, settings, revision); setSettings(normalizeSettings(response.settings)); setNotice({ kind: "success", message: "配置校验通过。" }); }
    catch (error) { showError(error, "配置校验失败。"); } finally { setBusy(false); }
  };
  const prepare = async () => {
    if (!settings) return; setBusy(true); setNotice({ kind: "info", message: "正在生成 YAML diff…" }); setValidationIssues([]);
    try { const response = await api.prepare(activePath, settings, revision); if (!response.diff) { const next = normalizeSettings(response.settings); setSettings(next); setSavedSettings(next); setNotice({ kind: "success", message: "没有需要写入的配置变更。" }); } else { setPending(response); setDialogError(null); setNotice(null); } }
    catch (error) { showError(error, "无法生成写入预览。"); } finally { setBusy(false); }
  };
  const commit = async () => {
    if (!pending) return; setBusy(true); setDialogError(null);
    try { const response = await api.commit(activePath, normalizeSettings(pending.settings), pending.revision); const next = normalizeSettings(response.settings); setRevision(response.revision); setSettings(next); setSavedSettings(next); setPending(null); setValidationIssues([]); setNotice({ kind: "success", message: "配置已原子写入，现在可以开始构建。" }); }
    catch (error) {
      if (error instanceof APIError && error.code === "revision_conflict") setDialogError(error);
      else if (error instanceof APIError && error.code === "validation_failed") { setPending(null); window.requestAnimationFrame(() => showError(error, "写入配置失败。")); }
      else setDialogError(error instanceof APIError ? error : new APIError("写入配置失败。", "commit_failed", 0));
    } finally { setBusy(false); }
  };
  const startBuild = async () => {
    setBusy(true); setNotice({ kind: "info", message: "正在启动构建…" });
    setActiveSection("build"); setMobileRuleIndex(null);
    try { setBuild(await api.startBuild(activePath)); setNotice(null); } catch (error) { showError(error, "无法启动构建。"); setBusy(false); }
  };
  useEffect(() => {
    if (!build?.id || build.stage === "completed" || build.stage === "failed") return;
    const timer = window.setTimeout(async () => { try { const next = await api.getBuild(build.id); setBuild(next); if (next.stage === "completed") { setLastSuccess(next); setNotice({ kind: "success", message: "构建完成。输出已保存在本机缓存目录。" }); setBusy(false); } if (next.stage === "failed") { setNotice({ kind: "error", message: next.error || "构建失败。" }); setBusy(false); } } catch (error) { showError(error, "读取构建状态失败。"); setBusy(false); } }, 500);
    return () => window.clearTimeout(timer);
  }, [build]);
  const updateRule = (index: number, rule: Rule) => { clearIssuesWithPrefix(`rules[${index}].`); setSettings((current) => current ? { ...current, rules: (current.rules ?? []).map((item, i) => i === index ? rule : item) } : current); };
  const moveRule = (index: number, offset: number) => { clearIssuesWithPrefix("rules["); setSettings((current) => { if (!current) return current; const rules = [...(current.rules ?? [])]; const target = index + offset; if (target < 0 || target >= rules.length) return current; [rules[index], rules[target]] = [rules[target], rules[index]]; return { ...current, rules }; }); };
  const selectSection = (section: string) => { setActiveSection(section as SectionID); setMobileRuleIndex(null); };

  const renderRules = () => {
    const rules = settings?.rules ?? [];
    const removeRule = (index: number) => { clearIssuesWithPrefix("rules["); setSettings((current) => current ? { ...current, rules: (current.rules ?? []).filter((_, i) => i !== index) } : current); setMobileRuleIndex(null); };
    const editor = (rule: Rule, index: number, mobileDetail = false) => <RuleEditor key={index} rule={rule} index={index} count={rules.length} mobileDetail={mobileDetail} fieldError={fieldError} onChange={(next) => updateRule(index, next)} onMove={(offset) => moveRule(index, offset)} onRemove={() => removeRule(index)} onOpenMobile={() => setMobileRuleIndex(index)} />;
    if (mobileRuleIndex !== null && rules[mobileRuleIndex]) return <Section title={`规则 ${mobileRuleIndex + 1}`} description="在窄屏独立编辑规则，完成后返回规则列表。" actions={<Button onClick={() => setMobileRuleIndex(null)}>返回规则列表</Button>}>{editor(rules[mobileRuleIndex], mobileRuleIndex, true)}</Section>;
    return <Section title="路径规则" description="按顺序全部匹配，后写覆盖先写；未编辑的空列表保持未设置。"><div className="rule-list">{rules.map((rule, index) => editor(rule, index))}</div><Button onClick={() => { clearIssuesWithPrefix("rules["); setSettings((current) => current ? { ...current, rules: [...(current.rules ?? []), { ...emptyFileOptions(), match: null }] } : current); }}>新增规则</Button></Section>;
  };
  const renderEffectiveValues = (entries: Array<[string, unknown]>) => entries.length ? <dl className="effective-values">{entries.map(([fieldPath, value]) => <div key={fieldPath}><dt>{fieldPath}</dt><dd>{formatEffectiveValue(value)}</dd><small>{sourceFor(fieldPath) === "repository" ? "仓库配置" : "有效默认值"}</small></div>)}</dl> : <p>没有可显示的有效值。</p>;
  const renderContextRail = () => <aside className="context-rail" aria-label="持续仓库上下文"><Section title="有效配置与来源" description="当前分区的合并快照；不会写入默认值。">{renderEffectiveValues(contextEntries)}</Section><Section title="最近成功构建" description="只在当前页面会话内保留。"><BuildSummary build={build} lastSuccess={lastSuccess} /></Section></aside>;
  const renderSection = () => {
    if (!settings) return null;
    if (activeSection === "overview") return <div className="overview-grid"><Section title="当前项目" description="仓库上下文会持续保留，所有写入都需先确认 diff。"><div className="field-stack"><div><strong>{activePath}</strong><p>修改仅针对仓库信任域；构建结果保存到仓库外缓存。</p></div><div className="button-row"><Badge kind={dirty ? "dirty" : "repository"}>{dirty ? "有未保存修改" : "配置已保存"}</Badge><Badge kind="default">revision {revision.slice(0, 12) || "—"}</Badge></div></div></Section><Section title="快速进入" description="每个状态只保留一个明确主动作。"><div className="overview-actions">{(["site", "render", "rules", "theme", "view", "agent"] as SectionID[]).map((section) => <Button key={section} onClick={() => selectSection(section)}>{({ site: "站点", render: "渲染", rules: "规则", theme: "主题", view: "浏览", agent: "Agent" } as Record<string, string>)[section]}配置</Button>)}</div></Section><Section title="有效配置" description="打开项目时的合并快照；写入仍只使用表单中的仓库值。">{renderEffectiveValues(effectiveEntries)}</Section><Section title="构建结果" description="只在当前页面会话内保留最近一次成功结果。"><BuildSummary build={build} lastSuccess={lastSuccess} /></Section></div>;
    if (activeSection === "site") return <Section title="站点" description="页面标题、语言与默认首页。"><div className="field-grid three"><Field label={label("标题", "site.title")}><TextInput aria-label="标题" value={settings.site.title ?? ""} onChange={(e) => setSettings({ ...settings, site: { ...settings.site, title: optionalText(e.target.value) } })} /></Field><Field label={label("语言", "site.language")}><TextInput aria-label="语言" value={settings.site.language ?? ""} placeholder="zh-CN" onChange={(e) => setSettings({ ...settings, site: { ...settings.site, language: optionalText(e.target.value) } })} /></Field><Field label={label("首页文件", "site.home")}><TextInput aria-label="首页文件" value={settings.site.home ?? ""} placeholder="README.md" onChange={(e) => setSettings({ ...settings, site: { ...settings.site, home: optionalText(e.target.value) } })} /></Field></div><div className="field-stack"><Field label={label("Git glob", "ignore")} hint="每行一个 Git glob；未编辑的空列表保持未设置。" error={fieldError("ignore")} errorID={errorID("ignore")}><TextArea name="ignore" aria-invalid={Boolean(fieldError("ignore"))} aria-describedby={fieldError("ignore") ? errorID("ignore") : undefined} rows={5} value={(settings.ignore ?? []).join("\n")} onChange={(e) => { clearIssue("ignore"); setSettings({ ...settings, ignore: e.target.value.split("\n").map((line) => line.trim()).filter(Boolean) }); }} /></Field></div></Section>;
    if (activeSection === "render") return <Section title="全局渲染" description="作为所有路径规则之前的第 0 条默认规则。"><div className="field-grid three"><Field label={label("生成阅读页", "render.render")}><TriState value={settings.render.render} onChange={(render) => setSettings({ ...settings, render: { ...settings.render, render } })} /></Field><Field label={label("目录", "render.markdown.toc")}><TriState value={settings.render.markdown.toc} onChange={(toc) => setSettings({ ...settings, render: { ...settings.render, markdown: { ...settings.render.markdown, toc } } })} /></Field><Field label={label("目录最少标题数", "render.markdown.toc_min_headings")} error={fieldError("render.markdown.toc_min_headings")} errorID={errorID("render.markdown.toc_min_headings")}><TextInput name="render.markdown.toc_min_headings" aria-invalid={Boolean(fieldError("render.markdown.toc_min_headings"))} aria-describedby={fieldError("render.markdown.toc_min_headings") ? errorID("render.markdown.toc_min_headings") : undefined} type="number" min="0" value={settings.render.markdown.toc_min_headings ?? ""} onChange={(e) => { clearIssue("render.markdown.toc_min_headings"); setSettings({ ...settings, render: { ...settings.render, markdown: { ...settings.render.markdown, toc_min_headings: optionalNumber(e.target.value) } } }); }} /></Field><Field label={label("标题锚点", "render.markdown.anchors")}><TriState value={settings.render.markdown.anchors} onChange={(anchors) => setSettings({ ...settings, render: { ...settings.render, markdown: { ...settings.render.markdown, anchors } } })} /></Field><Field label={label("Mermaid", "render.markdown.mermaid")}><TriState value={settings.render.markdown.mermaid} onChange={(mermaid) => setSettings({ ...settings, render: { ...settings.render, markdown: { ...settings.render.markdown, mermaid } } })} /></Field><Field label={label("数学公式", "render.markdown.math")}><TriState value={settings.render.markdown.math} onChange={(math) => setSettings({ ...settings, render: { ...settings.render, markdown: { ...settings.render.markdown, math } } })} /></Field><Field label={label("Frontmatter 标题", "render.markdown.frontmatter_title")}><TriState value={settings.render.markdown.frontmatter_title} onChange={(frontmatter_title) => setSettings({ ...settings, render: { ...settings.render, markdown: { ...settings.render.markdown, frontmatter_title } } })} /></Field><Field label={label("HTML 展示", "render.html.view")} error={fieldError("render.html.view")} errorID={errorID("render.html.view")}><NativeSelect name="render.html.view" aria-invalid={Boolean(fieldError("render.html.view"))} aria-describedby={fieldError("render.html.view") ? errorID("render.html.view") : undefined} value={settings.render.html.view ?? ""} onChange={(e) => { clearIssue("render.html.view"); setSettings({ ...settings, render: { ...settings.render, html: { view: optionalText(e.target.value) } } }); }}><option value="">使用默认值</option><option value="embed">嵌入</option><option value="direct">直接链接</option><option value="source">源码</option></NativeSelect></Field><Field label={label("代码行号", "render.code.line_numbers")}><TriState value={settings.render.code.line_numbers} onChange={(line_numbers) => setSettings({ ...settings, render: { ...settings.render, code: { ...settings.render.code, line_numbers } } })} /></Field><Field label={label("代码主题", "render.code.theme")}><TextInput value={settings.render.code.theme ?? ""} onChange={(e) => setSettings({ ...settings, render: { ...settings.render, code: { ...settings.render.code, theme: optionalText(e.target.value) } } })} /></Field><Field label={label("最大文件大小（字节）", "render.max_file_size")} error={fieldError("render.max_file_size")} errorID={errorID("render.max_file_size")}><TextInput name="render.max_file_size" aria-invalid={Boolean(fieldError("render.max_file_size"))} aria-describedby={fieldError("render.max_file_size") ? errorID("render.max_file_size") : undefined} type="number" min="0" value={settings.render.max_file_size ?? ""} onChange={(e) => { clearIssue("render.max_file_size"); setSettings({ ...settings, render: { ...settings.render, max_file_size: optionalNumber(e.target.value) } }); }} /></Field></div></Section>;
    if (activeSection === "rules") return renderRules();
    if (activeSection === "theme") return <Section title="主题" description="仅编辑仓库域支持的 vars、css 与 templates。"><div className="field-stack"><Field label={label("变量", "theme.vars")} hint="每行 `名称: 值`"><TextArea rows={6} value={formatVars(settings.theme.vars)} onChange={(e) => setSettings({ ...settings, theme: { ...settings.theme, vars: parseVars(e.target.value) } })} /></Field><Field label={label("附加 CSS", "theme.css")}><TextInput value={settings.theme.css ?? ""} onChange={(e) => setSettings({ ...settings, theme: { ...settings.theme, css: optionalText(e.target.value) } })} /></Field><Field label={label("模板目录", "theme.templates")}><TextInput value={settings.theme.templates ?? ""} onChange={(e) => setSettings({ ...settings, theme: { ...settings.theme, templates: optionalText(e.target.value) } })} /></Field></div></Section>;
    if (activeSection === "view") return <Section title="浏览" description="配置生成站点的浏览层。"><div className="field-stack"><Field label={label("文件树位置", "view.tree_position")} error={fieldError("view.tree_position")} errorID={errorID("view.tree_position")}><NativeSelect name="view.tree_position" aria-invalid={Boolean(fieldError("view.tree_position"))} aria-describedby={fieldError("view.tree_position") ? errorID("view.tree_position") : undefined} value={settings.view.tree_position ?? ""} onChange={(e) => { clearIssue("view.tree_position"); setSettings({ ...settings, view: { ...settings.view, tree_position: optionalText(e.target.value) } }); }}><option value="">使用默认值</option><option value="left">左侧</option><option value="right">右侧</option></NativeSelect></Field><Field label={label("展开层级", "view.tree_expand_depth")} error={fieldError("view.tree_expand_depth")} errorID={errorID("view.tree_expand_depth")}><TextInput name="view.tree_expand_depth" aria-invalid={Boolean(fieldError("view.tree_expand_depth"))} aria-describedby={fieldError("view.tree_expand_depth") ? errorID("view.tree_expand_depth") : undefined} type="number" min="0" value={settings.view.tree_expand_depth ?? ""} onChange={(e) => { clearIssue("view.tree_expand_depth"); setSettings({ ...settings, view: { ...settings.view, tree_expand_depth: optionalNumber(e.target.value) } }); }} /></Field><Field label={label("目录面板", "view.toc_panel")} error={fieldError("view.toc_panel")} errorID={errorID("view.toc_panel")}><NativeSelect name="view.toc_panel" aria-invalid={Boolean(fieldError("view.toc_panel"))} aria-describedby={fieldError("view.toc_panel") ? errorID("view.toc_panel") : undefined} value={settings.view.toc_panel ?? ""} onChange={(e) => { clearIssue("view.toc_panel"); setSettings({ ...settings, view: { ...settings.view, toc_panel: optionalText(e.target.value) } }); }}><option value="">使用默认值</option><option value="floating">浮动</option><option value="inline">内联</option></NativeSelect></Field><Field label={label("站内搜索", "view.search")}><TriState value={settings.view.search} onChange={(search) => setSettings({ ...settings, view: { ...settings.view, search } })} /></Field></div></Section>;
    if (activeSection === "build") return <Section title="构建结果" description="展示结构化阶段、统计、warning、错误与仓库外缓存路径。"><BuildSummary build={build} lastSuccess={lastSuccess} /></Section>;
    return <Section title="Agent 视图" description="配置面向 Agent 的生成产物。"><div className="field-stack"><Field label={label("生成 llms.txt", "agent.llms_txt")}><TriState value={settings.agent.llms_txt} onChange={(llms_txt) => setSettings({ ...settings, agent: { ...settings.agent, llms_txt } })} /></Field><Field label={label("生成 llms-full.txt", "agent.llms_full.enabled")}><TriState value={settings.agent.llms_full.enabled} onChange={(enabled) => setSettings({ ...settings, agent: { ...settings.agent, llms_full: { ...settings.agent.llms_full, enabled } } })} /></Field><Field label={label("llms-full 最大字节数", "agent.llms_full.max_size")} error={fieldError("agent.llms_full.max_size")} errorID={errorID("agent.llms_full.max_size")}><TextInput name="agent.llms_full.max_size" aria-invalid={Boolean(fieldError("agent.llms_full.max_size"))} aria-describedby={fieldError("agent.llms_full.max_size") ? errorID("agent.llms_full.max_size") : undefined} type="number" min="0" value={settings.agent.llms_full.max_size ?? ""} onChange={(e) => { clearIssue("agent.llms_full.max_size"); setSettings({ ...settings, agent: { ...settings.agent, llms_full: { ...settings.agent.llms_full, max_size: optionalNumber(e.target.value) } } }); }} /></Field><Field label={label("生成 index.json", "agent.index_json")}><TriState value={settings.agent.index_json} onChange={(index_json) => setSettings({ ...settings, agent: { ...settings.agent, index_json } })} /></Field></div></Section>;
  };

  return <div className="app-shell" aria-busy={busy}><WorkspaceHeader path={activePath} /><main className="page">
    {!settings ? <section className="open-panel"><div><span className="eyebrow">本地工作树</span><h1>安全地配置并构建仓库</h1><p>输入绝对路径。仓库内容只在本机读取，构建结果写入仓库外缓存。</p></div><div className="path-row"><Field label="仓库绝对路径" error={pathError ?? undefined} errorID="project-path-error"><TextInput id="project-path" aria-label="仓库绝对路径" aria-invalid={Boolean(pathError)} aria-describedby={pathError ? "project-path-error" : undefined} value={path} onChange={(e) => { setPath(e.target.value); setPathError(null); }} placeholder="/Users/name/Projects/repository" disabled={busy} /></Field><Button variant="primary" onClick={openProject} disabled={busy}>打开项目</Button></div></section> : <>
      <div className="workspace-heading"><div><span className="eyebrow">仓库配置</span><h1>编辑 `.repolens.yml`</h1></div><Badge kind={dirty ? "dirty" : "repository"}>{dirty ? "有未保存修改" : "已保存"}</Badge></div>
      {notice ? <Alert kind={notice.kind}>{notice.message}</Alert> : null}
      {validationIssues.length ? <Alert kind="error"><div className="validation-summary"><strong>请修复以下字段后重试：</strong><ul>{validationIssues.map((issue) => <li key={`${issue.path}:${issue.code}`}><button type="button" onClick={() => focusField(issue.path)}><code>{issue.path}</code>：{issue.message}</button></li>)}</ul></div></Alert> : null}
      <Alert kind="warning">仅编辑仓库信任域；`source`、`output`、`access` 不会出现在页面或写入 payload。</Alert>
      {loadWarnings.length ? <Alert kind="warning">读取配置时发现告警：<ul>{loadWarnings.map((warning) => <li key={warning}>{warning}</li>)}</ul></Alert> : null}
      <div className="workspace-shell"><WorkspaceSidebar active={activeSection} onSelect={selectSection} /><div className="workspace-content"><div className="workspace-main">{renderSection()}</div>{activeSection !== "overview" && activeSection !== "build" ? renderContextRail() : null}</div></div>
      <StickyActions state={dirty ? "请校验并预览写入" : "配置与磁盘一致"}><Button disabled={busy} onClick={validate}>校验配置</Button><Button ref={prepareButtonRef} variant="primary" disabled={busy || !dirty} onClick={prepare}>预览写入 diff</Button><Button variant="success" disabled={busy || dirty} onClick={startBuild}>开始构建</Button></StickyActions>
    </>}
    {settings && !notice ? null : notice && !settings ? <Alert kind={notice.kind}>{notice.message}</Alert> : null}
  </main>
  <AlertDialog.Root open={Boolean(pending)} onOpenChange={(open) => { if (!open && !busy) { setPending(null); setDialogError(null); window.requestAnimationFrame(() => prepareButtonRef.current?.focus()); } }}><AlertDialog.Portal><AlertDialog.Backdrop className="dialog-backdrop" /><AlertDialog.Viewport className="dialog-viewport"><AlertDialog.Popup className="dialog-popup"><AlertDialog.Title className="dialog-title">确认写入 `.repolens.yml`</AlertDialog.Title><AlertDialog.Description className="dialog-description">请检查目标路径与完整 diff。写入会规范化 YAML，不保留注释、空行、原键顺序或未知字段；文件已在外部修改时，提交会因 revision 冲突被拒绝。</AlertDialog.Description>{dialogError ? <Alert kind="error">{dialogError.message}{dialogError.code === "revision_conflict" ? <div className="button-row"><Button onClick={reloadProject} disabled={busy}>重新读取仓库</Button></div> : null}</Alert> : null}<code className="target-path">{activePath}/.repolens.yml</code><pre className="diff-view">{pending?.diff}</pre><div className="button-row end"><AlertDialog.Close className="button button-secondary" disabled={busy}>返回编辑</AlertDialog.Close><Button variant="primary" disabled={busy} onClick={commit}>确认原子写入</Button></div></AlertDialog.Popup></AlertDialog.Viewport></AlertDialog.Portal></AlertDialog.Root>
  </div>;
}

function BuildSummary({ build, lastSuccess }: { build: BuildResponse | null; lastSuccess: BuildResponse | null }) {
  return <div className="build-state"><strong>{build ? build.stage : "尚未构建"}</strong>{build?.stats ? <dl><div><dt>文件</dt><dd>{build.stats.Files}</dd></div><div><dt>页面</dt><dd>{build.stats.Pages}</dd></div><div><dt>耗时</dt><dd>{Math.round(build.stats.Duration / 1_000_000)}ms</dd></div></dl> : null}{build?.output_path ? <code>{build.output_path}</code> : null}{build?.warnings?.length ? <ul>{build.warnings.map((warning) => <li key={warning}>{warning}</li>)}</ul> : null}{build?.error ? <Alert kind="error">{build.error}</Alert> : null}{build?.stage === "failed" && lastSuccess?.output_path ? <Alert kind="success">最近一次成功构建仍可用：<code>{lastSuccess.output_path}</code></Alert> : null}</div>;
}
