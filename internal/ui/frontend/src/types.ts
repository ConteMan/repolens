export type Optional<T> = T | null;

export interface MarkdownOptions {
  toc: Optional<boolean>;
  toc_min_headings: Optional<number>;
  anchors: Optional<boolean>;
  mermaid: Optional<boolean>;
  math: Optional<boolean>;
  frontmatter_title: Optional<boolean>;
}

export interface FileOptions {
  render: Optional<boolean>;
  markdown: MarkdownOptions;
  html: { view: Optional<string> };
  code: { line_numbers: Optional<boolean>; theme: Optional<string> };
  max_file_size: Optional<number>;
}

export interface Rule extends FileOptions {
  match: Optional<string>;
}

export interface RepositorySettings {
  site: { title: Optional<string>; language: Optional<string>; home: Optional<string> };
  ignore: string[] | null;
  render: FileOptions;
  rules: Rule[] | null;
  theme: { vars: Optional<Record<string, string>>; css: Optional<string>; templates: Optional<string> };
  view: {
    tree_position: Optional<string>;
    tree_expand_depth: Optional<number>;
    toc_panel: Optional<string>;
    search: Optional<boolean>;
  };
  agent: {
    llms_txt: Optional<boolean>;
    llms_full: { enabled: Optional<boolean>; max_size: Optional<number> };
    index_json: Optional<boolean>;
  };
}

export interface ConfigResponse {
  settings: RepositorySettings;
  revision: string;
}

export interface PrepareResponse extends ConfigResponse {
  before: string;
  after: string;
  diff: string;
}

export interface BuildResponse {
  id: string;
  stage: string;
  stats?: { Files: number; Pages: number; Duration: number };
  warnings?: string[];
  error?: string;
  output_path?: string;
}

export function emptyFileOptions(): FileOptions {
  return {
    render: null,
    markdown: {
      toc: null,
      toc_min_headings: null,
      anchors: null,
      mermaid: null,
      math: null,
      frontmatter_title: null,
    },
    html: { view: null },
    code: { line_numbers: null, theme: null },
    max_file_size: null,
  };
}

export function normalizeSettings(value: Partial<RepositorySettings> | undefined): RepositorySettings {
  const file = emptyFileOptions();
  const render = value?.render ?? file;
  return {
    site: { title: null, language: null, home: null, ...value?.site },
    ignore: Array.isArray(value?.ignore) ? value.ignore : null,
    render: {
      ...file,
      ...render,
      markdown: { ...file.markdown, ...render.markdown },
      html: { ...file.html, ...render.html },
      code: { ...file.code, ...render.code },
    },
    rules: Array.isArray(value?.rules)
      ? value.rules.map((rule) => ({
          ...emptyFileOptions(),
          ...rule,
          match: rule.match ?? null,
          markdown: { ...emptyFileOptions().markdown, ...rule.markdown },
          html: { ...emptyFileOptions().html, ...rule.html },
          code: { ...emptyFileOptions().code, ...rule.code },
        }))
      : null,
    theme: { vars: null, css: null, templates: null, ...value?.theme },
    view: {
      tree_position: null,
      tree_expand_depth: null,
      toc_panel: null,
      search: null,
      ...value?.view,
    },
    agent: {
      llms_txt: null,
      index_json: null,
      ...value?.agent,
      llms_full: { enabled: null, max_size: null, ...value?.agent?.llms_full },
    },
  };
}
