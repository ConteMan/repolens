import type { BuildResponse, ConfigResponse, PrepareResponse, ProjectOpenResponse, RepositorySettings, ValidationIssue } from "./types";

const token = document.querySelector<HTMLMetaElement>('meta[name="repolens-csrf-token"]')?.content ?? "";

export class APIError extends Error {
  constructor(
    message: string,
    readonly code: string,
    readonly status: number,
    readonly field?: string,
    readonly issues: ValidationIssue[] = [],
    readonly warnings: ValidationIssue[] = [],
    readonly outputPath?: string,
  ) {
    super(message);
  }
}

async function parse<T>(response: Response): Promise<T> {
  const payload = (await response.json().catch(() => ({}))) as { code?: string; message?: string; field?: string; issues?: ValidationIssue[]; warnings?: ValidationIssue[]; output_path?: string } & T;
  if (!response.ok) {
    throw new APIError(
      payload.message || `请求失败（HTTP ${response.status}）`,
      payload.code || "request_failed",
      response.status,
      payload.field,
      payload.issues ?? [],
      payload.warnings ?? [],
      payload.output_path,
    );
  }
  return payload;
}

async function post<T>(path: string, body: unknown): Promise<T> {
  return parse<T>(
    await fetch(path, {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Repolens-CSRF-Token": token },
      body: JSON.stringify(body),
    }),
  );
}

export const api = {
  open: (path: string) => post<ProjectOpenResponse>("/api/project/open", { path }),
  validate: (path: string, settings: RepositorySettings, revision: string) =>
    post<ConfigResponse>("/api/config/validate", { path, settings, revision }),
  prepare: (path: string, settings: RepositorySettings, revision: string) =>
    post<PrepareResponse>("/api/config/prepare-write", { path, settings, revision }),
  commit: (path: string, settings: RepositorySettings, revision: string) =>
    post<ConfigResponse>("/api/config/commit", { path, settings, revision, confirm: true }),
  startBuild: (path: string, outputPath = "", confirmOverwrite = false) =>
    post<BuildResponse>("/api/build", { path, output_path: outputPath, confirm_overwrite: confirmOverwrite }),
  getBuild: async (id: string) => parse<BuildResponse>(await fetch(`/api/build/${encodeURIComponent(id)}`)),
};
