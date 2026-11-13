// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it } from "vitest";
import { useModuleDisplay } from "./use-module-display";

// How a project's modules list is laid out and ordered (P6a spec 2.4, R4): the browser's, by the project, as the old
// pages kept them (useLocalStorage): rows ordered by name until the person chooses otherwise; each project's own; read back
// as the page mounts again (a reload in the same browser); what the browser holds that is no layout or order of the
// list, the default. Each test renders the hook as the pages of a project's modules do: its header and its list, each
// a reader of its own.

type Display = ReturnType<typeof useModuleDisplay>;
const shown: Record<string, Display | undefined> = {};
/** A reader of the project's display, under name (the header, the list). */
function Reader({ name, projectId }: { name: string; projectId: string }) {
  shown[name] = useModuleDisplay(projectId);
  return null;
}

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  localStorage.clear();
  for (const name of Object.keys(shown)) delete shown[name];
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

/** The readers, each of its project, in a page. */
const render = (readers: Record<string, string>) =>
  act(async () =>
    root.render(
      Object.entries(readers).map(([name, projectId]) => <Reader key={name} name={name} projectId={projectId} />)
    )
  );
const seen = (name: string) => ({ layout: shown[name]?.layout, orderBy: shown[name]?.orderBy });
/** The key under which the browser keeps value, a choice the person made. */
const keptUnder = (value: string) => {
  const keys = Array.from({ length: localStorage.length }, (_, index) => localStorage.key(index) ?? "");
  return keys.find((key) => localStorage.getItem(key) === JSON.stringify(value));
};

it("lays a project's modules out as rows ordered by name, until the person chooses otherwise", async () => {
  await render({ list: "p-web" });
  expect(seen("list")).toEqual({ layout: "list", orderBy: "name" });
});

it("gives every reader of the project what one of them chose: the list follows its header", async () => {
  await render({ header: "p-web", list: "p-web" });
  await act(async () => shown.header?.setLayout("board"));
  await act(async () => shown.header?.setOrderBy("-progress"));
  expect(seen("list")).toEqual({ layout: "board", orderBy: "-progress" });
});

it("keeps each project's apart: another project's modules are laid out and ordered as chosen for it", async () => {
  await render({ web: "p-web", ops: "p-ops" });
  await act(async () => shown.web?.setLayout("board"));
  await act(async () => shown.web?.setOrderBy("target_date"));
  expect(seen("web")).toEqual({ layout: "board", orderBy: "target_date" });
  expect(seen("ops")).toEqual({ layout: "list", orderBy: "name" });
  await act(async () => shown.ops?.setOrderBy("-created_at"));
  expect(seen("web")).toEqual({ layout: "board", orderBy: "target_date" });
  expect(seen("ops")).toEqual({ layout: "list", orderBy: "-created_at" });
});

it("reads back what the browser kept for the project as the page mounts again, as a reload does", async () => {
  await render({ list: "p-web" });
  await act(async () => shown.list?.setLayout("board"));
  await act(async () => shown.list?.setOrderBy("-issues_length"));
  await act(async () => root.unmount());
  delete shown.list;
  root = createRoot(host);
  await render({ list: "p-web" });
  expect(seen("list")).toEqual({ layout: "board", orderBy: "-issues_length" });
});

it.each([
  { what: "layout", chosen: "board", kept: "calendar" },
  { what: "layout", chosen: "board", kept: 2 },
  { what: "order", chosen: "progress", kept: "-sort_order" },
  { what: "order", chosen: "progress", kept: "updated_at" },
  { what: "order", chosen: "progress", kept: null },
])("takes the default $what for $kept, which the browser keeps and is none of the list's", async (one) => {
  await render({ list: "p-web" });
  await act(async () => (one.what === "layout" ? shown.list?.setLayout("board") : shown.list?.setOrderBy("progress")));
  const key = keptUnder(one.chosen);
  expect(key).toContain("p-web");
  localStorage.setItem(key ?? "", JSON.stringify(one.kept));
  await act(async () => root.unmount());
  root = createRoot(host);
  await render({ list: "p-web" });
  expect(seen("list")).toEqual({ layout: "list", orderBy: "name" });
});
