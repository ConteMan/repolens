import type { InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TextareaHTMLAttributes } from "react";

export function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <label className="field">
      <span className="field-label">{label}</span>
      {children}
      {hint ? <span className="field-hint">{hint}</span> : null}
    </label>
  );
}

export function TextInput(props: InputHTMLAttributes<HTMLInputElement>) {
  return <input className="control" {...props} />;
}

export function TextArea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className="control textarea" {...props} />;
}

export function NativeSelect(props: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select className="control" {...props} />;
}

export function TriState({ value, onChange }: { value: boolean | null; onChange: (value: boolean | null) => void }) {
  return (
    <NativeSelect
      value={value === null ? "" : String(value)}
      onChange={(event) => onChange(event.target.value === "" ? null : event.target.value === "true")}
    >
      <option value="">使用默认值</option>
      <option value="true">启用</option>
      <option value="false">关闭</option>
    </NativeSelect>
  );
}

export function Section({ title, description, children }: { title: string; description?: string; children: ReactNode }) {
  return (
    <section className="section-card">
      <header className="section-header">
        <h3>{title}</h3>
        {description ? <p>{description}</p> : null}
      </header>
      <div className="section-content">{children}</div>
    </section>
  );
}

export function Status({ kind, children }: { kind: "info" | "success" | "warning" | "error"; children: ReactNode }) {
  return (
    <div className={`status status-${kind}`} role="status">
      {children}
    </div>
  );
}
