// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { beforeEach, afterEach, expect, it, vi } from "vitest";
import type { Issue } from "@nerve/api-client";
import type { RootStore } from "@/store/root.store";
import { heldChange } from "@/lib/auth/fake-tab";
import { issueOf } from "@/store/issue/fake-issues";
import { projectOf, loadProjects } from "@/store/project/fake-projects";
import { json, problem } from "@/lib/auth/fake-nerve";
import {
  issueSelectionTab,
  clickSelection,
  searchSelection,
  selectionButton,
} from "@/components/core/modals/fake-issue-selection";
import { ParentIssuesListModal } from "./parent-issues-list-modal";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "parent" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: undefined,
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
let pending: ReturnType<typeof heldChange<void>>;
const close = vi.fn();
const change = vi.fn<(issue: Issue) => void | Promise<void>>();
beforeEach(async () => {
  tab = await issueSelectionTab();
  pending = heldChange<void>();
  change.mockReset().mockImplementation(() => pending.sent);
  close.mockReset();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
});
afterEach(async () => tab.close());
const render = (project = "A", source?: string, open = true) =>
  tab.render(
    <ParentIssuesListModal isOpen={open} projectId={project} issueId={source} handleClose={close} onChange={change} />
  );
async function opened(source?: string) {
  await render("A", source);
  expect(tab.searches()).toHaveLength(1);
}
async function answer(name = "Candidate", project = "A", index = tab.searches().length - 1) {
  await act(async () => tab.searches()[index]!.answer(json(200, { data: [issueOf(name, project)] })));
}
const choose = () => clickSelection('[role="option"]');

it("newIssueSearchOmitsForAndSource, accepts an empty term and passes the native Issue to a void consumer", async () => {
  await opened();
  expect(tab.searches()[0]!.query).toEqual({ project_id: "A" });
  change.mockImplementation(() => {});
  await answer();
  await choose();
  expect(change).toHaveBeenCalledExactlyOnceWith(issueOf("Candidate"));
  expect(close).toHaveBeenCalledOnce();
});
it("existingParentSearchUsesSource and invalidates old rows as soon as the term changes", async () => {
  await opened("source");
  expect(tab.searches()[0]!.query).toEqual({ project_id: "A", for: "parent", issue_id: "source" });
  await answer();
  await searchSelection("Next");
  expect(document.querySelector('[role="option"]')).toBeNull();
  await act(async () => vi.advanceTimersByTimeAsync(500));
  expect(tab.searches()[1]!.query).toEqual({ project_id: "A", for: "parent", issue_id: "source", q: "Next" });
  await answer("Next parent");
  expect(document.body.textContent).not.toContain("Candidate");
  expect(document.body.textContent).toContain("Next parent");
});
it.each(["project", "source", "permission", "reopen"])("oldSearchCannotBeSelectedAfter %s changes", async (kind) => {
  await opened("source");
  if (kind === "project") await render("B", "source");
  if (kind === "source") await render("A", "other");
  if (kind === "reopen") {
    await render("A", "source", false);
    await render("A", "source");
  }
  if (kind === "permission")
    await act(async () => {
      tab.app.projectRoot.project.confirmProject(tab.project, (p) => ({ ...p, member_role: null, network: 0 }));
      tab.app.projectRoot.project.confirmProject(tab.project, () => tab.project);
    });
  expect(tab.searches()).toHaveLength(2);
  await answer("Current", kind === "project" ? "B" : "A");
  await answer("Retired", "A", 0);
  expect(document.body.textContent).toContain("Current");
  expect(document.body.textContent).not.toContain("Retired");
  expect(change).not.toHaveBeenCalled();
});
it("checks search proof at selection time, even before React renders a permission ABA", async () => {
  await opened();
  await answer();
  const option = document.querySelector<HTMLElement>('[role="option"]')!;
  await act(async () => {
    tab.app.projectRoot.project.confirmProject(tab.project, (p) => ({ ...p, member_role: null, network: 0 }));
    tab.app.projectRoot.project.confirmProject(tab.project, () => tab.project);
    tab.app.issue.issues.acceptRead(issueOf("Candidate"), tab.app.issue.captureRead());
    option.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
  });
  expect(change).not.toHaveBeenCalled();
  expect(close).not.toHaveBeenCalled();
});
it("waits for a Promise consumer, excludes duplicate selection and pending Escape, and retains a refusal", async () => {
  await opened();
  await answer();
  await choose();
  await choose();
  await act(async () =>
    document
      .querySelector("input")!
      .dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }))
  );
  expect(change).toHaveBeenCalledOnce();
  expect(close).not.toHaveBeenCalled();
  await act(async () => pending.refuse(new Error("refused")));
  expect(close).not.toHaveBeenCalled();
  expect(document.body.textContent).toContain("Candidate");
  change.mockImplementation(() => {});
  await choose();
  expect(change).toHaveBeenCalledTimes(2);
  expect(close).toHaveBeenCalledOnce();
});
it("an old consumer cannot close the new project", async () => {
  await opened();
  await answer();
  await choose();
  await render("B");
  await act(async () => pending.answer());
  expect(close).not.toHaveBeenCalled();
});
it("reports a current read refusal, retries only GET, and names the modal without nested option controls", async () => {
  await opened();
  await act(async () => tab.searches()[0]!.answer(problem(503, "unavailable")));
  expect(document.querySelector('[role="alert"]')).not.toBeNull();
  expect(document.querySelector('[role="listbox"] button')).toBeNull();
  await act(async () => selectionButton("common.retry").click());
  await answer();
  expect(tab.searches()).toHaveLength(2);
  expect(document.querySelector('[role="alert"]')).toBeNull();
  const dialog = document.querySelector('[role="dialog"]')!;
  expect(document.getElementById(dialog.getAttribute("aria-labelledby")!)?.textContent).toBe("issue.add.parent");
  expect(dialog.querySelector("input")?.getAttribute("aria-label")).toBe("common.search.placeholder");
  expect([...dialog.querySelectorAll<HTMLElement>("[tabindex]")].every((el) => el.tabIndex <= 0)).toBe(true);
  expect(dialog.querySelector('[role="option"] a, [role="option"] button')).toBeNull();
  expect(change).not.toHaveBeenCalled();
});
