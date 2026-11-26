// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The command palette's menus whose items' names need not be unique (a project's cycles, its views, the caller's
// workspaces) value each row by its item's id and give its name as the row's keywords, which the palette's search
// finds it by (powerKFilter; P8 spec 2.2, after P7c's F2 and F5). cmdk keeps a list's selection by its rows' values:
// were a row valued by its name, two of one name would be one to its keys, the second never picked. A row renamed
// while its list is open, its id the same, is found by its new name, no longer by its old one. The builder gives a row
// the keywords its getKeywords says; given none, a row has none and is found by its value, as before.
import { act, type ReactElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { Command } from "cmdk";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { cycleOf, preferencesOf, projectOf } from "@/store/project/fake-projects";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { powerKFilter } from "../ui/modal/filter";
import { PowerKMenuBuilder } from "./builder";
import { PowerKCyclesMenu } from "./cycles";
import { PowerKViewsMenu } from "./views";
import { PowerKWorkspacesMenu } from "./workspaces";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
// A workspace's logo loads the session's stores as it is imported; the rows here are told by their text.
vi.mock("@/components/workspace/logo", () => ({ WorkspaceLogo: () => null }));

/** An item of a menu: its id, never its name, which another may share. */
type Row = { id: string; name: string };
/** A menu of these rows, telling the id of the one picked. */
type Menu = (rows: Row[], pick: (id: string) => void) => ReactElement;

const none = () => {};
const project = projectOf("A", "id-ws");
const { display_filters, display_properties } = preferencesOf();
const viewOf = ({ id, name }: Row): View => ({
  id,
  workspace_id: project.workspace_id,
  project_id: project.id,
  name,
  description: "",
  logo_props: {},
  filters: {},
  display_filters,
  display_properties,
  owned_by_id: "u",
  created_at: "2026-11-01T09:00:00Z",
  updated_at: "2026-11-01T09:00:00Z",
});
const menus: [string, Menu][] = [
  [
    "cycles",
    (rows, pick) => (
      <PowerKCyclesMenu
        cycles={rows.map(({ id, name }) => cycleOf(project, name, { id }))}
        onSelect={(cycle) => pick(cycle.id)}
      />
    ),
  ],
  ["views", (rows, pick) => <PowerKViewsMenu views={rows.map(viewOf)} onSelect={(view) => pick(view.id)} />],
  [
    "workspaces",
    (rows, pick) => (
      <PowerKWorkspacesMenu
        workspaces={rows.map(({ id, name }) => workspaceOf(id, { id, name }))}
        onSelect={(workspace) => pick(workspace.id)}
      />
    ),
  ],
];

let host: HTMLDivElement, root: Root;
/** The menu under the search, filtered once there is one as the palette's hosts filter it. */
const render = (menu: ReactElement, search: string) =>
  act(async () =>
    root.render(
      <Command shouldFilter={search !== ""} filter={powerKFilter}>
        <Command.Input value={search} />
        <Command.List>{menu}</Command.List>
      </Command>
    )
  );
const found = () => [...host.querySelectorAll('[role="option"]')].map((option) => option.textContent);
/** A key pressed in the palette's search, as its list hears it. */
const key = (name: string) =>
  act(async () => {
    host.querySelector("input")!.dispatchEvent(new KeyboardEvent("keydown", { key: name, bubbles: true }));
  });
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  // oxlint-disable-next-line no-extend-native -- jsdom has no scrolling, which cmdk asks of the item it selects.
  HTMLElement.prototype.scrollIntoView = vi.fn();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  Reflect.deleteProperty(HTMLElement.prototype, "scrollIntoView");
  vi.unstubAllGlobals();
});

it.each(menus)(
  "the %s menu's search finds two rows of one name, and its keys pick each, by its id",
  async (_, menu) => {
    const pick = vi.fn();
    await render(
      menu(
        [
          { id: "x-1", name: "Sprint" },
          { id: "x-2", name: "Sprint" },
        ],
        pick
      ),
      "spr"
    );
    expect(found()).toEqual(["Sprint", "Sprint"]);
    await key("Enter");
    await key("ArrowDown");
    await key("Enter");
    expect(pick.mock.calls).toEqual([["x-1"], ["x-2"]]);
  }
);

it.each(menus)("a row of the %s menu renamed while its list is open is found by its new name", async (_, menu) => {
  await render(menu([{ id: "x-1", name: "Sprint" }], none), "spr");
  expect(found()).toEqual(["Sprint"]);
  await render(menu([{ id: "x-1", name: "Launch" }], none), "lau");
  expect(found(), "the new name").toEqual(["Launch"]);
  await render(menu([{ id: "x-1", name: "Launch" }], none), "spr");
  expect(found(), "the old name").toEqual([]);
});

/** The builder's one row, valued x-1 and named Sprint, its name its keywords when keywords says so. */
const builder = (keywords: boolean, search: string) =>
  render(
    <PowerKMenuBuilder<Row>
      items={[{ id: "x-1", name: "Sprint" }]}
      getKey={(row) => row.id}
      getValue={(row) => row.id}
      getKeywords={keywords ? (row) => [row.name] : undefined}
      getLabel={(row) => row.name}
      onSelect={none}
    />,
    search
  );

it.each([
  ["given getKeywords, by its keywords, never by its value", true, "spr", "x-1"],
  ["given none, by its value, as before", false, "x-1", "spr"],
] as const)("the builder's row is found %s", async (_, keywords, finds, misses) => {
  await builder(keywords, finds);
  expect(found(), finds).toEqual(["Sprint"]);
  await builder(keywords, misses);
  expect(found(), misses).toEqual([]);
});
