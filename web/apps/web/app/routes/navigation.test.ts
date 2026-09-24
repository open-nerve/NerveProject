/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { matchRoutes } from "react-router";
import type { RouteObject } from "react-router";
import ts from "typescript";
import { describe, expect, it } from "vitest";
import type { RouteConfigEntry } from "@react-router/dev/routes";
import {
  PROFILE_TABS,
  PROJECT_SETTINGS,
  WORKSPACE_SETTINGS,
  WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS,
  WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS,
} from "@plane/constants";
import { generateWorkItemLink, joinUrlPath } from "@plane/utils";
import routes from "../routes";

// Every internal navigation has to land on a page, not on "page not found" (M1 design 4.2): Plane's legacy
// address redirects are gone (3.12), so nothing may lean on them. The check reads the real route table and
// the web app's own source, so it follows both as they change.

// The route table as React Router matches it; the catch-all "*" is "page not found".
const toRouteObjects = (entries: RouteConfigEntry[]): RouteObject[] =>
  entries.map((entry) =>
    entry.index ? { index: true } : { path: entry.path, children: entry.children && toRouteObjects(entry.children) }
  );
const table = toRouteObjects(routes);

/** Whether React Router renders a page for this concrete path. */
const lands = (url: string) => {
  const leaf = matchRoutes(table, url)?.at(-1);
  return leaf !== undefined && leaf.route.path !== "*";
};

/** The full path of every page, such as "/:workspaceSlug/projects/:projectId/issues". */
const pagePaths = (entries: RouteConfigEntry[], parent = ""): string[] =>
  entries.flatMap((entry) => {
    const own = entry.path ? `${parent}/${entry.path}` : parent;
    return entry.children?.length ? pagePaths(entry.children, own) : [own || "/"];
  });
const pages = pagePaths(routes);

const escapeRegExp = (text: string) => text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

/**
 * Whether a path written in the source lands on a page. A "*" in it is a `${…}` of the source: it stands for any
 * text within one segment, so it is filled in from each page's path in turn and the result is matched.
 */
const reaches = (target: string) => {
  if (!target.includes("*")) return lands(target);
  const segments = target.replace(/(.)\/$/, "$1").split("/");
  return pages.some((page) => {
    const own = page.split("/");
    if (own.length !== segments.length) return false;
    const filled = segments.map((segment, i) => {
      if (!segment.includes("*")) return segment;
      if (own[i].startsWith(":")) return segment.replaceAll("*", "x");
      const pattern = new RegExp(`^${segment.split("*").map(escapeRegExp).join(".*")}$`);
      return pattern.test(own[i]) ? own[i] : segment;
    });
    return lands(filled.join("/"));
  });
};

type TTarget = { at: string; path: string };

const appDirectory = fileURLToPath(new URL("../..", import.meta.url));

/**
 * The in-app paths written in one source file: string and template literals that start with "/" (API paths and
 * files left out), a `${prefix}/…` template whose prefix is such a constant of the same file, and the segments of
 * each command palette navigation. Every `${…}` is written "*".
 */
const targetsIn = (file: string): TTarget[] => {
  const source = ts.createSourceFile(
    file,
    readFileSync(join(appDirectory, file), "utf8"),
    ts.ScriptTarget.Latest,
    true
  );
  const prefixes = new Map<string, string>();
  const text = (node: ts.Node): string | undefined => {
    if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) return node.text;
    if (!ts.isTemplateExpression(node)) return undefined;
    const [first] = node.templateSpans;
    const prefix = node.head.text === "" && ts.isIdentifier(first.expression) && prefixes.get(first.expression.text);
    const spans = node.templateSpans.map((span, i) => (i === 0 && prefix ? prefix : "*") + span.literal.text);
    return node.head.text + spans.join("");
  };
  const found: TTarget[] = [];
  const add = (node: ts.Node, written: string) => {
    const path = written.split(/[?#]/)[0];
    if (!path.startsWith("/") || /\s/.test(path) || /^\/(api|auth)\//.test(path) || /\.\w+$/.test(path)) return;
    found.push({ at: `${file}:${source.getLineAndCharacterOfPosition(node.getStart(source)).line + 1}`, path });
  };
  const visit = (node: ts.Node): void => {
    if (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) return;
    const written = text(node);
    if (written !== undefined) return add(node, written);
    if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name) && node.initializer) {
      const value = text(node.initializer);
      if (value?.startsWith("/")) prefixes.set(node.name.text, value);
    }
    if (ts.isCallExpression(node) && node.expression.getText(source) === "handlePowerKNavigate") {
      const segments = node.arguments[1];
      if (segments && ts.isArrayLiteralExpression(segments))
        add(node, `/${segments.elements.map((e) => (ts.isStringLiteral(e) ? e.text : "*")).join("/")}`);
    }
    ts.forEachChild(node, visit);
  };
  visit(source);
  return found;
};

const sourceFiles = ["app", "core", "helpers"].flatMap((directory) =>
  readdirSync(join(appDirectory, directory), { recursive: true, encoding: "utf8" })
    .filter((file) => /\.tsx?$/.test(file) && !/\.(test|d)\.ts$/.test(file))
    .map((file) => join(directory, file))
);

describe("internal navigation", () => {
  const targets = sourceFiles.flatMap(targetsIn);

  it("finds the paths written in the web app", () => {
    // a scan that silently found nothing would let everything below pass
    expect(targets.length).toBeGreaterThan(200);
  });

  it("lands every path written in the web app on a page", () => {
    expect(targets.filter((target) => !reaches(target.path))).toEqual([]);
  });

  it("writes every in-app path without a trailing slash", () => {
    // the web app's own addresses never end with "/" (M1 design 4.1)
    expect(targets.filter((target) => target.path.length > 1 && target.path.endsWith("/"))).toEqual([]);
  });

  it("lands every entry of the navigation constants on a page, without a trailing slash", () => {
    const slug = "acme";
    const entries = [
      // the sidebar joins its items to the workspace, and "your work" to the user too (workspace/sidebar/sidebar-item.tsx)
      ...[...WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS, ...WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS].map((item) =>
        item.key === "your_work" ? joinUrlPath(slug, item.href, "u1") : joinUrlPath(slug, item.href)
      ),
      // settings/workspace/sidebar/item-categories.tsx and settings/project/sidebar/item-categories.tsx
      ...Object.values(WORKSPACE_SETTINGS).map((item) => joinUrlPath(slug, item.href)),
      ...Object.values(PROJECT_SETTINGS).map((item) => `/${slug}/settings/projects/p1${item.href}`),
      // the profile page's tabs (profile/[userId]/navbar.tsx)
      ...PROFILE_TABS.map((tab) => `/${slug}/profile/u1/${tab.route}`),
      ...[false, true].map((isArchived) =>
        generateWorkItemLink({
          workspaceSlug: slug,
          projectId: "p1",
          issueId: "i1",
          projectIdentifier: "WEB",
          sequenceId: 1,
          isArchived,
        })
      ),
    ];
    expect(entries.filter((url) => !lands(url))).toEqual([]);
    expect(entries.filter((url) => url.endsWith("/"))).toEqual([]);
  });

  it("sends a path that no page serves to page not found", () => {
    // Plane's old account settings address, whose redirect is gone
    expect(lands("/acme/settings/account")).toBe(false);
    // what the disabled feature pages used to link to
    expect(reaches("/*/settings/projects/*/features")).toBe(false);
  });
});
