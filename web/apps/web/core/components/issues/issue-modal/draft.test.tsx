// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Draft } from "@nerve/api-client";
import React, { act, forwardRef, useImperativeHandle } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { draftOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { toasts } from "@/lib/fake-toast";
import { EIssuesStoreType } from "@nerve/types";
import type { RichTextEditor } from "@/components/editor/rich-text";
import { CreateUpdateIssueModalBase } from "./base";
import { DraftIssueBlock } from "../workspace-draft/draft-issue-block";
import type { IssuesModalProps } from "./modal";
import type { AlertModalCore } from "@nerve/ui";
const ui = vi.hoisted(() => ({
  close: vi.fn(),
  confirmed: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  projects: [] as string[],
}));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "draft-login" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/context/use-issue-modal", () => ({
  useIssueModal: () => ({
    allowedProjectIds: ui.projects,
    selectedParentIssue: null,
    setSelectedParentIssue: () => {},
  }),
}));
vi.mock("@/hooks/store/use-issues", () => ({ useIssues: () => ({ issues: tab.app.issue.workspaceDraftIssues }) }));
vi.mock("@/hooks/store/workspace-draft", () => ({ useWorkspaceDraftIssues: () => tab.app.issue.workspaceDraftIssues }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => tab.app.state }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => tab.app.workspaceRoot }));
vi.mock("@/hooks/store/use-editor-asset", () => ({ useEditorAsset: () => ({}) }));
vi.mock("@/hooks/use-issue-layout-store", () => ({ useIssueStoreType: () => EIssuesStoreType.WORKSPACE_DRAFT }));
vi.mock("@/hooks/use-issues-actions", () => ({
  useIssuesActions: () => ({ createIssue: ui.create, updateIssue: ui.update }),
}));
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
vi.mock("./components/default-properties", () => ({ IssueDefaultProperties: () => null }));
vi.mock("./components/parent-tag", () => ({ IssueParentTag: () => null }));
vi.mock("@/components/dropdowns/project/dropdown", () => ({
  ProjectDropdown: ({
    disabled,
    value,
    onChange,
  }: {
    disabled?: boolean;
    value?: string | null;
    onChange: (v: string | null) => void;
  }) => (
    <select
      aria-label="project"
      disabled={disabled}
      value={value ?? ""}
      onChange={(event) => onChange(event.target.value || null)}
    >
      <option value="">No project</option>
      <option value="A">A</option>
    </select>
  ),
}));
vi.mock("@/components/editor/rich-text", () => ({
  RichTextEditor: forwardRef((props: React.ComponentProps<typeof RichTextEditor>, ref) => {
    useImperativeHandle(ref, () => ({ isEditorReadyToDiscard: () => true, clearEditor: () => {} }));
    return (
      <textarea
        aria-label="description"
        value={props.initialValue ?? ""}
        onChange={(event) => props.onChange?.({}, event.target.value)}
      />
    );
  }),
}));
vi.mock("@/hooks/store/use-app-theme", () => ({ useAppTheme: () => ({ sidebarCollapsed: false }) }));
vi.mock("../workspace-draft/draft-issue-properties", () => ({ DraftIssueProperties: () => null }));
vi.mock("../workspace-draft/quick-action", () => ({
  WorkspaceDraftIssueQuickActions: ({ MENU_ITEMS }: { MENU_ITEMS: { key: string; action: () => void }[] }) => (
    <div>
      {MENU_ITEMS.map((item) => (
        <button key={item.key} type="button" onClick={item.action}>
          {item.key}
        </button>
      ))}
    </div>
  ),
}));
vi.mock("./modal", () => ({
  CreateUpdateIssueModal: (props: IssuesModalProps) =>
    props.isOpen ? <CreateUpdateIssueModalBase {...props} withDraftIssueWrapper={false} /> : null,
}));
vi.mock("@nerve/ui", async (original) => ({
  ...(await original<typeof import("@nerve/ui")>()),
  AlertModalCore: (props: React.ComponentProps<typeof AlertModalCore>) =>
    props.isOpen ? (
      <div role="alertdialog">
        <button type="button" onClick={() => void props.handleSubmit()}>
          confirm delete
        </button>
        <button type="button" onClick={props.handleClose}>
          cancel delete
        </button>
      </div>
    ) : null,
  ModalCore: ({ children, isOpen }: React.PropsWithChildren<{ isOpen: boolean }>) =>
    isOpen ? <div role="dialog">{children}</div> : null,
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let renderer: Root;
let container: HTMLElement;
let row: Draft;
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      disconnect() {}
    }
  );
  tab = await issueStoreTab();
  ui.projects = [];
  ui.close.mockClear();
  ui.confirmed.mockClear();
  toasts.length = 0;
  row = { ...draftOf("d"), project_id: null };
  const read = await requestedIssue(tab.nerve, () => tab.app.issue.workspaceDraftIssues.fetchIssues("ws"), [
    "GET",
    "/api/v0/workspaces/ws/drafts",
  ]);
  read.call.answer(json(200, { data: [row], total: 1, next_cursor: null }));
  await read.promise;
  tab.nerve.calls.length = 0;
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  container.remove();
  tab.app.dispose();
  vi.useRealTimers();
});
const render = (draft: Draft | null = row, publish = false, wrapper = false) =>
  act(async () =>
    renderer.render(
      <CreateUpdateIssueModalBase
        isOpen
        isDraft
        draft={draft ?? undefined}
        onClose={ui.close}
        onSubmit={ui.confirmed}
        fetchIssueDetails={false}
        moveToIssue={publish}
        withDraftIssueWrapper={wrapper}
        storeType={EIssuesStoreType.WORKSPACE_DRAFT}
      />
    )
  );
async function input(selector: string, text: string) {
  const node = container.querySelector<HTMLInputElement | HTMLTextAreaElement>(selector)!;
  expect(node).not.toBeNull();
  await act(async () => {
    const proto = node instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
    Object.getOwnPropertyDescriptor(proto, "value")!.set!.call(node, text);
    node.dispatchEvent(new Event("input", { bubbles: true }));
    node.dispatchEvent(new Event("change", { bubbles: true }));
  });
}
const submit = () =>
  act(async () =>
    container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
it("opens a real editor with no eligible projects and saves an incomplete draft", async () => {
  await render();
  expect(container.querySelector('[aria-label="description"]')).not.toBeNull();
  await input('input[name="name"]', "");
  await submit();
  await act(async () => until(() => tab.nerve.calls.length === 1, "save unassigned draft"));
  expect(tab.nerve.calls[0]!.path).toBe("/api/v0/drafts/d");
  expect(tab.nerve.calls[0]!.body).toEqual({ project_id: null, payload: { ...row.payload, name: "" } });
  await act(async () => tab.nerve.calls[0]!.answer(json(200, { ...row, payload: { ...row.payload, name: "" } })));
  expect(ui.close).toHaveBeenCalledOnce();
  expect(ui.confirmed).toHaveBeenCalledOnce();
  expect(tab.nerve.calls).toHaveLength(1);
});
it.each([403, 422])("keepsDraftInputAndDialogAfterRefusal (%s)", async (status) => {
  await render();
  await input('input[name="name"]', "unsaved name");
  await input('[aria-label="description"]', "new HTML");
  await submit();
  await act(async () => until(() => tab.nerve.calls.length === 1, "save"));
  await act(async () => tab.nerve.calls[0]!.answer(problem(status, "forbidden")));
  expect(ui.close).not.toHaveBeenCalled();
  expect(ui.confirmed).not.toHaveBeenCalled();
  expect(container.querySelector<HTMLInputElement>('input[name="name"]')!.value).toBe("unsaved name");
  expect(container.querySelector<HTMLTextAreaElement>('[aria-label="description"]')!.value).toBe("new HTML");
  expect(toasts.at(-1)?.message).toBe("errors.forbidden");
});
it("clearing the selected project is saved as null and is not replaced by the first allowed project", async () => {
  ui.projects = ["A"];
  row = { ...row, project_id: "A" };
  await render(row);
  const select = container.querySelector<HTMLSelectElement>('[aria-label="project"]')!;
  expect(select.disabled).toBe(false);
  await act(async () => {
    select.value = "";
    select.dispatchEvent(new Event("change", { bubbles: true }));
  });
  await submit();
  await act(async () => until(() => tab.nerve.calls.length === 1, "clear project"));
  expect(tab.nerve.calls[0]!.body).toMatchObject({ project_id: null });
  const saved = { ...row, project_id: null };
  await act(async () => tab.nerve.calls[0]!.answer(json(200, saved)));
  expect(tab.app.issue.workspaceDraftIssues.getIssueById("d")?.project_id).toBeNull();
  let reread!: Promise<Draft[]>;
  await act(async () => {
    reread = tab.app.issue.workspaceDraftIssues.fetchIssues("ws");
  });
  await act(async () => until(() => tab.nerve.calls.length === 2, "reread cleared project"));
  expect(tab.nerve.calls[1]).toMatchObject({ method: "GET", path: "/api/v0/workspaces/ws/drafts" });
  await act(async () => tab.nerve.calls[1]!.answer(json(200, { data: [saved], next_cursor: null, total: 1 })));
  await expect(reread).resolves.toEqual([saved]);
  expect(tab.app.issue.workspaceDraftIssues.getIssueById("d")?.project_id).toBeNull();
});
it("a publish refusal retains the saved native draft and the real form's latest input", async () => {
  await render(row, true);
  await input('input[name="name"]', "publish me");
  await act(async () =>
    [...container.querySelectorAll<HTMLButtonElement>("button")]
      .find((button) => button.textContent === "add_to_project")!
      .click()
  );
  await act(async () => until(() => tab.nerve.calls.length === 1, "publish saves first"));
  expect(tab.nerve.calls[0]!.method).toBe("PATCH");
  const saved = { ...row, payload: { ...row.payload, name: "publish me" } };
  await act(async () => tab.nerve.calls[0]!.answer(json(200, saved)));
  await act(async () => until(() => tab.nerve.calls.length === 2, "publish request"));
  expect(tab.nerve.calls[1]!.path).toBe("/api/v0/drafts/d/publish");
  expect(tab.nerve.calls[1]!.body).toBeUndefined();
  await act(async () => tab.nerve.calls[1]!.answer(problem(422, "validation_failed")));
  expect(ui.close).not.toHaveBeenCalled();
  expect(tab.app.issue.workspaceDraftIssues.getIssueById("d")).toEqual(saved);
  expect(container.querySelector<HTMLInputElement>('input[name="name"]')!.value).toBe("publish me");
});

it("editsAndDeletesAnUnassignedDraft: the actual row opens an unassigned draft and deletes it with workspace membership", async () => {
  await act(async () => renderer.render(<DraftIssueBlock workspaceSlug="ws" issueId="d" />));
  expect(container.textContent).toContain(row.payload.name);
  await act(async () =>
    container.querySelector("#issue-d")!.dispatchEvent(new MouseEvent("dblclick", { bubbles: true }))
  );
  expect(container.querySelector('[aria-label="description"]')).not.toBeNull();
  await input('input[name="name"]', "edited in row");
  await submit();
  await act(async () => until(() => tab.nerve.calls.length === 1, "row save"));
  await act(async () =>
    tab.nerve.calls[0]!.answer(json(200, { ...row, payload: { ...row.payload, name: "edited in row" } }))
  );
  expect(tab.nerve.calls).toHaveLength(1);
  expect(container.querySelector('[role="dialog"]')).toBeNull();
  await click("delete");
  await click("confirm delete");
  await act(async () => until(() => tab.nerve.calls.length === 2, "delete"));
  expect(tab.nerve.calls[1]).toMatchObject({ method: "DELETE", path: "/api/v0/drafts/d" });
  await act(async () => tab.nerve.calls[1]!.answer(new Response(null, { status: 204 })));
  await act(async () => until(() => tab.nerve.calls.length === 3, "authoritative total"));
  await act(async () => tab.nerve.calls[2]!.answer(json(200, { data: [], total: 0, next_cursor: null })));
  expect(container.querySelector("#issue-d")).toBeNull();
});
it.each([403, 422])("a refused draft deletion leaves the actual dialog and record present (%s)", async (status) => {
  await act(async () => renderer.render(<DraftIssueBlock workspaceSlug="ws" issueId="d" />));
  await click("delete");
  await click("confirm delete");
  await act(async () => until(() => tab.nerve.calls.length === 1, "refused delete"));
  await click("cancel delete");
  expect(container.querySelector('[role="alertdialog"]')).not.toBeNull();
  await act(async () => tab.nerve.calls[0]!.answer(problem(status, "forbidden")));
  expect(container.querySelector('[role="alertdialog"]')).not.toBeNull();
  expect(container.querySelector("#issue-d")).not.toBeNull();
  expect(toasts.at(-1)?.message).toBe("errors.forbidden");
});
async function click(text: string) {
  const button = [...container.querySelectorAll<HTMLButtonElement>("button")].find((item) => item.textContent === text);
  expect(button, text).toBeDefined();
  await act(async () => button!.click());
}

it("a late save after workspace permission loss cannot close, navigate, notify or expose the former draft", async () => {
  await render();
  await input('input[name="name"]', "retired edit");
  await submit();
  await act(async () => until(() => tab.nerve.calls.length === 1, "pending save"));
  const save = tab.nerve.calls[0]!;
  await act(async () => loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 5 }]));
  expect(container.querySelector('[role="dialog"]')).toBeNull();
  await act(async () => loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace]));
  await act(async () => save.answer(json(200, { ...row, payload: { name: "retired edit" } })));
  expect(container.querySelector('[role="dialog"]')).toBeNull();
  expect(ui.close).not.toHaveBeenCalled();
  expect(ui.confirmed).not.toHaveBeenCalled();
  expect(toasts).toHaveLength(0);
});

it("saving a new unassigned draft on close holds the discard dialog and preserves input after refusal", async () => {
  await render(null, false, true);
  await input('input[name="name"]', "keep this draft");
  await click("discard");
  await click("Save to Drafts");
  await act(async () => until(() => tab.nerve.calls.length === 1, "save on close"));
  expect(tab.nerve.calls[0]).toMatchObject({
    method: "POST",
    path: "/api/v0/workspaces/ws/drafts",
    body: { project_id: null, payload: { name: "keep this draft" } },
  });
  await click("Discard");
  expect(ui.close).not.toHaveBeenCalled();
  await click("Cancel");
  expect(ui.close).not.toHaveBeenCalled();
  await act(async () => tab.nerve.calls[0]!.answer(problem(422, "validation_failed")));
  expect(container.textContent).toContain("Save this draft?");
  expect(container.querySelector<HTMLInputElement>('input[name="name"]')!.value).toBe("keep this draft");
  expect(ui.close).not.toHaveBeenCalled();
});
