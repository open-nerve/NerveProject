// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createContext, useContext } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { issueOf } from "@/store/issue/fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import {
  issueSelectionTab,
  clickSelection,
  searchSelection,
  selectionButton,
} from "@/components/core/modals/fake-issue-selection";
import { SelectDuplicateInboxIssueModal } from "./select-duplicate";

let route = { workspaceSlug: "ws", projectId: "A", issueId: "inbox-source" };
const Params = createContext(route);
vi.mock("react-router", () => ({ useParams: () => useContext(Params) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "duplicate" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: undefined,
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
const submit = vi.fn<(id: string) => void>();
const close = vi.fn();
beforeEach(async () => {
  tab = await issueSelectionTab();
  route = { ...route, issueId: "inbox-source" };
  submit.mockReset();
  close.mockReset();
});
afterEach(async () => tab.close());
const render = (open = true) => {
  return tab.render(
    <Params.Provider value={route}>
      <SelectDuplicateInboxIssueModal isOpen={open} onClose={close} onSubmit={submit} />
    </Params.Provider>
  );
};
const answer = async (name: string, index = tab.searches().length - 1) => {
  await act(async () => tab.searches()[index]!.answer(json(200, { data: [issueOf(name), issueOf(route.issueId)] })));
};
async function opened() {
  await render();
  expect(tab.searches()).toHaveLength(1);
}
it("reads native candidates without a term or Issue-only source filter and keeps the void ID callback", async () => {
  await opened();
  expect(tab.searches()[0]!.query).toEqual({ project_id: "A" });
  await answer("Candidate");
  expect(document.querySelectorAll('[role="option"]')).toHaveLength(1);
  expect(document.body.textContent).not.toContain("inbox-source");
  await clickSelection('[role="option"]');
  expect(submit).toHaveBeenCalledExactlyOnceWith("Candidate");
  expect(close).toHaveBeenCalledOnce();
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
});
it("retires a candidate immediately on typing and ignores answers for earlier terms", async () => {
  await opened();
  await answer("Initial");
  await searchSelection("First");
  expect(document.querySelector('[role="option"]')).toBeNull();
  await act(async () => vi.advanceTimersByTimeAsync(500));
  await searchSelection("Last");
  await act(async () => vi.advanceTimersByTimeAsync(500));
  expect(tab.searches()[2]!.query).toEqual({ project_id: "A", q: "Last" });
  await answer("Last candidate");
  await answer("Old candidate", 1);
  expect(document.body.textContent).toContain("Last candidate");
  expect(document.body.textContent).not.toContain("Old candidate");
});
it.each(["permission", "source", "reopen"])("retires late reads across %s changes", async (kind) => {
  await opened();
  if (kind === "permission")
    await act(async () => {
      tab.app.projectRoot.project.confirmProject(tab.project, (p) => ({ ...p, member_role: null, network: 0 }));
      tab.app.projectRoot.project.confirmProject(tab.project, () => tab.project);
    });
  if (kind === "source") {
    route = { ...route, issueId: "other-source" };
    await render();
  }
  if (kind === "reopen") {
    await render(false);
    await render();
  }
  expect(tab.searches()).toHaveLength(2);
  await answer("Current");
  await answer("Retired", 0);
  expect(document.body.textContent).toContain("Current");
  expect(document.body.textContent).not.toContain("Retired");
  expect(submit).not.toHaveBeenCalled();
});
it("checks the read proof when clicked before a permission ABA renders", async () => {
  await opened();
  await answer("Candidate");
  const option = document.querySelector<HTMLElement>('[role="option"]')!;
  await act(async () => {
    tab.app.projectRoot.project.confirmProject(tab.project, (p) => ({ ...p, member_role: null, network: 0 }));
    tab.app.projectRoot.project.confirmProject(tab.project, () => tab.project);
    tab.app.issue.issues.acceptRead(issueOf("Candidate"), tab.app.issue.captureRead());
    option.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
  });
  expect(submit).not.toHaveBeenCalled();
  expect(close).not.toHaveBeenCalled();
});
it("names the dialog, shows a read refusal and retries only the read", async () => {
  await opened();
  await act(async () => tab.searches()[0]!.answer(problem(503, "unavailable")));
  expect(document.querySelector('[role="alert"]')).not.toBeNull();
  expect(document.querySelector('[role="listbox"] button')).toBeNull();
  await act(async () => selectionButton("common.retry").click());
  await answer("Recovered");
  expect(document.querySelector('[role="alert"]')).toBeNull();
  const dialog = document.querySelector('[role="dialog"]')!;
  expect(document.getElementById(dialog.getAttribute("aria-labelledby")!)?.textContent).toBe(
    "inbox_issue.actions.mark_as_duplicate"
  );
  expect(dialog.querySelector("input")?.getAttribute("aria-label")).toBe("common.search.placeholder");
  expect(dialog.querySelector('[role="option"] a, [role="option"] button')).toBeNull();
  expect(tab.searches()).toHaveLength(2);
  expect(submit).not.toHaveBeenCalled();
});
