import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import type { Page } from "@playwright/test";

export function createRepository(config: string): string {
  const repository = mkdtempSync(join(tmpdir(), "repolens-ui-e2e-"));
  execFileSync("git", ["init", "--quiet", repository]);
  execFileSync("git", ["-C", repository, "config", "user.email", "ui-test@example.invalid"]);
  execFileSync("git", ["-C", repository, "config", "user.name", "repolens UI test"]);
  writeFileSync(join(repository, "README.md"), "# Fixture\n");
  writeConfig(repository, config);
  execFileSync("git", ["-C", repository, "add", "README.md", ".repolens.yml"]);
  execFileSync("git", ["-C", repository, "commit", "--quiet", "-m", "test: add fixture"]);
  return repository;
}

export function writeConfig(repository: string, config: string): void {
  writeFileSync(join(repository, ".repolens.yml"), config);
}

export function readConfig(repository: string): string {
  return readFileSync(join(repository, ".repolens.yml"), "utf8");
}

export function removeRepository(repository: string): void {
  rmSync(repository, { recursive: true, force: true });
}

export async function openProject(page: Page, repository: string): Promise<void> {
  await page.getByLabel("仓库绝对路径").fill(repository);
  await page.getByRole("button", { name: "打开项目" }).click();
  await page.getByText("配置已加载。修改后请先校验并预览 diff。", { exact: true }).waitFor();
}

export async function expectNoPageOverflow(page: Page): Promise<void> {
  const overflows = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth);
  if (!overflows) throw new Error("页面出现了横向溢出");
}
