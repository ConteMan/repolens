import { execFileSync } from "node:child_process";
import { createReadStream, mkdtempSync, mkdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { createServer, type Server } from "node:http";
import { tmpdir } from "node:os";
import { extname, join } from "node:path";
import { expect, test } from "@playwright/test";

let fixtureRoot = "";
let siteURL = "";
let server: Server;
let output = "";

test.beforeAll(async () => {
  fixtureRoot = mkdtempSync(join(tmpdir(), "repolens-glossary-e2e-"));
  const repository = join(fixtureRoot, "repository");
  output = join(fixtureRoot, "dist");
  mkdirSync(join(repository, ".repolens", "glossary"), { recursive: true });
  mkdirSync(join(repository, "docs"), { recursive: true });
  writeFileSync(join(repository, ".repolens.yml"), "render:\n  markdown:\n    glossary: true\nsite:\n  language: zh-CN\n");
  writeFileSync(join(repository, ".repolens/glossary/mediation.yml"), "title: Mediation\nsummary: A layer.\n");
  writeFileSync(join(repository, "README.md"), "# Home\n\nSee [广告聚合](term:mediation).\n\n[Other](docs/other.md)\n");
  writeFileSync(join(repository, "docs/other.md"), "# Other\n\nNo terms here.\n");
  execFileSync("git", ["init", "--quiet", repository]);
  execFileSync("git", ["-C", repository, "config", "user.email", "glossary-test@example.invalid"]);
  execFileSync("git", ["-C", repository, "config", "user.name", "repolens glossary test"]);
  execFileSync("git", ["-C", repository, "add", "."]);
  execFileSync("git", ["-C", repository, "commit", "--quiet", "-m", "test: glossary fixture"]);
  execFileSync("go", ["run", "../../../cmd/repolens", "build", repository, "-o", output]);

  server = createServer((request, response) => {
    const pathname = decodeURIComponent(new URL(request.url ?? "/", "http://127.0.0.1").pathname);
    let filePath = join(output, pathname.replace(/^\/(?:first\/|second\/)?/, ""));
    try {
      if (statSync(filePath).isDirectory()) filePath = join(filePath, "index.html");
      const contentTypes: Record<string, string> = {
        ".css": "text/css",
        ".html": "text/html",
        ".js": "text/javascript",
        ".json": "application/json",
      };
      response.writeHead(200, { "Content-Type": contentTypes[extname(filePath)] ?? "application/octet-stream" });
      createReadStream(filePath).pipe(response);
    } catch {
      response.writeHead(404).end();
    }
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  if (!address || typeof address === "string") throw new Error("static server did not bind to a TCP port");
  siteURL = `http://127.0.0.1:${address.port}`;
});

test.afterAll(async () => {
  await new Promise<void>((resolve, reject) => server.close((error) => (error ? reject(error) : resolve())));
  rmSync(fixtureRoot, { recursive: true, force: true });
});

test("opens the glossary drawer from a term, restores focus on Escape", async ({ page }) => {
  await page.goto(`${siteURL}/view/README.md/`);
  const term = page.locator("a.term").first();
  await expect(term).toHaveAttribute("href", "#glossary-mediation");
  await term.focus();
  await term.click();
  const drawer = page.locator("#glossary-drawer");
  await expect(drawer).toBeVisible();
  await expect(drawer).toHaveAttribute("role", "dialog");
  await expect(drawer).toHaveAttribute("aria-modal", "true");
  await page.keyboard.press("Escape");
  await expect(drawer).toBeHidden();
  await expect(term).toBeFocused();
});

test("anchor navigation works without JavaScript", async ({ browser }) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  const page = await context.newPage();
  await page.goto(`${siteURL}/view/README.md/#glossary-mediation`);
  await expect(page.locator("#glossary-mediation")).toBeVisible();
  await expect(page.locator("#glossary-mediation .glossary-title")).toHaveText("Mediation");
  await expect(page.locator("#btn-glossary")).toHaveCount(1);
  await expect(page.locator("html")).toHaveAttribute("lang", "zh-CN");
  await expect(page.locator("#btn-language")).toBeHidden();
  await context.close();
});

test("pjax navigation closes the drawer and drops previous terms", async ({ page }) => {
  await page.goto(`${siteURL}/view/README.md/`);
  await page.locator("a.term").first().click();
  await expect(page.locator("#glossary-drawer")).toBeVisible();
  await page.locator("#content a[href*='other.md']").evaluate((el: HTMLAnchorElement) => el.click());
  await expect(page).toHaveURL(/docs\/other\.md/);
  await expect(page.locator("#glossary-drawer")).toHaveCount(0);
  await expect(page.locator("a.term")).toHaveCount(0);
});

test("persists interface language through PJAX history and isolates deployments", async ({ page }) => {
  await page.goto(`${siteURL}/first/view/README.md/`);
  const language = page.getByRole("button", { name: "界面语言" });
  await language.click();
  await page.keyboard.press("Escape");
  await expect(language).toBeFocused();
  await language.click();
  await page.getByRole("radio", { name: "English" }).click();

  const term = page.locator("a.term").first();
  await expect(term).toHaveAttribute("aria-label", "广告聚合 (term, view definition)");
  await expect(term).toHaveText("广告聚合");
  await expect(page.getByRole("link", { name: "root", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Interface language" })).toBeFocused();
  await page.evaluate(() => { document.body.dataset.testPjax = "active"; });

  await page.locator("#content a[href*='other.md']").click();
  await expect(page).toHaveURL(/first\/view\/docs\/other\.md/);
  await expect(page.locator("#content h1")).toHaveText("Other ¶");
  await expect(page.locator("#btn-info")).toHaveAttribute("aria-label", "Page information");

  await page.goBack();
  await expect(term).toHaveAttribute("aria-label", "广告聚合 (term, view definition)");
  await page.goForward();
  await expect(page.locator("#content h1")).toHaveText("Other ¶");
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
  await expect(page.locator("body")).toHaveAttribute("data-test-pjax", "active");

  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
  await page.goto(`${siteURL}/second/view/README.md/`);
  await expect(page.locator("html")).toHaveAttribute("lang", "zh-CN");
  await expect(term).toHaveAttribute("aria-label", "广告聚合（术语，查看解释）");
  await expect(page.getByRole("link", { name: "根目录", exact: true })).toBeVisible();
});
