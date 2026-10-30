// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { IssueDetailHost } from "../detail-host";
import { useIssueOperations } from "../issue-detail/use-issue-operations";
import { ArchivedIssueQuickActions } from "../issue-layouts/quick-action-dropdowns/archived-issue";
import { IssueView } from "./view";
import { openIssuePeek } from "./selection";
const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// Leave the actual view -> header -> menu -> factory -> operation chain intact; omit unrelated content loads.
vi.mock("../issue-detail/issue-activity", () => ({ IssueActivity: () => null }));
vi.mock("../issue-detail-widgets", () => ({ IssueDetailWidgets: () => null }));
vi.mock("./issue-detail", () => ({ PeekOverviewIssueDetails: () => null }));
vi.mock("./properties", () => ({ PeekOverviewProperties: () => null }));
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
const restored = (id = "old") => ({ ...issueDetailOf(id), archived_at: null, updated_at: "2026-10-30T00:01:00Z" });
const archived = (id = "old") => ({ ...issueDetailOf(id), archived_at: "2026-10-29T00:00:00Z" });
const writes = () => tab.nerve.calls.filter((call) => call.method === "POST");
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  toasts.length = 0;
  for (const id of ["old", "next"]) tab.app.issue.issues.acceptRead(archived(id), tab.app.issue.captureRead());
});
afterEach(async () => tab.close());
function Peek({ id }: { id: string }) {
  const operations = useIssueOperations("ws", "A", id);
  return (
    <IssueDetailHost workspaceSlug="ws" projectId="A" issueId={id}>
      <IssueView workspaceSlug="ws" projectId="A" issueId={id} is_archived issueOperations={operations} embedIssue />
    </IssueDetailHost>
  );
}
async function render(id = "old", kind: "peek" | "archived" = "peek") {
  openIssuePeek(tab.app.issue.issueDetail, { workspaceSlug: "ws", projectId: "A", issueId: id, isArchived: true });
  await tab.render(
    <MemoryRouter>
      {kind === "peek" ? (
        <Peek key={id} id={id} />
      ) : (
        <ArchivedIssueQuickActions
          issue={archived(id)}
          parentRef={{ current: null }}
          handleDelete={async () => {}}
          handleRestore={async () => {
            await tab.app.issue.archivedIssues.restoreIssue("ws", "A", id);
          }}
        />
      )}
    </MemoryRouter>
  );
}
async function restore() {
  await act(async () => document.querySelector<HTMLButtonElement>("[data-main-menu] > button")!.click());
  const item = [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')].find(
    (option) => option.textContent === "Restore" && !option.closest("[inert]")
  )!;
  expect(item).toBeDefined();
  await act(async () => item.click());
  await until(() => writes().length === 1, "restore POST");
  expect(writes()[0]!.path).toBe("/api/v0/issues/old/unarchive");
}
it.each(["peek", "archived"] as const)(
  "%s restore refusal shows exactly one native reason and remains archived",
  async (kind) => {
    await render("old", kind);
    await restore();
    await act(async () => writes()[0]!.answer(problem(403, "forbidden")));
    expect(toasts).toEqual([expect.objectContaining({ type: "error", message: "errors.forbidden" })]);
    expect(tab.app.issue.issues.getIssueById("old")?.archived_at).not.toBeNull();
    if (kind === "peek") expect(tab.app.issue.issueDetail.peekIssue?.issueId).toBe("old");
  }
);
it.each(["peek", "archived"] as const)("%s restore confirms with exactly one success", async (kind) => {
  await render("old", kind);
  await restore();
  expect(toasts).toEqual([]);
  await act(async () => writes()[0]!.answer(json(200, restored())));
  expect(toasts).toEqual([expect.objectContaining({ type: "success", message: "issue.restore.success.message" })]);
  expect(tab.app.issue.issues.getIssueById("old")?.archived_at).toBeNull();
  if (kind === "peek") expect(tab.app.issue.issueDetail.peekIssue).toBeUndefined();
});
it.each(["peek", "archived"] as const)("%s old restore cannot report after host unmount", async (kind) => {
  await render("old", kind);
  await restore();
  await tab.render(null);
  await act(async () => writes()[0]!.answer(json(200, restored())));
  expect(toasts).toEqual([]);
});
it.each(["replace", "unmount", "permission ABA"] as const)("old restore cannot report after %s", async (end) => {
  await render();
  await restore();
  if (end === "replace") await render("next");
  else if (end === "unmount") await tab.render(null);
  else
    await act(async () => {
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    });
  await act(async () => writes()[0]!.answer(json(200, restored())));
  expect(toasts).toEqual([]);
  if (end === "replace") expect(tab.app.issue.issueDetail.peekIssue?.issueId).toBe("next");
});
