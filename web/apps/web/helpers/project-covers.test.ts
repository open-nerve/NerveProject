/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { existsSync, readFileSync } from "node:fs";
import { expect, it } from "vitest";
import { PROJECT_COVERS, projectCoverURL } from "./project-covers";

it("has exactly the 29 stable IDs, each resolving to its own bundled local SVG", () => {
  const ids = Array.from({ length: 29 }, (_, i) => `cover-${String(i + 1).padStart(2, "0")}`);
  expect(PROJECT_COVERS.map(({ id }) => id)).toEqual(ids);
  expect(new Set(PROJECT_COVERS.map(({ url }) => url)).size).toBe(29);
  for (const { id, url } of PROJECT_COVERS) {
    expect(projectCoverURL(id)).toBe(url);
    // Vitest inlines small local SVGs; the application build emits files (assetsInlineLimit: 0).
    if (!url.startsWith("data:image/svg+xml,")) expect(url).toContain(`${id}.svg`);
    expect(url).not.toMatch(/^https?:/);
    const path = new URL(`../app/assets/project-covers/${id}.svg`, import.meta.url);
    expect(existsSync(path), id).toBe(true);
    const svg = readFileSync(path, "utf8");
    expect(svg).toContain('viewBox="0 0 1600 900"');
    expect(svg).toContain("Copyright (c) 2026-present OpenNerve");
    expect(svg).not.toMatch(/<script|<image|<foreignObject|\bon\w+=|(?:href|src)=|url\((?!#)/i);
  }
});

it.each([null, undefined, "", "cover-00", "cover-30", "cover-1", "toString", "__proto__", "/assets/cover-01.svg"])(
  "unknown or absent preset %s has no random or external fallback",
  (id) => expect(projectCoverURL(id)).toBeUndefined()
);
