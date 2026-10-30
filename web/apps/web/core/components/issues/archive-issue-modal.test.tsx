// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { IssueDetailQuickActions } from "./issue-detail/issue-detail-quick-actions";
import { stateOf } from "@/store/project/fake-projects";
import { MemoryRouter } from "react-router";
import { observer } from "mobx-react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { issueDetailOf, issuePageOf, requestedIssue } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { useIssueListRefresh, IssueListRetry } from "@/hooks/use-issue-list-refresh";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { ArchiveIssueModal } from "./archive-issue-modal";
import { IssueArchiveHost, useArchiveRequest } from "./issue-layouts/archive-host";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined, navigate: vi.fn() }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "archive" } }, publicClient: {} }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useNavigate: () => bound.navigate,
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
const close = vi.fn();
let submissions: string[];
const writes = () => tab.nerve.calls.filter((call) => call.method === "POST");
const button = (text: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find((b) => b.textContent === text)!;
const dialog = () => document.querySelector<HTMLElement>('[role="dialog"][aria-modal="true"]');
const archived = (id: string) => ({
  ...issueDetailOf(id),
  archived_at: "2026-10-31T00:00:00Z",
  updated_at: "2026-10-31T00:00:00Z",
});
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  close.mockReset();
  bound.navigate.mockReset();
  toasts.length = 0;
  submissions = [];
  for (const id of ["old", "new"]) tab.app.issue.issues.acceptRead(issueDetailOf(id), tab.app.issue.captureRead());
});
afterEach(async () => tab.close());
const Refresh = observer(function Refresh() {
  const list = tab.app.issue.archivedIssues.currentList;
  const result = useIssueListRefresh(
    list,
    [...(list?.value?.groups.keys() ?? [])].map((group) => ({ group }))
  );
  return <IssueListRetry {...result} />;
});
async function render(id = "old", open = true, refresh = false) {
  await tab.render(
    <MemoryRouter>
      {refresh && <Refresh />}
      <ArchiveIssueModal
        dataId={id}
        isOpen={open}
        handleClose={close}
        onSubmit={async () => {
          submissions.push(id);
          await tab.app.issue.projectIssues.archiveIssue("ws", "A", id);
        }}
      />
    </MemoryRouter>
  );
}
async function submit(twice = false) {
  await act(async () => {
    const b = button("common.archive");
    expect(b).toBeDefined();
    b.click();
    if (twice) b.click();
  });
  await until(() => writes().length === 1, "archive POST");
}
it("names the actual dialog and keeps normal keyboard order", async () => {
  await render();
  const name = dialog()?.getAttribute("aria-labelledby");
  expect(name).toBeTruthy();
  expect(document.getElementById(name!)?.textContent).toContain("issue.archive.label");
  expect([...dialog()!.querySelectorAll<HTMLElement>("[tabindex]")].every((node) => node.tabIndex <= 0)).toBe(true);
});
it("holds one archive until confirmation and cannot dismiss while pending", async () => {
  await render();
  await submit(true);
  expect(submissions).toEqual(["old"]);
  expect(writes()[0]!.path).toBe("/api/v0/issues/old/archive");
  await act(async () => {
    button("common.cancel").click();
    dialog()!.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toEqual([]);
  await act(async () => writes()[0]!.answer(json(200, archived("old"))));
  expect(close).toHaveBeenCalledTimes(1);
  expect(toasts).toEqual([expect.objectContaining({ type: "success" })]);
  expect(writes()).toHaveLength(1);
});
it("keeps the dialog on a native refusal and permits an explicit retry", async () => {
  await render();
  await submit();
  await act(async () => writes()[0]!.answer(problem(422, "issue.not_archivable")));
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toEqual([expect.objectContaining({ type: "error", message: "errors.issue_not_archivable" })]);
  expect(button("common.archive").disabled).toBe(false);
  await act(async () => button("common.archive").click());
  await until(() => writes().length === 2, "explicit retry");
  await act(async () => writes()[1]!.answer(json(200, archived("old"))));
  expect(close).toHaveBeenCalledTimes(1);
});
it.each(["item ABA", "open ABA", "permission ABA", "unmount", "retire"] as const)(
  "does not complete a newer host after %s",
  async (ending) => {
    await render();
    await submit();
    if (ending === "item ABA") {
      await render("new");
      await render();
    } else if (ending === "open ABA") {
      await render("old", false);
      await render();
    } else if (ending === "permission ABA") {
      await act(async () => {
        await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
        await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
      });
    } else if (ending === "retire") tab.app.dispose();
    else await tab.render(null);
    await act(async () => writes()[0]!.answer(json(200, archived("old"))));
    expect(close).not.toHaveBeenCalled();
    expect(toasts).toEqual([]);
  }
);
it("does not turn a missing item into a successful command", async () => {
  await expect(render("missing")).resolves.toBeUndefined();
  expect(dialog()).toBeNull();
  expect(writes()).toEqual([]);
  expect(close).not.toHaveBeenCalled();
});

const ArchivableRow = observer(function ArchivableRow() {
  const requestArchive = useArchiveRequest();
  const issue = tab.app.issue.issues.getIssueById("old");
  if (!issue || issue.archived_at) return null;
  return (
    <button onClick={() => requestArchive!(issue, () => tab.app.issue.projectIssues.archiveIssue("ws", "A", issue.id))}>
      Archive row
    </button>
  );
});
async function openRowConfirmation() {
  await tab.render(
    <MemoryRouter>
      <IssueArchiveHost>
        <ArchivableRow />
      </IssueArchiveHost>
    </MemoryRouter>
  );
  await act(async () => button("Archive row").click());
  await submit(true);
}
it("the layout keeps the confirmation owner when the confirmed archive removes its last row", async () => {
  await openRowConfirmation();
  await act(async () => writes()[0]!.answer(json(200, archived("old"))));
  expect(button("Archive row")).toBeUndefined();
  expect(dialog()).toBeNull();
  expect(toasts).toEqual([expect.objectContaining({ type: "success", title: "issue.archive.success.label" })]);
  expect(writes()).toHaveLength(1);
});
it("the layout ends the pending confirmation on permission ABA without reviving it or announcing its late result", async () => {
  await openRowConfirmation();
  await act(async () => loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []));
  expect(dialog()).toBeNull();
  await act(async () => loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]));
  expect(dialog()).toBeNull();
  await act(async () => writes()[0]!.answer(json(200, archived("old"))));
  expect(toasts).toEqual([]);
  expect(writes()).toHaveLength(1);
});
it("an archive confirmation stays successful when a crossing read requires verification that fails, and retry sends only GET", async () => {
  const first = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.archivedIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  const oldPage = issuePageOf([]);
  first.call.answer(json(200, oldPage));
  await first.promise;
  await render("old", true, true);
  const crossing = await requestedIssue(tab.nerve, () => tab.app.issue.archivedIssues.currentList!.fetchFirst(), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  await submit();
  await act(async () => writes()[0]!.answer(json(200, archived("old"))));
  await act(async () => {
    crossing.call.answer(json(200, oldPage));
  });
  const verification = () => tab.nerve.to("/api/v0/issues/old").filter((call) => call.method === "GET");
  await act(async () => until(() => verification().length === 1, "post-confirmation verification"));
  const crossed = expect(crossing.promise).rejects.toMatchObject({ status: 503 });
  await act(async () => verification()[0]!.answer(problem(503, "unavailable")));
  await crossed;
  expect(close).toHaveBeenCalledTimes(1);
  expect(toasts.filter((t) => t.type === "success")).toHaveLength(1);
  expect(button("common.retry")).toBeDefined();
  await act(async () => button("common.retry").click());
  await act(async () => until(() => verification().length === 2, "GET retry"));
  await act(async () => verification()[1]!.answer(json(200, archived("old"))));
  expect(writes()).toHaveLength(1);
});

async function mountDetail() {
  tab.nerve.respond = (call) => {
    if (call.path.endsWith("/states"))
      call.answer(json(200, { data: [stateOf(tab.project, "Done", "completed", 1, { id: "state" })] }));
  };
  await tab.app.state.fetchProjectStates("A");
  await tab.render(
    <MemoryRouter>
      <IssueDetailQuickActions workspaceSlug="ws" projectId="A" issueId="old" />
    </MemoryRouter>
  );
  await act(async () => document.querySelector<HTMLButtonElement>("[data-main-menu] > button")!.click());
  const archive = [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')].find(
    (item) => item.textContent === "common.actions.archive" && !item.closest("[inert]")
  );
  expect(archive).toBeDefined();
  await act(async () => archive!.click());
}
it("the actual detail menu does not swallow an archive refusal and then announce success", async () => {
  await mountDetail();
  await submit();
  await act(async () => writes()[0]!.answer(problem(422, "issue.not_archivable")));
  expect(toasts).toEqual([expect.objectContaining({ type: "error", message: "errors.issue_not_archivable" })]);
  expect(dialog()).not.toBeNull();
  expect(bound.navigate).not.toHaveBeenCalled();
});
it.each([false, true])(
  "the detail archive navigates and reports only for its current confirmation (unmounted=%s)",
  async (unmount) => {
    await mountDetail();
    await submit();
    if (unmount) await tab.render(null);
    await act(async () => writes()[0]!.answer(json(200, archived("old"))));
    expect(bound.navigate).toHaveBeenCalledTimes(unmount ? 0 : 1);
    expect(toasts.filter((toast) => toast.type === "success")).toHaveLength(unmount ? 0 : 1);
  }
);
