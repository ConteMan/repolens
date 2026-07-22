import { forwardRef, type ButtonHTMLAttributes, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes, type TextareaHTMLAttributes } from "react";

type ButtonVariant = "primary" | "secondary" | "success" | "ghost" | "danger";

export const Button = forwardRef<HTMLButtonElement, ButtonHTMLAttributes<HTMLButtonElement> & { variant?: ButtonVariant }>(
  function Button({ variant = "secondary", className = "", type = "button", ...props }, ref) {
    return <button ref={ref} type={type} className={`button button-${variant} ${className}`.trim()} {...props} />;
  },
);

export function Badge({ kind = "default", children }: { kind?: "default" | "repository" | "default-value" | "dirty"; children: ReactNode }) {
  return <span className={`badge badge-${kind}`}>{children}</span>;
}

export function Field({ label, hint, error, errorID, children }: { label: ReactNode; hint?: string; error?: string; errorID?: string; children: ReactNode }) {
  return (
    <label className={`field${error ? " field-invalid" : ""}`}>
      <span className="field-label">{label}</span>
      {children}
      {hint ? <span className="field-hint">{hint}</span> : null}
      {error ? <span id={errorID} className="field-error">{error}</span> : null}
    </label>
  );
}

export function TextInput(props: InputHTMLAttributes<HTMLInputElement>) { return <input className="control" {...props} />; }
export function TextArea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) { return <textarea className="control textarea" {...props} />; }
export function NativeSelect(props: SelectHTMLAttributes<HTMLSelectElement>) { return <select className="control" {...props} />; }

export function TriState({ value, onChange }: { value: boolean | null; onChange: (value: boolean | null) => void }) {
  return <NativeSelect value={value === null ? "" : String(value)} onChange={(event) => onChange(event.target.value === "" ? null : event.target.value === "true")}>
    <option value="">使用默认值</option><option value="true">启用</option><option value="false">关闭</option>
  </NativeSelect>;
}

export function Section({ title, description, children, actions }: { title: ReactNode; description?: string; children: ReactNode; actions?: ReactNode }) {
  return <section className="section-card"><header className="section-header"><div><h2>{title}</h2>{description ? <p>{description}</p> : null}</div>{actions}</header><div className="section-content">{children}</div></section>;
}

export function Alert({ kind, role, children }: { kind: "info" | "success" | "warning" | "error"; role?: "status" | "alert"; children: ReactNode }) {
  return <div className={`alert alert-${kind}`} role={role ?? (kind === "error" ? "alert" : "status")}>{children}</div>;
}

export const Status = Alert;

export function WorkspaceHeader({ path }: { path: string }) {
  return <header className="topbar"><div className="brand"><strong>repolens ui</strong><span>本地仓库配置与构建</span></div><code title={path}>{path || "未打开项目"}</code></header>;
}

export function WorkspaceSidebar({ active, onSelect }: { active: string; onSelect: (section: string) => void }) {
  const items = [["overview", "概览"], ["site", "站点"], ["render", "渲染"], ["rules", "规则"], ["theme", "主题"], ["view", "浏览"], ["agent", "Agent"]] as const;
  return <nav className="workspace-nav" aria-label="配置分区">{items.map(([id, label]) => <button key={id} type="button" className={active === id ? "nav-item active" : "nav-item"} aria-current={active === id ? "page" : undefined} onClick={() => onSelect(id)}>{label}</button>)}</nav>;
}

export function StickyActions({ children, state }: { children: ReactNode; state: ReactNode }) {
  return <div className="sticky-actions"><span>{state}</span><div className="button-row">{children}</div></div>;
}
