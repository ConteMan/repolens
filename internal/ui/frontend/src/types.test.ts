import assert from "node:assert/strict";
import test from "node:test";
import { normalizeSettings } from "./types.ts";

test("normalizeSettings preserves absent list fields", () => {
  const settings = normalizeSettings({ site: { title: "Repository" } });

  assert.equal(settings.ignore, null);
  assert.equal(settings.rules, null);
  assert.equal(settings.site.title, "Repository");
});

test("normalizeSettings preserves explicit empty list fields", () => {
  const settings = normalizeSettings({ ignore: [], rules: [] });

  assert.deepEqual(settings.ignore, []);
  assert.deepEqual(settings.rules, []);
});
