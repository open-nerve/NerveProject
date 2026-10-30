// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { issueOf, issueDetailOf } from "@/store/issue/fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import { issueSelectionTab, clickSelection, selectionButton } from "@/components/core/modals/fake-issue-selection";
import { CalendarQuickAddIssueActions } from "./quick-add-issue-actions";
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: undefined,
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
// Creation has its own form tests; keep the actual calendar menu, selector, store and HTTP writes here.
vi.mock("../quick-add", () => ({
  QuickAddIssueRoot: ({ customQuickAddButton }: { customQuickAddButton: ReactNode }) => customQuickAddButton,
}));
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
beforeEach(async () => {
  tab = await issueSelectionTab();
  toasts.length = 0;
});
afterEach(async () => tab.close());
const writes = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
const send = () => act(async () => selectionButton("issue.select.add_selected").click());
async function open() {
  await tab.render(
    <CalendarQuickAddIssueActions
      prePopulatedData={{ target_date: "2026-11-01", priority: "urgent", label_ids: ["unrelated"] }}
    />
  );
  await clickSelection('button[aria-haspopup="menu"]');
  await act(async () => {
    const item = [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')].find(
      (element) => element.textContent === "issue.add.existing"
    )!;
    item.click();
  });
  expect(tab.searches()[0]?.query).toEqual({ target_date: "none", project_id: "A" });
  await act(async () =>
    tab.searches()[0]!.answer(json(200, { data: [issueOf("first"), { ...issueOf("second"), sequence_id: 2 }] }))
  );
  await act(async () => selectionButton("issue.select.select_all").click());
}
it("awaits both confirmations, retains only the failed choice, and retries only its target_date PATCH", async () => {
  await open();
  await send();
  expect(writes()).toHaveLength(2);
  expect(writes().map((call) => [call.path, call.body])).toEqual([
    ["/api/v0/issues/first", { target_date: "2026-11-01" }],
    ["/api/v0/issues/second", { target_date: "2026-11-01" }],
  ]);
  expect(document.querySelector('[role="dialog"]')).not.toBeNull();
  expect(selectionButton("common.adding").disabled).toBe(true);
  await act(async () => writes()[0]!.answer(json(200, { ...issueDetailOf("first"), target_date: "2026-11-01" })));
  expect(selectionButton("common.adding").disabled).toBe(true);
  await act(async () => writes()[1]!.answer(problem(422, "validation_failed")));
  expect(document.querySelector('[role="dialog"]')).not.toBeNull();
  expect(document.querySelector('[aria-label="common.remove first"]')).toBeNull();
  expect(document.querySelector('[aria-label="common.remove second"]')).not.toBeNull();
  expect(tab.app.issue.issues.getIssueById("first")?.target_date).toBe("2026-11-01");
  expect(tab.app.issue.issues.getIssueById("second")?.target_date).toBeNull();
  expect(toasts).toHaveLength(1);
  await send();
  expect(writes()).toHaveLength(3);
  expect(writes()[2]!.path).toBe("/api/v0/issues/second");
  await act(async () => writes()[2]!.answer(json(200, { ...issueDetailOf("second"), target_date: "2026-11-01" })));
  await act(async () => vi.advanceTimersByTimeAsync(300));
  expect(document.querySelector('[role="dialog"]')).toBeNull();
});
it("closing the calendar selector ends followups while confirmed writes remain canonical", async () => {
  await open();
  await send();
  await act(async () => selectionButton("common.cancel").click());
  await act(async () => {
    writes()[0]!.answer(json(200, { ...issueDetailOf("first"), target_date: "2026-11-01" }));
    writes()[1]!.answer(problem(403, "forbidden"));
  });
  expect(toasts).toHaveLength(0);
  expect(tab.app.issue.issues.getIssueById("first")?.target_date).toBe("2026-11-01");
});
