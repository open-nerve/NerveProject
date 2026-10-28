// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { runInAction } from "mobx";
import type { ICycle } from "@nerve/types";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, expect, it, vi } from "vitest";
import { rootStore } from "@/lib/store-context";
import { CycleStore } from "@/store/cycle.store";
import Page from "./page";
import { CycleIssuesHeader } from "../header";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-router")>()),
  useNavigate: () => vi.fn(),
}));
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/components/dropdowns/date-range", () => ({ DateRangeDropdown: () => null }));

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
globalThis.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
};
const container = document.createElement("div");
document.body.append(container);
const root = createRoot(container);
afterEach(async () => {
  await act(async () => root.render(null));
  vi.restoreAllMocks();
});

it("loads a cold cycle once and reloads when the route changes", async () => {
  const pending = new Promise<never>(() => {});
  const fetch = vi.spyOn((rootStore.cycle as CycleStore).cycleService, "getCycleDetails").mockReturnValue(pending);
  const params = { workspaceSlug: "ws", projectId: "project", cycleId: "first" };
  const cache = new Map();
  const render = (cycleId: string) =>
    root.render(
      <SWRConfig value={{ provider: () => cache, shouldRetryOnError: false }}>
        <Page {...({ params: { ...params, cycleId } } as ComponentProps<typeof Page>)} />
      </SWRConfig>
    );
  await act(async () => render("first"));
  expect(fetch).toHaveBeenCalledTimes(1);
  expect(fetch).toHaveBeenCalledWith("ws", "project", "first");
  expect(container.textContent).not.toContain("Cycle does not exist");
  await act(async () => render("second"));
  expect(fetch).toHaveBeenCalledTimes(2);
  expect(fetch).toHaveBeenLastCalledWith("ws", "project", "second");
});

it("keeps the real cycle sidebar and loads details once when the cycle is present", async () => {
  const cycle: ICycle = {
    id: "present",
    name: "Spring cycle",
    description: "Cycle details retained",
    project_id: "project",
    workspace_id: "ws",
    owned_by_id: "u",
    sort_order: 1,
    start_date: null,
    end_date: null,
    archived_at: null,
    project_detail: { id: "project" },
    view_props: { filters: {} },
    progress_snapshot: undefined,
    total_issues: 2,
    completed_issues: 1,
    backlog_issues: 1,
    started_issues: 0,
    unstarted_issues: 0,
    cancelled_issues: 0,
    distribution: { assignees: [], labels: [], completion_chart: {} },
  };
  runInAction(() => {
    rootStore.cycle.cycleMap.present = cycle;
  });
  const service = (rootStore.cycle as CycleStore).cycleService;
  const pending = new Promise<never>(() => {});
  const fetch = vi.spyOn(service, "getCycleDetails").mockReturnValue(pending);
  vi.spyOn(service, "cycleProgress").mockReturnValue(pending);
  const params = { workspaceSlug: "ws", projectId: "project", cycleId: "present" };
  await act(async () =>
    root.render(
      <MemoryRouter>
        <SWRConfig value={{ provider: () => new Map(), shouldRetryOnError: false }}>
          <CycleIssuesHeader {...params} />
          <Page {...({ params } as ComponentProps<typeof Page>)} />
        </SWRConfig>
      </MemoryRouter>
    )
  );
  expect(fetch).toHaveBeenCalledOnce();
  expect(container.textContent).toContain("Spring cycle");
  expect(container.textContent).toContain("project_cycles.active_cycle.progress");
  expect(container.textContent).not.toContain("issue.add.label");
  expect(container.textContent).not.toContain("common.display");
  expect(container.querySelector("textarea")?.value).toBe("Cycle details retained");
  runInAction(() => {
    delete rootStore.cycle.cycleMap.present;
  });
});
