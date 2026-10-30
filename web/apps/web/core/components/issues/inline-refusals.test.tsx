// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { problem, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import type { PriorityDropdown } from "@/components/dropdowns/priority";
import type { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { IssueProperties } from "./issue-layouts/properties/all-properties";
import { IssueColumn } from "./issue-layouts/spreadsheet/issue-column";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "inline" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => tab.app.state }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }), useNavigate: () => vi.fn() }));
// Only the input boundary is substituted; actual row/column UI ownership and native confirmation stay real.
vi.mock("@/components/dropdowns/priority", () => ({
  PriorityDropdown: (props: ComponentProps<typeof PriorityDropdown>) => (
    <button onClick={() => props.onChange("high")}>change</button>
  ),
}));
vi.mock("@/components/dropdowns/date-range", () => ({
  DateRangeDropdown: (props: ComponentProps<typeof DateRangeDropdown>) => (
    <button onClick={() => props.onSelect?.(undefined)}>change</button>
  ),
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
const source = () => ({ ...issueDetailOf("x"), start_date: "2026-10-01", target_date: "2026-10-10" });
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  toasts.length = 0;
  tab = await issueStoreTab();
  tab.app.issue.issues.acceptRead(source(), tab.app.issue.captureRead());
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
const update: NonNullable<ComponentProps<typeof IssueProperties>["updateIssue"]> = (project, id, body) =>
  tab.app.issue.projectIssues.updateIssue("ws", project!, id, body);
async function change(kind: "row" | "table" | "range") {
  await act(async () =>
    renderer.render(
      kind === "table" ? (
        <table>
          <tbody>
            <tr>
              <IssueColumn
                displayProperties={{ priority: true }}
                issueDetail={source()}
                disableUserActions={false}
                property="priority"
                updateIssue={update}
              />
            </tr>
          </tbody>
        </table>
      ) : (
        <IssueProperties
          issue={source()}
          displayProperties={kind === "range" ? { start_date: true, due_date: true } : { priority: true }}
          isReadOnly={false}
          className=""
          activeLayout="list"
          updateIssue={update}
        />
      )
    )
  );
  await act(async () => host.querySelector<HTMLButtonElement>("button")!.click());
  await act(async () => until(() => tab.nerve.calls.some((call) => call.method === "PATCH"), "inline write"));
  return tab.nerve.calls.find((call) => call.method === "PATCH")!;
}
for (const kind of ["row", "table", "range"] as const) {
  it(`${kind} announces a current refusal once and preserves canonical properties`, async () => {
    const write = await change(kind);
    await act(async () => {
      write.answer(problem(403, "forbidden"));
      await vi.advanceTimersByTimeAsync(20);
    });
    expect(toasts.map((toast) => toast.message)).toEqual(["errors.forbidden"]);
    expect(tab.app.issue.issues.getIssueById("x")).toMatchObject(source());
  });
  it(`${kind} suppresses a refusal after its host unmounts`, async () => {
    const write = await change(kind);
    await act(async () => renderer.render(null));
    await act(async () => {
      write.answer(problem(403, "forbidden"));
      await vi.advanceTimersByTimeAsync(20);
    });
    expect(toasts).toEqual([]);
  });
  it(`${kind} cannot revive a pending error after project permission disappears and returns`, async () => {
    const write = await change(kind);
    await act(async () => {
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
      write.answer(problem(403, "forbidden"));
      await vi.advanceTimersByTimeAsync(20);
    });
    expect(toasts).toEqual([]);
  });
  it(`${kind} keeps a confirmed change with no extra write or refusal`, async () => {
    const write = await change(kind);
    const patch = kind === "range" ? { start_date: null, target_date: null } : { priority: "high" as const };
    await act(async () => {
      write.answer(json(200, { ...source(), ...patch, updated_at: "2026-10-30T12:00:00Z" }));
      await vi.advanceTimersByTimeAsync(20);
    });
    expect(tab.app.issue.issues.getIssueById("x")).toMatchObject(patch);
    expect(tab.nerve.calls.filter((call) => call.method === "PATCH")).toHaveLength(1);
    expect(toasts).toEqual([]);
  });
}
