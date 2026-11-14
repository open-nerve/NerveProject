// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7b spec 2.5, R8: the grouping and the sub-grouping panels of a list page. On a page of a project (its list, a
// cycle's, a module's: the route names the project) the cycle and the module are offered while the route's project has
// each domain turned on; one saved stays shown and checked while its domain is off, and is changed as any other: nerve
// still groups by it, and the panel never changes a saved query by itself. On the member view, which has no project of
// its route, the panels offer what the page's options give. Every other grouping is offered as the options say.
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
import type { IIssueDisplayFilterOptions, TIssueGroupByOptions } from "@nerve/types";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { FilterGroupBy } from "./group-by";
import { FilterSubGroupBy } from "./sub-group-by";

const bound = vi.hoisted(() => ({ params: {} as Record<string, string> }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "panel" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", () => ({ useParams: () => bound.params }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));

let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
  host = document.createElement("div");
  document.body.append(host);
  renderer = createRoot(host);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});

const { list, kanban } = ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.layoutOptions;
type Update = (key: TIssueGroupByOptions) => void;
/**
 * The two panels, each with the project page's options and its saved grouping: the list's grouping, the board's
 * sub-grouping under its grouping by priority.
 */
const panels: { panel: string; render: (saved: TIssueGroupByOptions, update: Update) => ReactNode }[] = [
  {
    panel: "grouping",
    render: (saved, update) => (
      <FilterGroupBy
        displayFilters={{ layout: "list", group_by: saved } as IIssueDisplayFilterOptions}
        groupByOptions={list!.display_filters.group_by!}
        handleUpdate={update}
      />
    ),
  },
  {
    panel: "sub-grouping",
    render: (saved, update) => (
      <FilterSubGroupBy
        displayFilters={{ layout: "kanban", group_by: "priority", sub_group_by: saved } as IIssueDisplayFilterOptions}
        subGroupByOptions={kanban!.display_filters.sub_group_by!}
        handleUpdate={update}
      />
    ),
  },
];
/** The panel's options as it shows them, a checked one marked. */
async function offered(panel: ReactNode, switches: Partial<Project> = {}) {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, ...switches }]);
  await act(async () => renderer.render(panel));
  return [...host.querySelectorAll("button[aria-pressed]")].map(
    (button) => `${button.textContent}${button.getAttribute("aria-pressed") === "true" ? " (checked)" : ""}`
  );
}
const planning = ["common.cycle", "common.modules"];
const option = (title: string) =>
  [...host.querySelectorAll<HTMLButtonElement>("button[aria-pressed]")].find((button) => button.textContent === title);

describe.each(panels)("the $panel panel", ({ render }) => {
  it.each([
    { switches: {}, own: ["common.cycle", "common.modules"] },
    { switches: { cycle_view: false }, own: ["common.modules"] },
    { switches: { module_view: false }, own: ["common.cycle"] },
    { switches: { cycle_view: false, module_view: false }, own: [] },
  ])("offers on a project's page only what the route's project has on: $own", async ({ switches, own }) => {
    bound.params = { workspaceSlug: "ws", projectId: "A" };
    const shown = await offered(render(null, vi.fn()), switches);
    expect(shown.filter((title) => planning.includes(title))).toEqual(own);
    expect(shown).toContain("common.labels");
    expect(shown).toContain("common.none (checked)");
  });

  it.each([
    { saved: "cycle", title: "common.cycle", switches: { cycle_view: false } },
    { saved: "module", title: "common.modules", switches: { module_view: false } },
  ] as const)(
    "keeps a saved grouping by the $saved checked while its domain is off, changed as any other",
    async ({ saved, title, switches }) => {
      bound.params = { workspaceSlug: "ws", projectId: "A" };
      const update = vi.fn<Update>();
      const shown = await offered(render(saved, update), switches);
      expect(shown.filter((name) => name.startsWith(title))).toEqual([`${title} (checked)`]);
      await act(async () => option("common.labels")?.click());
      expect(update).toHaveBeenCalledWith("labels");
      // once changed, the grouping of the domain turned off is no longer offered
      await act(async () => renderer.render(render("labels", update)));
      expect(option(title)).toBeUndefined();
      expect(option("common.labels")?.getAttribute("aria-pressed")).toBe("true");
    }
  );

  it("offers on the member view, which has no project of its route, what the options give, whatever the projects have on", async () => {
    bound.params = { workspaceSlug: "ws", userId: "u-bob" };
    const shown = await offered(render(null, vi.fn()), { cycle_view: false, module_view: false });
    expect(shown.filter((title) => planning.includes(title))).toEqual(planning);
  });
});
