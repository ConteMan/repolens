import { execFileSync } from "node:child_process";
import { createReadStream, mkdtempSync, mkdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { createServer, type Server } from "node:http";
import { tmpdir } from "node:os";
import { extname, join } from "node:path";
import { expect, test } from "@playwright/test";

const currentPath = "docs/deeply/nested/a-very-long-file-name-that-needs-a-tooltip.md";
let fixtureRoot = "";
let siteURL = "";
let server: Server;

test.beforeAll(async () => {
  fixtureRoot = mkdtempSync(join(tmpdir(), "repolens-tree-e2e-"));
  const repository = join(fixtureRoot, "repository");
  const output = join(fixtureRoot, "dist");
  mkdirSync(join(repository, "docs", "deeply", "nested"), { recursive: true });
  writeFileSync(join(repository, "README.md"), "# Home\n");
  writeFileSync(join(repository, currentPath), "# Current file\n\nCurrent content.\n");
  writeFileSync(join(repository, "docs", "other.md"), "# Other file\n");
  for (let index = 0; index < 64; index += 1) {
    const section = join(repository, "docs", "generated", `section-${String(index).padStart(2, "0")}`);
    mkdirSync(section, { recursive: true });
    writeFileSync(join(section, "README.md"), `# Generated section ${index}\n`);
  }
  writeFileSync(join(repository, ".repolens.yml"), "site:\n  language: en\n");
  execFileSync("git", ["init", "--quiet", repository]);
  execFileSync("git", ["-C", repository, "config", "user.email", "tree-test@example.invalid"]);
  execFileSync("git", ["-C", repository, "config", "user.name", "repolens tree test"]);
  execFileSync("git", ["-C", repository, "add", "."]);
  execFileSync("git", ["-C", repository, "commit", "--quiet", "-m", "test: add tree fixture"]);
  execFileSync("go", ["run", "../../../cmd/repolens", "build", repository, "-o", output]);

  server = createServer((request, response) => {
    const pathname = decodeURIComponent(new URL(request.url ?? "/", "http://127.0.0.1").pathname);
    let filePath = join(output, pathname.replace(/^\/+/, ""));
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

test("highlights and locates the default home file", async ({ page }) => {
  await page.goto(`${siteURL}/view/`);

  const fixedTree = page.locator("#tree-src");
  const current = fixedTree.locator("li.current");
  await expect(current).toHaveCount(1);
  await expect(current.getByRole("link", { name: "README.md", exact: true })).toHaveAttribute("title", "README.md");

  await fixedTree.getByRole("button", { name: "Collapse all" }).click();
  await fixedTree.getByRole("button", { name: "Locate current file" }).click();
  await expect.poll(() => current.evaluate((item) => {
    const itemRect = item.getBoundingClientRect();
    const containerRect = item.closest("[data-tree-scroll]")?.getBoundingClientRect();
    return !!containerRect && itemRect.top >= containerRect.top && itemRect.bottom <= containerRect.bottom;
  })).toBe(true);
});

test("keeps bulk tree actions synchronized and locates the current file", async ({ page }) => {
  await page.addInitScript(() => {
    const original = Storage.prototype.setItem;
    (window as typeof window & { __treeWrites: Record<string, number> }).__treeWrites = {};
    Storage.prototype.setItem = function (key: string, value: string): void {
      if (this === window.sessionStorage && key.startsWith("repolens:tree:") && key !== "repolens:tree:scroll") {
        const writes = (window as typeof window & { __treeWrites: Record<string, number> }).__treeWrites;
        writes[key] = (writes[key] ?? 0) + 1;
      }
      original.call(this, key, value);
    };
  });
  await page.goto(`${siteURL}/view/${currentPath}/`);

  const fixedTree = page.locator("#tree-src");
  const overlayTree = page.locator("#overlay-tree");
  await expect(fixedTree.getByRole("group", { name: "Repository tree actions" })).toBeVisible();
  await expect(fixedTree.getByRole("button", { name: "Expand all" })).toBeVisible();
  await expect(fixedTree.getByRole("button", { name: "Collapse all" })).toBeVisible();
  await expect(fixedTree.getByRole("button", { name: "Locate current file" })).toBeVisible();
  await expect(fixedTree.getByRole("link", { name: currentPath, exact: true })).toHaveAttribute("title", currentPath);
  await expect(page.locator(".tb-file")).toHaveAttribute("title", currentPath);

  const fixedScroll = fixedTree.locator(":scope > .tree");
  const fixedActions = fixedTree.getByRole("group", { name: "Repository tree actions" });
  const fixedSearch = fixedTree.locator(".tree-search");
  const fixedActionsTop = (await fixedActions.boundingBox())?.y;
  const fixedSearchTop = (await fixedSearch.boundingBox())?.y;
  await fixedScroll.evaluate((tree) => {
    tree.scrollTop = tree.scrollHeight;
  });
  await expect.poll(() => fixedScroll.evaluate((tree) => tree.scrollTop)).toBeGreaterThan(0);
  await expect.poll(async () => (await fixedActions.boundingBox())?.y).toBe(fixedActionsTop);
  await expect.poll(async () => (await fixedSearch.boundingBox())?.y).toBe(fixedSearchTop);
  await expect(fixedScroll).toHaveAttribute("data-tree-scroll", "");
  await expect(page.locator(".sidebar")).not.toHaveAttribute("data-tree-scroll", "");

  const uniquePathCount = await page.locator("details[data-tree-path]").evaluateAll((details) => (
    new Set(details.map((detail) => detail.getAttribute("data-tree-path"))).size
  ));
  await page.evaluate(() => {
    (window as typeof window & { __treeWrites: Record<string, number> }).__treeWrites = {};
  });
  await fixedTree.getByRole("button", { name: "Collapse all" }).click();
  await expect.poll(() => page.locator("details[open]").count()).toBe(0);
  await expect.poll(() => page.evaluate(() => Object.values(
    (window as typeof window & { __treeWrites: Record<string, number> }).__treeWrites,
  ).reduce((total, count) => total + count, 0))).toBe(uniquePathCount);

  await page.locator("#btn-tree").click();
  await page.locator("#btn-tree").click();
  await expect(page.locator("body")).toHaveAttribute("data-overlay", "open");
  await page.evaluate(() => {
    (window as typeof window & { __treeWrites: Record<string, number> }).__treeWrites = {};
  });
  await overlayTree.getByRole("button", { name: "Expand all" }).click();
  const detailCount = await page.locator("details[data-tree-path]").count();
  await expect.poll(() => page.locator("details[open]").count()).toBe(detailCount);
  await expect.poll(() => page.evaluate(() => Object.values(
    (window as typeof window & { __treeWrites: Record<string, number> }).__treeWrites,
  ).reduce((total, count) => total + count, 0))).toBe(uniquePathCount);

  const overlayScroll = overlayTree.locator(":scope > .tree");
  const overlayActions = overlayTree.getByRole("group", { name: "Repository tree actions" });
  const overlaySearch = overlayTree.locator(".tree-search");
  const overlayActionsTop = (await overlayActions.boundingBox())?.y;
  const overlaySearchTop = (await overlaySearch.boundingBox())?.y;
  await overlayScroll.evaluate((tree) => {
    tree.scrollTop = tree.scrollHeight;
  });
  await expect.poll(() => overlayScroll.evaluate((tree) => tree.scrollTop)).toBeGreaterThan(0);
  await expect.poll(async () => (await overlayActions.boundingBox())?.y).toBe(overlayActionsTop);
  await expect.poll(async () => (await overlaySearch.boundingBox())?.y).toBe(overlaySearchTop);
  await expect(overlayScroll).toHaveAttribute("data-tree-scroll", "");
  await expect(overlayTree).not.toHaveAttribute("data-tree-scroll", "");

  await overlayTree.getByRole("button", { name: "Collapse all" }).click();
  const ancestorPaths = await overlayTree.locator("li.current").evaluate((current) => {
    const paths: string[] = [];
    let node = current.parentElement;
    while (node) {
      if (node.tagName === "DETAILS" && node.hasAttribute("data-tree-path")) {
        paths.push(node.getAttribute("data-tree-path") ?? "");
      }
      node = node.parentElement;
    }
    return paths;
  });
  await overlayTree.getByRole("button", { name: "Locate current file" }).click();
  for (const path of ancestorPaths) {
    await expect(page.locator(`details[data-tree-path="${path}"][open]`)).toHaveCount(2);
  }
  await expect.poll(() => overlayTree.locator("li.current").evaluate((current) => {
    const item = current.getBoundingClientRect();
    const container = current.closest("[data-tree-scroll]")?.getBoundingClientRect();
    return !!container && item.top >= container.top && item.bottom <= container.bottom;
  })).toBe(true);

  await overlayTree.getByRole("link", { name: "README.md", exact: true }).click();
  await expect(page).toHaveURL(`${siteURL}/view/README.md/`);
  await expect(page.locator('#tree-src [data-tree-action="locate"]')).toHaveCount(1);
  await page.locator("#btn-tree").click();
  await expect(page.locator("body")).toHaveAttribute("data-overlay", "open");
  await expect(page.locator("#overlay-tree").getByRole("button", { name: "Locate current file" })).toBeVisible();
});

test("keeps the generated tree navigable without JavaScript", async ({ browser }) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  const page = await context.newPage();
  await page.goto(`${siteURL}/view/${currentPath}/`);
  await expect(page.locator("#tree-src").getByRole("link", { name: currentPath, exact: true })).toBeVisible();
  await expect(page.locator(".tree-actions")).toBeHidden();
  await context.close();
});
