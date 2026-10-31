// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createContext, useContext, StrictMode, type ComponentProps, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { loadProjects } from "@/store/project/fake-projects";
import { CommandPaletteStore } from "@/store/command-palette.store";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { settle } from "@/lib/auth/fake-time";
import { json } from "@/lib/auth/fake-nerve";
import type { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
import type { DeleteIssueModal } from "@/components/issues/delete-issue-modal";
import { WorkItemLevelModals } from "./work-item-level";

const ui = vi.hoisted(() => ({
  navigate: vi.fn(),
  remove: vi.fn(),
  params: { workspaceSlug: "ws", projectId: "A" as string | undefined },
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "host" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("react-router", () => ({ useParams: () => useContext(ParamsContext), useNavigate: () => ui.navigate }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-command-palette", () => ({ useCommandPalette: () => commands }));
vi.mock("@/hooks/use-issues-actions", () => ({ useIssuesActions: () => ({ removeIssue: ui.remove }) }));
// The host is the target. These leaves expose the real props it hands to its forms.
vi.mock("@/components/issues/issue-modal/modal", () => ({
  CreateUpdateIssueModal: (props: ComponentProps<typeof CreateUpdateIssueModal>) => {
    createProps = props;
    return props.isOpen ? <span>create</span> : null;
  },
}));
vi.mock("@/components/issues/delete-issue-modal", () => ({
  DeleteIssueModal: (props: ComponentProps<typeof DeleteIssueModal>) => {
    deleteProps = props;
    return props.isOpen ? <span>delete</span> : null;
  },
}));
const ParamsContext = createContext(ui.params);
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let commands: CommandPaletteStore;
let createProps: ComponentProps<typeof CreateUpdateIssueModal>;
let deleteProps: ComponentProps<typeof DeleteIssueModal>;
let element: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  tab = await issueStoreTab();
  commands = new CommandPaletteStore();
  ui.navigate.mockReset();
  ui.remove.mockReset();
  ui.params = { workspaceSlug: "ws", projectId: "A" };
  tab.nerve.respond = (call) => call.answer(json(200, issueDetailOf("old")));
  expect(
    (await settle(tab.app.issue.issueDetail.issue.fetchIssueCore("ws", "A", "old"), "detail")).error
  ).toBeUndefined();
  element = document.createElement("div");
  document.body.append(element);
  root = createRoot(element);
});
afterEach(async () => {
  await act(async () => root.unmount());
  element.remove();
  tab.app.dispose();
  vi.useRealTimers();
});
const view = (key = "commands", identifier: string | undefined = `${tab.project.identifier}-1`) => (
  <WorkItemLevelModals key={key} workItemIdentifier={identifier} />
);
const render = async (children: ReactNode) =>
  act(async () => root.render(<ParamsContext.Provider value={ui.params}>{children}</ParamsContext.Provider>));

it("closesOwnedModalsOnScopeChange and clears allowed projects", async () => {
  await render(view());
  await act(async () => {
    commands.toggleCreateIssueModal(true, undefined, ["A"]);
    commands.toggleDeleteIssueModal(true);
  });
  ui.params = { ...ui.params, projectId: "B" };
  await render(view());
  expect(commands.isCreateIssueModalOpen).toBe(false);
  expect(commands.isDeleteIssueModalOpen).toBe(false);
  expect(commands.createWorkItemAllowedProjectIds).toBeUndefined();
});
it("keepsTheNewOwnersModalWhenTheOldHostUnmounts", async () => {
  await render(<>{view("old")}</>);
  await render(
    <>
      {view("old")}
      {view("new")}
    </>
  );
  await act(async () => commands.toggleCreateIssueModal(true));
  await render(<>{view("new")}</>);
  expect(commands.isCreateIssueModalOpen).toBe(true);
  await render(null);
  expect(commands.isCreateIssueModalOpen).toBe(false);
});
it("doesNotFinishAnOldHostsActionInTheNewScope", async () => {
  await render(view());
  const oldClose = createProps.onClose;
  let finish!: () => void;
  ui.remove.mockImplementation(
    () =>
      new Promise<void>((resolve) => {
        finish = resolve;
      })
  );
  let pending!: Promise<void>;
  await act(async () => {
    pending = Promise.resolve(deleteProps.onSubmit!());
  });
  ui.params = { ...ui.params, projectId: "B" };
  await render(view());
  await act(async () => commands.toggleCreateIssueModal(true));
  await act(async () => {
    finish();
    await pending;
    oldClose();
  });
  expect(ui.navigate).not.toHaveBeenCalled();
  expect(commands.isCreateIssueModalOpen).toBe(true);
});
it("keeps workspace creation available without a project under StrictMode", async () => {
  ui.params = { ...ui.params, projectId: undefined };
  await render(<StrictMode>{view("workspace", undefined)}</StrictMode>);
  await act(async () => commands.toggleCreateIssueModal(true));
  expect(element.textContent).toContain("create");
  await render(null);
  expect(commands.isCreateIssueModalOpen).toBe(false);
});
it("a current deletion refusal is not reported to the modal as success", async () => {
  await render(view());
  const error = new Error("refused");
  ui.remove.mockRejectedValue(error);
  await expect(deleteProps.onSubmit!()).rejects.toBe(error);
  expect(ui.navigate).not.toHaveBeenCalled();
});

it("permission ABA retires the command owner's callbacks even when the route and issue return unchanged", async () => {
  await render(view());
  await act(async () => commands.toggleCreateIssueModal(true));
  const oldClose = createProps.onClose;
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  });
  expect(commands.isCreateIssueModalOpen).toBe(false);
  await act(async () => commands.toggleCreateIssueModal(true));
  await act(async () => oldClose());
  expect(commands.isCreateIssueModalOpen).toBe(true);
});

it("a route ABA does not let the first same-issue close callback dismiss the new modal", async () => {
  await render(view());
  await act(async () => commands.toggleCreateIssueModal(true));
  const oldClose = createProps.onClose;
  ui.params = { ...ui.params, projectId: "B" };
  await render(view());
  ui.params = { ...ui.params, projectId: "A" };
  await render(view());
  await act(async () => commands.toggleCreateIssueModal(true));
  await act(async () => oldClose());
  expect(commands.isCreateIssueModalOpen).toBe(true);
});
