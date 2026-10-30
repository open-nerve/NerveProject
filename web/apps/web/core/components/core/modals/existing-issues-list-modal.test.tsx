// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { beforeEach, afterEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { heldChange } from "@/lib/auth/fake-tab";
import { issueOf } from "@/store/issue/fake-issues";
import { projectOf, loadProjects } from "@/store/project/fake-projects";
import { json, problem } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import { ExistingIssuesListModal, type SelectionSubmitResult } from "./existing-issues-list-modal";
import { issueSelectionTab, selectionButton, clickSelection, searchSelection } from "./fake-issue-selection";
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: undefined,
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
let pending: ReturnType<typeof heldChange<SelectionSubmitResult>>;
const close = vi.fn();
const submit = vi.fn<(ids: readonly string[]) => Promise<SelectionSubmitResult>>();
beforeEach(async () => {
  tab = await issueSelectionTab();
  close.mockReset();
  submit.mockReset();
  toasts.length = 0;
  pending = heldChange<SelectionSubmitResult>();
  submit.mockImplementation(() => pending.sent);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
});
afterEach(async () => tab.close());
const render = (project = "A", source = "parent", open = true) =>
  tab.render(
    <ExistingIssuesListModal
      workspaceSlug="ws"
      projectId={project}
      isOpen={open}
      handleClose={close}
      searchParams={{ for: "sub_issue", issue_id: source }}
      handleOnSubmit={submit}
      workspaceLevelToggle
    />
  );
const answer = (id = "Selected", project = "A", index = tab.searches().length - 1) =>
  act(async () => {
    tab.searches()[index]!.answer(json(200, { data: [issueOf(id, project)] }));
  });
const select = () => clickSelection('[role="option"]');
const send = () => act(async () => selectionButton("issue.select.add_selected").click());
it("retains the selection and handles a failed submit without closing", async () => {
  await render();
  await answer();
  await select();
  await send();
  await act(async () => pending.refuse(new Error("refused")));
  expect(close).not.toHaveBeenCalled();
  expect(selectionButton("issue.select.add_selected").disabled).toBe(false);
  expect(document.querySelector('[role="option"]')?.getAttribute("aria-selected")).toBe("true");
  expect(toasts).toHaveLength(1);
});
it("a submit from the previous project does not close the current modal", async () => {
  await render();
  await answer();
  await select();
  await send();
  await render("B");
  await act(async () => pending.answer({ confirmedIds: ["Selected"], failures: [] }));
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toHaveLength(0);
});
it("shows a search refusal and retries only the read", async () => {
  await render();
  await act(async () => tab.searches()[0]!.answer(problem(403, "forbidden")));
  expect(document.querySelector('[role="alert"]')).not.toBeNull();
  await act(async () => selectionButton("common.retry").click());
  await answer();
  expect(tab.searches()).toHaveLength(2);
  expect(document.querySelector('[role="alert"]')).toBeNull();
  expect(document.body.textContent).toContain("Selected");
  expect(submit).not.toHaveBeenCalled();
});
it("a search from an old project cannot replace the new project's rows", async () => {
  await render();
  await render("B");
  await answer("Current project", "B");
  await answer("Old project", "A", 0);
  expect(document.body.textContent).toContain("Current project");
  expect(document.body.textContent).not.toContain("Old project");
});
it.each(["project", "source", "workspace", "permission", "reopen"])(
  "changing %s ends the old selection instead of submitting hidden rows",
  async (change) => {
    await render();
    await answer();
    await select();
    expect(selectionButton("issue.select.add_selected").disabled).toBe(false);
    if (change === "project") await render("B");
    if (change === "source") await render("A", "new-parent");
    if (change === "workspace") await clickSelection('[role="switch"]');
    if (change === "permission")
      await act(async () => {
        tab.app.projectRoot.project.confirmProject(tab.project, (p) => ({ ...p, member_role: null, network: 0 }));
        tab.app.projectRoot.project.confirmProject(tab.project, () => tab.project);
      });
    if (change === "reopen") {
      await render("A", "parent", false);
      await render();
    }
    expect(selectionButton("issue.select.add_selected").disabled).toBe(true);
    expect(document.querySelector('[aria-label="common.remove Selected"]')).toBeNull();
  }
);
it("keeps proven selections across terms while invalidating candidate rows immediately", async () => {
  await render();
  await answer();
  await select();
  await searchSelection("new");
  expect(document.querySelector('[role="option"]')).toBeNull();
  expect(document.querySelector('[aria-label="common.remove Selected"]')).not.toBeNull();
  await act(async () => vi.advanceTimersByTimeAsync(500));
  await answer("Second");
  await select();
  await send();
  expect(submit).toHaveBeenCalledWith(["Selected", "Second"]);
});
it("names the modal and options, has no positive tabindex or nested candidate controls, and submits once", async () => {
  await render();
  await answer();
  const dialog = document.querySelector('[role="dialog"]')!;
  expect(document.getElementById(dialog.getAttribute("aria-labelledby")!)?.textContent).toBe("issue.add.existing");
  expect([...dialog.querySelectorAll<HTMLElement>("[tabindex]")].every((element) => element.tabIndex <= 0)).toBe(true);
  expect(dialog.querySelector('[role="option"] a, [role="option"] input, [role="option"] button')).toBeNull();
  await select();
  await act(async () => {
    selectionButton("issue.select.add_selected").click();
    selectionButton("issue.select.add_selected").click();
  });
  expect(submit).toHaveBeenCalledTimes(1);
  expect(close).not.toHaveBeenCalled();
  await act(async () => pending.answer({ confirmedIds: ["Selected"], failures: [] }));
  expect(close).toHaveBeenCalledTimes(1);
});
it("rejects a confirmation for IDs outside the submitted snapshot", async () => {
  await render();
  await answer();
  await select();
  await send();
  await act(async () => pending.answer({ confirmedIds: ["foreign"], failures: [] }));
  expect(close).not.toHaveBeenCalled();
  expect(selectionButton("issue.select.add_selected").disabled).toBe(false);
  expect(toasts).toHaveLength(1);
});
