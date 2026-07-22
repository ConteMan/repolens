import { expect, test } from "@playwright/test";
import { existsSync, mkdtempSync, realpathSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { basename, dirname, join } from "node:path";
import { createRepository, expectNoPageOverflow, openProject, readConfig, removeRepository, writeConfig } from "./fixtures";

test("binds an invalid project path error to the path field", async ({ page }) => {
  await page.goto("/");
  const input = page.getByLabel("仓库绝对路径");
  await input.fill("relative/repository");
  await page.getByRole("button", { name: "打开项目" }).click();

  await expect(input).toHaveValue("relative/repository");
  await expect(input).toBeFocused();
  await expect(input).toHaveAttribute("aria-invalid", "true");
  await expect(input).toHaveAttribute("aria-describedby", "project-path-error");
  await expect(page.locator("#project-path-error")).toContainText("path must name an existing absolute directory");
  await expect(page.getByRole("alert")).toContainText("path must name an existing absolute directory");
});

test("opens a repository and prepares a configuration diff", async ({ page }) => {
  const repository = createRepository("site:\n  title: Before migration\n");
  try {
    const response = await page.goto("/");
    expect(response?.headers()["content-security-policy"]).toContain("script-src 'self'");
    expect(response?.headers()["content-security-policy"]).not.toContain("unsafe-inline");

    await openProject(page, repository);
    await page.getByRole("navigation", { name: "配置分区" }).getByRole("button", { name: "站点" }).click();

    await page.getByLabel("标题", { exact: true }).fill("After migration");
    await page.getByRole("button", { name: "校验配置" }).click();
    await expect(page.getByText("配置校验通过。", { exact: true })).toBeVisible();

    await page.getByRole("button", { name: "预览写入 diff" }).click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toBeVisible();
    await expect(dialog).toContainText("After migration");
    await expect(dialog).toContainText("写入会规范化 YAML");

  } finally {
    removeRepository(repository);
  }
});

test("shows three validation errors, maps them to fields, and restores focus across sections", async ({ page }) => {
  const repository = createRepository("site:\n  title: Validation fixture\n");
  try {
    await page.goto("/");
    await openProject(page, repository);
    const navigation = page.getByRole("navigation", { name: "配置分区" });

    await navigation.getByRole("button", { name: "站点" }).click();
    await page.locator('textarea[name="ignore"]').fill("[");
    await navigation.getByRole("button", { name: "渲染" }).click();
    await page.locator('input[name="render.max_file_size"]').fill("-1");
    await page.locator('select[name="render.html.view"]').evaluate((element) => {
      const select = element as HTMLSelectElement;
      const option = document.createElement("option");
      option.value = "unsupported";
      option.text = "unsupported";
      select.add(option);
      select.value = "unsupported";
      select.dispatchEvent(new Event("change", { bubbles: true }));
    });

    await page.getByRole("button", { name: "校验配置" }).click();
    const summary = page.locator(".validation-summary");
    await expect(summary).toContainText("ignore[0]");
    await expect(summary).toContainText("render.max_file_size");
    await expect(summary).toContainText("render.html.view");
    await expect(summary.getByRole("button")).toHaveCount(3);

    const ignore = page.locator('textarea[name="ignore"]');
    await expect(ignore).toBeFocused();
    await expect(ignore).toHaveAttribute("aria-invalid", "true");
    await expect(ignore).toHaveAttribute("aria-describedby", "error-ignore");
    await expect(page.locator("#error-ignore")).toBeVisible();

    await summary.getByRole("button").nth(1).click();
    const maxFileSize = page.locator('input[name="render.max_file_size"]');
    const htmlView = page.locator('select[name="render.html.view"]');
    await expect(maxFileSize).toBeFocused();
    await expect(maxFileSize).toHaveAttribute("aria-invalid", "true");
    await expect(maxFileSize).toHaveAttribute("aria-describedby", "error-render-max_file_size");
    await expect(page.locator("#error-render-max_file_size")).toBeVisible();
    await expect(htmlView).toHaveAttribute("aria-invalid", "true");
    await expect(htmlView).toHaveAttribute("aria-describedby", "error-render-html-view");
    await expect(page.locator("#error-render-html-view")).toBeVisible();

    await summary.getByRole("button").nth(2).click();
    await expect(htmlView).toBeFocused();
  } finally {
    removeRepository(repository);
  }
});

test("reloads the repository after a revision conflict without overwriting the external change", async ({ page }) => {
  const repository = createRepository("site:\n  title: Before conflict\n");
  try {
    await page.goto("/");
    await openProject(page, repository);
    await page.getByRole("navigation", { name: "配置分区" }).getByRole("button", { name: "站点" }).click();
    await page.getByLabel("标题", { exact: true }).fill("Draft title");
    await page.getByRole("button", { name: "预览写入 diff" }).click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toBeVisible();

    writeConfig(repository, "site:\n  title: External title\n");
    await dialog.getByRole("button", { name: "确认原子写入" }).click();
    await expect(dialog).toContainText("repository configuration changed");
    await expect(dialog.getByRole("button", { name: "重新读取仓库" })).toBeVisible();
    await dialog.getByRole("button", { name: "重新读取仓库" }).click();

    await expect(dialog).toBeHidden();
    await expect(page.getByLabel("标题", { exact: true })).toHaveValue("External title");
    expect(readConfig(repository)).toContain("title: External title");
  } finally {
    removeRepository(repository);
  }
});

test("shows open warnings and preserves the previous successful build after a failed build", async ({ page }) => {
  const repository = createRepository("source:\n  repo: ignored\nsite:\n  title: Build fixture\n");
  try {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/");
    await openProject(page, repository);
    await expect(page.getByText(/source is ignored in repository config/)).toBeVisible();

    await page.getByRole("button", { name: "开始构建" }).click();
    const buildState = page.locator(".build-state");
    await expect(buildState.getByText("completed", { exact: true })).toBeVisible({ timeout: 15_000 });
    const successPath = await buildState.locator("code").textContent();
    expect(successPath).toBeTruthy();
    const statRows = buildState.locator("dl div");
    await expect(statRows).toHaveCount(3);
    const statPositions = await Promise.all([0, 1, 2].map(async (index) => (await statRows.nth(index).boundingBox())?.y ?? 0));
    expect(statPositions[0]).toBeLessThan(statPositions[1]);
    expect(statPositions[1]).toBeLessThan(statPositions[2]);

    writeConfig(repository, "theme:\n  css: missing.css\n");
    await page.getByRole("button", { name: "开始构建" }).click();
    await expect(buildState.getByText("failed", { exact: true })).toBeVisible({ timeout: 15_000 });
    await expect(buildState).toContainText("missing.css");
    await expect(buildState).toContainText("最近一次成功构建仍可用");
    await expect(buildState).toContainText(successPath ?? "");
  } finally {
    removeRepository(repository);
  }
});

test("builds to a custom output and explicitly confirms replacing owned output", async ({ page }) => {
  const repository = createRepository("site:\n  title: Custom output fixture\n");
  const outputRoot = mkdtempSync(join(tmpdir(), "repolens-ui-output-"));
  const output = join(outputRoot, "published-site");
  const normalizedOutput = join(realpathSync(dirname(output)), basename(output));
  const originalConfig = readConfig(repository);
  try {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto("/");
    await openProject(page, repository);
    await page.getByRole("navigation", { name: "配置分区" }).getByRole("button", { name: "构建" }).click();

    await page.getByRole("button", { name: "自定义目录" }).click();
    const outputInput = page.getByLabel("绝对输出路径");
    await outputInput.fill(output);
    await page.getByRole("button", { name: "开始构建" }).click();

    const buildState = page.locator(".build-state");
    await expect(buildState.getByText("completed", { exact: true })).toBeVisible({ timeout: 15_000 });
    await expect(buildState.getByText("自定义目录", { exact: true })).toBeVisible();
    await expect(buildState.locator("code").first()).toHaveText(normalizedOutput);
    expect(existsSync(join(output, "view", "index.html"))).toBeTruthy();
    expect(readConfig(repository)).toBe(originalConfig);

    await page.getByRole("button", { name: "开始构建" }).click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toContainText(normalizedOutput);
    await expect(dialog).toContainText(".repolens-build");
    await dialog.getByRole("button", { name: "返回修改" }).click();
    await expect(dialog).toBeHidden();
    await expect(outputInput).toBeFocused();

    await page.getByRole("button", { name: "开始构建" }).click();
    await expect(dialog).toBeVisible();
    const accepted = page.waitForResponse((response) => response.url().endsWith("/api/build") && response.request().method() === "POST" && response.status() === 202);
    await dialog.getByRole("button", { name: "确认替换并构建" }).click();
    await accepted;
    await expect(dialog).toBeHidden();
    await expect(buildState.getByText("completed", { exact: true })).toBeVisible({ timeout: 15_000 });
    expect(existsSync(join(output, ".repolens-build"))).toBeTruthy();
    await expectNoPageOverflow(page);
  } finally {
    removeRepository(repository);
    rmSync(outputRoot, { recursive: true, force: true });
  }
});

test("binds an unsafe custom output error on a 390px viewport", async ({ page }) => {
  const repository = createRepository("site:\n  title: Unsafe output fixture\n");
  try {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/");
    await openProject(page, repository);
    await page.getByRole("navigation", { name: "配置分区" }).getByRole("button", { name: "构建" }).click();
    await page.getByRole("button", { name: "自定义目录" }).click();

    const outputInput = page.getByLabel("绝对输出路径");
    await outputInput.fill(join(repository, "dist"));
    await page.getByRole("button", { name: "开始构建" }).click();

    await expect(outputInput).toBeFocused();
    await expect(outputInput).toHaveAttribute("aria-invalid", "true");
    await expect(outputInput).toHaveAttribute("aria-describedby", "error-output_path");
    await expect(page.locator("#error-output_path")).toContainText("outside");
    await expect(page.locator(".validation-summary")).toContainText("output_path");
    await expectNoPageOverflow(page);
  } finally {
    removeRepository(repository);
  }
});

test("keeps the 1440px and 1024px workspace layouts free of page overflow", async ({ browser }) => {
  const repository = createRepository("site:\n  title: Desktop fixture\nrules:\n  - match: docs/**\n");
  try {
    for (const viewport of [{ width: 1440, height: 900 }, { width: 1024, height: 768 }]) {
      const page = await browser.newPage({ viewport });
      await page.goto("/");
      await openProject(page, repository);
      await page.getByRole("navigation", { name: "配置分区" }).getByRole("button", { name: "站点" }).click();
      await expect(page.getByLabel("标题", { exact: true })).toBeVisible();
      const context = page.getByRole("complementary", { name: "持续仓库上下文" });
      await expect(context).toBeVisible();
      await page.getByRole("navigation", { name: "配置分区" }).getByRole("button", { name: "规则" }).click();
      await expect(context).toContainText("rules[0].match");
      await expectNoPageOverflow(page);
      await page.close();
    }
  } finally {
    removeRepository(repository);
  }
});

test("keeps the 390px flow reachable with a rule detail page and a horizontally scrollable diff", async ({ page }) => {
  const repository = createRepository(`site:
  title: Narrow fixture
rules:
  - match: documentation/with/a/very/long/path/that/must/not/overflow/**
  - match: packages/another/long/path/**
  - match: internal/ui/frontend/e2e/**
`);
  try {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/");
    await openProject(page, repository);
    await expectNoPageOverflow(page);

    const navigation = page.getByRole("navigation", { name: "配置分区" });
    await navigation.getByRole("button", { name: "规则" }).click();
    await page.getByRole("button", { name: "编辑" }).first().click();
    await expect(page.getByRole("heading", { name: "规则 1" })).toBeVisible();
    await expectNoPageOverflow(page);
    await page.getByRole("button", { name: "返回规则列表" }).click();

    await navigation.getByRole("button", { name: "站点" }).click();
    await page.getByLabel("标题", { exact: true }).fill("A".repeat(600));
    const preview = page.getByRole("button", { name: "预览写入 diff" });
    await preview.focus();
    await preview.click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toBeVisible();
    expect(await page.locator(".diff-view").evaluate((element) => element.scrollWidth > element.clientWidth)).toBeTruthy();
    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();
    await expect(preview).toBeFocused();

    await page.getByLabel("标题", { exact: true }).scrollIntoViewIfNeeded();
    const [control, sticky] = await Promise.all([
      page.getByLabel("标题", { exact: true }).boundingBox(),
      page.locator(".sticky-actions").boundingBox(),
    ]);
    expect(control).not.toBeNull();
    expect(sticky).not.toBeNull();
    expect((control?.y ?? Number.POSITIVE_INFINITY) + (control?.height ?? 0)).toBeLessThanOrEqual(sticky?.y ?? Number.NEGATIVE_INFINITY);
    await expectNoPageOverflow(page);
  } finally {
    removeRepository(repository);
  }
});
