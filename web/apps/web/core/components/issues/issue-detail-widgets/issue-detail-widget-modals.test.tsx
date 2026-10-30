// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { IssueDetailModals } from "@/store/issue/issue-details/modals";
import { IssueDetailModalContext } from "@/hooks/store/use-issue-detail";
import { issueOf } from "@/store/issue/fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import { issueSelectionTab, clickSelection, selectionButton } from "@/components/core/modals/fake-issue-selection";
import { IssueDetailWidgetModals } from "./issue-detail-widget-modals";
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: undefined,
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
// These independent dialogs are closed in every selection scenario.
vi.mock("../issue-modal/modal", () => ({ CreateUpdateIssueModal: () => null }));
vi.mock("../issue-detail/links/create-update-link-modal", () => ({ IssueLinkCreateUpdateModal: () => null }));
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
let modals: IssueDetailModals;
beforeEach(async () => {
  tab = await issueSelectionTab();
  modals = new IssueDetailModals(() => {});
  modals.activate();
});
afterEach(async () => tab.close());
it.each(["sub_issue", "relation"] as const)(
  "uses the native %s query and atomic batch confirmation, including a failed followup read",
  async (mode) => {
    if (mode === "sub_issue") {
      modals.setIssueCrudOperationState({
        create: { toggle: false, parentIssueId: undefined },
        existing: { toggle: true, parentIssueId: "parent" },
      });
      modals.toggleSubIssuesModal("parent");
    } else {
      modals.setRelationKey("blocked_by");
      modals.toggleRelationModal("source", "blocked_by");
    }
    await tab.render(
      <IssueDetailModalContext.Provider value={modals}>
        <IssueDetailWidgetModals workspaceSlug="ws" projectId="A" issueId="source" />
      </IssueDetailModalContext.Provider>
    );
    const source = mode === "sub_issue" ? "parent" : "source";
    expect(tab.searches()).toHaveLength(1);
    expect(tab.searches()[0]!.query).toEqual({ for: mode, issue_id: source, project_id: "A" });
    if (mode === "relation") {
      await clickSelection('[role="switch"]');
      expect(tab.searches()[1]!.query).toEqual({ for: "relation", issue_id: "source" });
    }
    const selected = issueOf("selected");
    await act(async () =>
      tab
        .searches()
        .at(-1)!
        .answer(json(200, { data: [selected] }))
    );
    await clickSelection('[role="option"]');
    await act(async () => selectionButton("issue.select.add_selected").click());
    const path = `/api/v0/issues/${source}/${mode === "sub_issue" ? "sub-issues" : "relations"}`;
    const writes = () => tab.nerve.to(path).filter((call) => call.method === "POST");
    expect(writes()).toHaveLength(1);
    expect(writes()[0]!.body).toEqual(
      mode === "sub_issue" ? { sub_issue_ids: ["selected"] } : { relation_type: "blocked_by", issue_ids: ["selected"] }
    );
    await act(async () => writes()[0]!.answer(problem(422, "validation_failed")));
    expect(selectionButton("issue.select.add_selected").disabled).toBe(false);
    const directory = tab.nerve.respond;
    tab.nerve.respond = (call) => {
      if (call.method === "GET" && /^\/api\/v0\/issues\/[^/]+$/.test(call.path))
        call.answer(problem(503, "server_busy"));
      else directory?.(call);
    };
    await act(async () => selectionButton("issue.select.add_selected").click());
    expect(writes()).toHaveLength(2);
    await act(async () =>
      writes()[1]!.answer(
        json(mode === "sub_issue" ? 200 : 201, {
          data:
            mode === "sub_issue"
              ? [{ ...selected, parent_id: "parent" }]
              : [{ id: "relation", relation_type: "blocked_by", issue: selected, created_at: selected.created_at }],
        })
      )
    );
    await act(async () => vi.advanceTimersByTimeAsync(300));
    expect(document.querySelector('[role="dialog"]')).toBeNull();
    expect(writes()).toHaveLength(2);
    const root = tab.app.issue.issueDetail;
    if (mode === "sub_issue") expect(root.subIssues.subIssuesByIssueId("parent")).toEqual(["selected"]);
    else
      expect(root.relation.getRelationRows("source")).toEqual([
        { relationId: "relation", type: "blocked_by", targetId: "selected" },
      ]);
  }
);
