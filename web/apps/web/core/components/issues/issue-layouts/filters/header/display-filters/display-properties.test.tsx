// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7a spec 2.5, R11: the display panel's properties of a list page. On a page of a project (its list, a cycle's, a
// module's) the cycle and the modules are offered while the route's project has each turned on, the page's own domain
// too; on the member view, which has no project of its route, both are offered. Every other property is offered as
// the page's options say.
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { ISSUE_DISPLAY_FILTERS_BY_PAGE, ISSUE_STORE_TO_FILTERS_MAP } from "@nerve/constants";
import { EIssuesStoreType, type IIssueDisplayProperties } from "@nerve/types";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { FilterDisplayProperties } from "./display-properties";

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

/** Every property shown. */
const shown: IIssueDisplayProperties = Object.fromEntries(
  ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.layoutOptions.list!.display_properties.map((key) => [key, true])
);
/** The properties the page's list layout offers, as the panel shows them. */
async function offered(keys: (keyof IIssueDisplayProperties)[], switches: Partial<Project> = {}) {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, ...switches }]);
  await act(async () =>
    renderer.render(
      <FilterDisplayProperties displayProperties={shown} displayPropertiesToRender={keys} handleUpdate={vi.fn()} />
    )
  );
  return [...host.querySelectorAll("button[aria-pressed]")].map((button) => button.textContent);
}
const others = [
  "issue.display.properties.id",
  "common.assignee",
  "common.order_by.start_date",
  "common.order_by.due_date",
  "common.labels",
  "common.priority",
  "common.state",
  "issue.display.properties.sub_issue_count",
  "common.link",
  "content_files.title",
  "common.sort.created_on",
  "common.sort.updated_on",
];
const listOf = (type: EIssuesStoreType) => ISSUE_STORE_TO_FILTERS_MAP[type]!.layoutOptions.list!.display_properties;

it.each<{ page: string; type: EIssuesStoreType; params: Record<string, string> }>([
  { page: "the project's", type: EIssuesStoreType.PROJECT, params: {} },
  { page: "a cycle's", type: EIssuesStoreType.CYCLE, params: { cycleId: "C1" } },
  { page: "a module's", type: EIssuesStoreType.MODULE, params: { moduleId: "M1" } },
])("offers the modules and the cycle on $page page while the route's project has both on", async ({ type, params }) => {
  bound.params = { workspaceSlug: "ws", projectId: "A", ...params };
  expect(await offered(listOf(type))).toEqual([...others, "common.modules", "common.cycle"]);
});

it.each([
  { switches: { cycle_view: false }, own: ["common.modules"] },
  { switches: { module_view: false }, own: ["common.cycle"] },
  { switches: { cycle_view: false, module_view: false }, own: [] },
])("offers on the project's page only what its project has on: $own", async ({ switches, own }) => {
  bound.params = { workspaceSlug: "ws", projectId: "A" };
  expect(await offered(listOf(EIssuesStoreType.PROJECT), switches)).toEqual([...others, ...own]);
});

it("offers both on the member view, which has no project of its route, whatever the projects have on", async () => {
  bound.params = { workspaceSlug: "ws", userId: "u-bob" };
  const keys = ISSUE_DISPLAY_FILTERS_BY_PAGE.profile_issues.layoutOptions.list!.display_properties;
  expect(await offered(keys, { cycle_view: false, module_view: false })).toEqual([
    ...others,
    "common.modules",
    "common.cycle",
  ]);
});
