import { execFileSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { expect, test } from "@playwright/test";

test("opens a repository and prepares a configuration diff", async ({ page }) => {
  const repository = mkdtempSync(join(tmpdir(), "repolens-ui-e2e-"));
  try {
    execFileSync("git", ["init", "--quiet", repository]);
    execFileSync("git", ["-C", repository, "config", "user.email", "ui-test@example.invalid"]);
    execFileSync("git", ["-C", repository, "config", "user.name", "repolens UI test"]);
    writeFileSync(join(repository, "README.md"), "# Fixture\n");
    writeFileSync(join(repository, ".repolens.yml"), "site:\n  title: Before migration\n");
    execFileSync("git", ["-C", repository, "add", "README.md", ".repolens.yml"]);
    execFileSync("git", ["-C", repository, "commit", "--quiet", "-m", "test: add fixture"]);

    const response = await page.goto("/");
    expect(response?.headers()["content-security-policy"]).toContain("script-src 'self'");
    expect(response?.headers()["content-security-policy"]).not.toContain("unsafe-inline");

    await page.getByLabel("仓库绝对路径").fill(repository);
    await page.getByRole("button", { name: "打开项目" }).click();
    await expect(page.getByText("配置已加载。修改后请先校验并预览 diff。", { exact: true })).toBeVisible();

    await page.getByLabel("标题", { exact: true }).fill("After migration");
    await page.getByRole("button", { name: "校验配置" }).click();
    await expect(page.getByText("配置校验通过。", { exact: true })).toBeVisible();

    await page.getByRole("button", { name: "预览写入 diff" }).click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toBeVisible();
    await expect(dialog).toContainText("After migration");
    await expect(dialog).toContainText("写入会规范化 YAML");
  } finally {
    rmSync(repository, { recursive: true, force: true });
  }
});
