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
let leftOutput = "";
let rightOutput = "";

test.beforeAll(async () => {
  fixtureRoot = mkdtempSync(join(tmpdir(), "repolens-tree-e2e-"));
  const repository = join(fixtureRoot, "repository");
  leftOutput = join(fixtureRoot, "dist-left");
  rightOutput = join(fixtureRoot, "dist-right");
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
  execFileSync("go", ["run", "../../../cmd/repolens", "build", repository, "-o", leftOutput]);
  writeFileSync(join(repository, ".repolens.yml"), "site:\n  language: en\nview:\n  tree_position: right\n");
  execFileSync("git", ["-C", repository, "add", ".repolens.yml"]);
  execFileSync("git", ["-C", repository, "commit", "--quiet", "-m", "test: move tree to the right"]);
  execFileSync("go", ["run", "../../../cmd/repolens", "build", repository, "-o", rightOutput]);

  server = createServer((request, response) => {
    const pathname = decodeURIComponent(new URL(request.url ?? "/", "http://127.0.0.1").pathname);
    const right = pathname === "/right" || pathname.startsWith("/right/");
    const docs = pathname === "/docs" || pathname.startsWith("/docs/");
    const relativePath = (right
      ? pathname.slice("/right".length)
      : docs
        ? pathname.slice("/docs".length)
        : pathname).replace(/^\/+/, "");
    let filePath = join(right ? rightOutput : leftOutput, relativePath);
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
  await expect(page.getByRole("separator", { name: "Resize repository tree" })).toBeHidden();
  await expect.poll(async () => Math.round((await page.locator(".sidebar").boundingBox())?.width ?? 0)).toBe(300);
  await context.close();
});

test("mirrors fixed and floating trees for left and right layouts", async ({ page }) => {
  for (const fixture of [
    { prefix: "", position: "left", fixedAtStart: true },
    { prefix: "/right", position: "right", fixedAtStart: false },
  ]) {
    await page.goto(`${siteURL}${fixture.prefix}/view/${currentPath}/`);
    await page.evaluate(() => window.localStorage.setItem("repolens:tree:preference", "expanded"));
    await page.reload();
    await expect(page.locator("body")).toHaveAttribute("data-tree-position", fixture.position);

    const sidebar = page.locator(".sidebar");
    const content = page.locator("#content");
    const sidebarBox = await sidebar.boundingBox();
    const contentBox = await content.boundingBox();
    if (!sidebarBox || !contentBox) throw new Error("fixed layout boxes are unavailable");
    expect(sidebarBox.x < contentBox.x).toBe(fixture.fixedAtStart);

    const border = await sidebar.evaluate((element, position) => {
      const style = getComputedStyle(element);
      return position === "left" ? style.borderRightWidth : style.borderLeftWidth;
    }, fixture.position);
    expect(border).toBe("1px");

    await page.locator("#btn-tree").click();
    await page.locator("#btn-tree").click();
    await expect(page.locator("body")).toHaveAttribute("data-overlay", "open");
    if (fixture.position === "left") {
      await expect.poll(async () => Math.round((await page.locator("#tree-overlay").boundingBox())?.x ?? -1)).toBe(0);
    } else {
      await expect.poll(async () => {
        const box = await page.locator("#tree-overlay").boundingBox();
        return box ? Math.round(box.x + box.width) : -1;
      }).toBe(await page.evaluate(() => window.innerWidth));
    }

    await page.locator("#overlay-tree").getByRole("link", { name: "README.md", exact: true }).click();
    await expect(page).toHaveURL(`${siteURL}${fixture.prefix}/view/README.md/`);
    await expect(page.locator("body")).toHaveAttribute("data-tree-position", fixture.position);
  }
});

test("keeps the right-side static tree readable without JavaScript", async ({ browser }) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  const page = await context.newPage();
  await page.goto(`${siteURL}/right/view/${currentPath}/`);
  await expect(page.locator("body")).toHaveAttribute("data-tree-position", "right");
  await expect(page.locator("#tree-src").getByRole("link", { name: currentPath, exact: true })).toBeVisible();
  await context.close();
});

test("resizes the left fixed tree with pointer and keyboard controls", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto(`${siteURL}/view/${currentPath}/`);
  await page.evaluate(() => {
    window.localStorage.removeItem("repolens:sidebar-width:v1:/");
    window.localStorage.setItem("repolens:tree:preference", "expanded");
  });
  await page.reload();

  const resizer = page.getByRole("separator", { name: "Resize repository tree" });
  const sidebar = page.locator(".sidebar");
  const width = async (): Promise<number> => Math.round((await sidebar.boundingBox())?.width ?? 0);
  await expect(resizer).toBeVisible();
  await expect(resizer).toHaveAttribute("aria-valuemin", "220");
  await expect(resizer).toHaveAttribute("aria-valuemax", "520");
  await expect(resizer).toHaveAttribute("aria-valuenow", "300");
  await expect.poll(width).toBe(300);

  await resizer.focus();
  await page.keyboard.press("ArrowRight");
  await expect.poll(width).toBe(308);
  await page.keyboard.press("Shift+ArrowRight");
  await expect.poll(width).toBe(340);
  await page.keyboard.press("Home");
  await expect.poll(width).toBe(220);
  await page.keyboard.press("End");
  await expect.poll(width).toBe(520);
  await page.keyboard.press("Enter");
  await expect.poll(width).toBe(300);
  expect(await page.evaluate(() => window.localStorage.getItem("repolens:sidebar-width:v1:/"))).toBeNull();

  const box = await resizer.boundingBox();
  if (!box) throw new Error("resizer box is unavailable");
  await page.mouse.move(box.x + box.width / 2, box.y + 80);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width / 2 + 68, box.y + 80);
  await expect.poll(width).toBe(368);
  expect(await page.evaluate(() => window.localStorage.getItem("repolens:sidebar-width:v1:/"))).toBeNull();
  await page.mouse.up();
  expect(await page.evaluate(() => window.localStorage.getItem("repolens:sidebar-width:v1:/"))).toBe("368");

  await page.locator("#tree-src").getByRole("link", { name: "README.md", exact: true }).click();
  await expect(page).toHaveURL(`${siteURL}/view/README.md/`);
  await expect.poll(width).toBe(368);
  await resizer.focus();
  await page.keyboard.press("ArrowRight");
  await expect.poll(width).toBe(376);

  await page.locator("#btn-tree").click();
  await expect(resizer).toBeHidden();
  await page.locator("#btn-tree").click();
  await page.locator("#btn-pin-tree").click();
  await expect(resizer).toBeVisible();
  await expect.poll(width).toBe(376);
  await resizer.dblclick();
  await expect.poll(width).toBe(300);
});

test("cancels pointer drags without committing intermediate widths", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto(`${siteURL}/view/${currentPath}/`);
  await page.evaluate(() => {
    window.localStorage.setItem("repolens:tree:preference", "expanded");
    window.localStorage.setItem("repolens:sidebar-width:v1:/", "360");
  });
  await page.reload();

  const resizer = page.getByRole("separator", { name: "Resize repository tree" });
  const sidebar = page.locator(".sidebar");
  const width = async (): Promise<number> => Math.round((await sidebar.boundingBox())?.width ?? 0);
  const box = await resizer.boundingBox();
  if (!box) throw new Error("resizer box is unavailable");
  const startX = box.x + box.width / 2;

  await resizer.dispatchEvent("pointerdown", {
    pointerId: 31, pointerType: "pen", isPrimary: true, button: 0, clientX: startX, clientY: box.y + 80,
  });
  await resizer.dispatchEvent("pointermove", {
    pointerId: 31, pointerType: "pen", isPrimary: true, buttons: 1, clientX: startX + 40, clientY: box.y + 80,
  });
  await expect.poll(width).toBe(400);
  await resizer.dispatchEvent("pointercancel", {
    pointerId: 31, pointerType: "pen", isPrimary: true, clientX: startX + 40, clientY: box.y + 80,
  });
  await expect.poll(width).toBe(360);
  expect(await page.evaluate(() => window.localStorage.getItem("repolens:sidebar-width:v1:/"))).toBe("360");

  await resizer.dispatchEvent("pointerdown", {
    pointerId: 32, pointerType: "touch", isPrimary: true, button: 0, clientX: startX, clientY: box.y + 120,
  });
  await resizer.dispatchEvent("pointermove", {
    pointerId: 32, pointerType: "touch", isPrimary: true, buttons: 1, clientX: startX + 32, clientY: box.y + 120,
  });
  await page.keyboard.press("Escape");
  await expect.poll(width).toBe(360);
  expect(await page.evaluate(() => window.localStorage.getItem("repolens:sidebar-width:v1:/"))).toBe("360");

  await resizer.dispatchEvent("pointerdown", {
    pointerId: 33, pointerType: "pen", isPrimary: true, button: 0, clientX: startX, clientY: box.y + 160,
  });
  await resizer.dispatchEvent("pointermove", {
    pointerId: 33, pointerType: "pen", isPrimary: true, buttons: 1, clientX: startX + 16, clientY: box.y + 160,
  });
  await resizer.dispatchEvent("pointerup", {
    pointerId: 33, pointerType: "pen", isPrimary: true, button: 0, clientX: startX + 16, clientY: box.y + 160,
  });
  await expect.poll(width).toBe(376);
  expect(await page.evaluate(() => window.localStorage.getItem("repolens:sidebar-width:v1:/"))).toBe("376");

  await resizer.dispatchEvent("pointerdown", {
    pointerId: 34, pointerType: "touch", isPrimary: true, button: 0, clientX: startX, clientY: box.y + 200,
  });
  await resizer.dispatchEvent("pointermove", {
    pointerId: 34, pointerType: "touch", isPrimary: true, buttons: 1, clientX: startX + 24, clientY: box.y + 200,
  });
  await resizer.dispatchEvent("pointerup", {
    pointerId: 34, pointerType: "touch", isPrimary: true, button: 0, clientX: startX + 24, clientY: box.y + 200,
  });
  await expect.poll(width).toBe(400);
  expect(await page.evaluate(() => window.localStorage.getItem("repolens:sidebar-width:v1:/"))).toBe("400");
});

test("isolates widths by base path and safely handles invalid storage and defaults", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto(`${siteURL}/view/${currentPath}/`);
  await page.evaluate(() => {
    window.localStorage.setItem("repolens:tree:preference", "expanded");
    window.localStorage.setItem("repolens:sidebar-width:v1:/", "360");
    window.localStorage.setItem("repolens:sidebar-width:v1:/docs/", "420");
  });
  await page.reload();
  await expect.poll(async () => Math.round((await page.locator(".sidebar").boundingBox())?.width ?? 0)).toBe(360);

  await page.goto(`${siteURL}/docs/view/${currentPath}/`);
  await expect.poll(async () => Math.round((await page.locator(".sidebar").boundingBox())?.width ?? 0)).toBe(420);

  await page.goto(`${siteURL}/view/${currentPath}/`);
  await page.evaluate(() => window.localStorage.setItem("repolens:sidebar-width:v1:/", "invalid"));
  await page.reload();
  await expect.poll(async () => Math.round((await page.locator(".sidebar").boundingBox())?.width ?? 0)).toBe(300);

  await page.evaluate(() => {
    document.documentElement.style.setProperty("--sidebar-width", "invalid");
    window.dispatchEvent(new Event("resize"));
  });
  await expect.poll(async () => Math.round((await page.locator(".sidebar").boundingBox())?.width ?? 0)).toBe(300);

  await page.evaluate(() => {
    document.documentElement.style.removeProperty("--sidebar-width");
    window.localStorage.setItem("repolens:sidebar-width:v1:/", "999");
  });
  await page.reload();
  await expect.poll(async () => Math.round((await page.locator(".sidebar").boundingBox())?.width ?? 0)).toBe(520);
  await page.setViewportSize({ width: 1024, height: 900 });
  await expect.poll(async () => Math.round((await page.locator(".sidebar").boundingBox())?.width ?? 0)).toBe(460);
  expect(await page.evaluate(() => window.localStorage.getItem("repolens:sidebar-width:v1:/"))).toBe("999");
  await page.setViewportSize({ width: 1440, height: 900 });
  await expect.poll(async () => Math.round((await page.locator(".sidebar").boundingBox())?.width ?? 0)).toBe(520);
});

test("mirrors keyboard direction and preserves width across responsive modes", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto(`${siteURL}/right/view/${currentPath}/`);
  await page.evaluate(() => {
    window.localStorage.setItem("repolens:tree:preference", "expanded");
    window.localStorage.removeItem("repolens:sidebar-width:v1:/right/");
  });
  await page.reload();

  const resizer = page.getByRole("separator", { name: "Resize repository tree" });
  const sidebar = page.locator(".sidebar");
  const width = async (): Promise<number> => Math.round((await sidebar.boundingBox())?.width ?? 0);
  await resizer.focus();
  await page.keyboard.press("ArrowRight");
  await expect.poll(width).toBe(292);
  await page.keyboard.press("ArrowLeft");
  await expect.poll(width).toBe(300);
  await page.keyboard.press("Shift+ArrowLeft");
  await expect.poll(width).toBe(332);
  expect(await page.locator(".shell").evaluate((element) => getComputedStyle(element).transitionDuration)).toBe("0s");

  const rightBox = await resizer.boundingBox();
  if (!rightBox) throw new Error("right resizer box is unavailable");
  await page.mouse.move(rightBox.x + rightBox.width / 2, rightBox.y + 80);
  await page.mouse.down();
  await page.mouse.move(rightBox.x + rightBox.width / 2 - 32, rightBox.y + 80);
  await page.mouse.up();
  await expect.poll(width).toBe(364);

  await page.setViewportSize({ width: 720, height: 900 });
  await expect(resizer).toBeHidden();
  await expect(page.locator("#btn-tree")).toBeFocused();
  await expect(page.locator("body")).toHaveAttribute("data-tree-mode", "floating");
  await page.locator("#btn-tree").click();
  await expect(page.locator("#tree-overlay")).toBeVisible();
  await expect(page.locator("#sidebar-resizer")).toBeHidden();
  const zoomOverflow = await page.evaluate(() => ({
    innerWidth: window.innerWidth,
    scrollWidth: document.documentElement.scrollWidth,
    offenders: Array.from(document.querySelectorAll<HTMLElement>("body *"))
      .filter((element) => {
        const rect = element.getBoundingClientRect();
        return rect.right > window.innerWidth + 1 || rect.left < -1;
      })
      .slice(0, 12)
      .map((element) => ({
        className: element.className,
        id: element.id,
        left: element.getBoundingClientRect().left,
        right: element.getBoundingClientRect().right,
      })),
  }));
  expect(zoomOverflow.scrollWidth, JSON.stringify(zoomOverflow.offenders)).toBeLessThanOrEqual(zoomOverflow.innerWidth);

  await page.setViewportSize({ width: 390, height: 844 });
  await expect.poll(async () => Math.round((await page.locator("#tree-overlay").boundingBox())?.width ?? 0)).toBe(320);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.locator("#scrim").click({ position: { x: 10, y: 400 } });

  await page.setViewportSize({ width: 1440, height: 900 });
  await expect(resizer).toBeVisible();
  await expect.poll(width).toBe(364);
});

test("falls back when localStorage is unavailable", async ({ browser }) => {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  await context.addInitScript(() => {
    const getItem = Storage.prototype.getItem;
    const setItem = Storage.prototype.setItem;
    const removeItem = Storage.prototype.removeItem;
    Storage.prototype.getItem = function (key: string): string | null {
      if (key.startsWith("repolens:sidebar-width:")) throw new Error("storage unavailable");
      return getItem.call(this, key);
    };
    Storage.prototype.setItem = function (key: string, value: string): void {
      if (key.startsWith("repolens:sidebar-width:")) throw new Error("storage unavailable");
      setItem.call(this, key, value);
    };
    Storage.prototype.removeItem = function (key: string): void {
      if (key.startsWith("repolens:sidebar-width:")) throw new Error("storage unavailable");
      removeItem.call(this, key);
    };
  });
  const page = await context.newPage();
  await page.goto(`${siteURL}/view/${currentPath}/`);
  await expect.poll(async () => Math.round((await page.locator(".sidebar").boundingBox())?.width ?? 0)).toBe(300);
  const resizer = page.getByRole("separator", { name: "Resize repository tree" });
  await resizer.focus();
  await page.keyboard.press("ArrowRight");
  await expect.poll(async () => Math.round((await page.locator(".sidebar").boundingBox())?.width ?? 0)).toBe(308);
  await context.close();
});
