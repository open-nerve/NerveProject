// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */

import type { Issue } from "@nerve/api-client";
import React, { act, forwardRef, useImperativeHandle } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi, type MockInstance } from "vitest";
import { CreateUpdateIssueModalBase } from "./base";
import type { IssuesModalProps } from "./modal";
import type { RichTextEditor } from "@/components/editor/rich-text";
import { configure, runInAction } from "mobx";
// MobX actions remain real; configurable descriptors allow call-through observations.
configure({ safeDescriptors: false });
import { AxiosHeaders, type AxiosResponse } from "axios";
import { APIService } from "@/services/api.service";
import { heldChange } from "@/lib/auth/fake-tab";
import { EIssuesStoreType } from "@nerve/types";
import { FakeNerve, answered } from "@/lib/auth/fake-nerve";
import { issueOf } from "@/store/issue/fake-issues";
import { workspaceOf, loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { projectOf, loadProjects, stateOf } from "@/store/project/fake-projects";
// Public APIService replies include Axios metadata; entity data remains the complete real response fixture.
function reply<T>(data: T): AxiosResponse<T> {
  return { data, status: 200, statusText: "OK", headers: {}, config: { headers: new AxiosHeaders() } };
}
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: { state: { loginId: "issue-modal-session" } },
  publicClient: {},
}));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
const { RootStore } = await import("@/store/root.store");
let app: InstanceType<typeof RootStore>;
const ws = workspaceOf("ws");
const project = projectOf("A", ws.id);
const a = stateOf(project, "A", "unstarted", 1000);
const b = stateOf(project, "B", "started", 2000);
beforeEach(async () => {
  vi.restoreAllMocks();
  vi.useFakeTimers();
  const nerve = new FakeNerve();
  app = new RootStore(nerve.client());
  binding.app = app;
  runInAction(() => {
    app.issue.workspaceSlug = ws.slug;
    app.issue.projectId = project.id;
    app.router.setQuery({ workspaceSlug: ws.slug, projectId: project.id });
  });
  await loadWorkspaces(nerve, app.workspaceRoot, [ws]);
  await loadProjects(nerve, app.projectRoot.project, ws, [project]);
  await answered(
    nerve,
    () => app.state.fetchProjectStates(project.id),
    ["GET", `/api/v0/projects/${project.id}/states`],
    { data: [a, b] },
    "states"
  );
});
afterEach(() => {
  app.dispose();
  vi.useRealTimers();
});

const ui = vi.hoisted(() => ({ create: vi.fn(), update: vi.fn(), close: vi.fn(), draft: vi.fn() }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws", projectId: "p-a" }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/context/use-issue-modal", () => ({
  useIssueModal: () => ({
    allowedProjectIds: ["p-a", "p-b"],
    selectedParentIssue: null,
    setSelectedParentIssue: () => {},
  }),
}));
vi.mock("@/hooks/store/use-issues", () => ({ useIssues: () => ({ issues: {} }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => app.projectRoot.project }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => app.state }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => app.workspaceRoot }));
vi.mock("@/hooks/store/use-editor-asset", () => ({ useEditorAsset: () => ({}) }));
vi.mock("@/hooks/store/workspace-draft", () => ({
  useWorkspaceDraftIssues: () => ({ moveIssue: vi.fn(), createIssue: ui.draft }),
}));
vi.mock("@/hooks/use-issue-layout-store", () => ({ useIssueStoreType: () => EIssuesStoreType.PROJECT }));
vi.mock("@/hooks/use-issues-actions", () => ({
  useIssuesActions: () => ({ createIssue: ui.create, updateIssue: ui.update }),
}));
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
vi.mock("./components/default-properties", () => ({ IssueDefaultProperties: () => null }));
vi.mock("./components/parent-tag", () => ({ IssueParentTag: () => null }));
vi.mock("@/components/dropdowns/project/dropdown", () => ({
  ProjectDropdown: ({ disabled, value }: { disabled?: boolean; value: string | null }) => (
    <select aria-label="project" disabled={disabled} value={value ?? ""} onChange={() => {}}>
      <option value="p-a">A</option>
      <option value="p-b">B</option>
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
        onChange={(e) => props.onChange?.({}, e.target.value)}
      />
    );
  }),
}));
vi.mock("@nerve/ui", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@nerve/ui")>()),
  ModalCore: ({ children, isOpen }: { children: React.ReactNode; isOpen: boolean }) =>
    isOpen ? <>{children}</> : null,
}));
class NoResize {
  observe() {}
  unobserve() {}
  disconnect() {}
}
let root: Root;
let container: HTMLElement;
let get: MockInstance<APIService["get"]>;
let failure: string | undefined;
let sourceHasHtml: boolean;
let sourceHtml: string;
function core(id: string): Issue {
  return {
    ...issueOf(id, project.id, project.workspace_id),
    parent_id: "parent",
    ...(sourceHasHtml ? { description_html: sourceHtml } : {}),
  };
}
beforeEach(() => {
  vi.useRealTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal("ResizeObserver", NoResize);
  ui.create.mockReset().mockResolvedValue(issueOf("created", project.id, project.workspace_id));
  ui.update.mockReset();
  ui.close.mockReset();
  ui.draft.mockReset().mockResolvedValue(undefined);
  failure = undefined;
  sourceHasHtml = true;
  sourceHtml = "<p>source</p>";
  const detail = app.issue.issueDetail;
  vi.spyOn(detail.activity, "fetchActivities").mockResolvedValue([]);
  vi.spyOn(detail.comment, "fetchComments").mockResolvedValue([]);
  vi.spyOn(detail.subIssues, "fetchSubIssues").mockResolvedValue({
    state_distribution: { backlog: [], unstarted: [], started: [], completed: [], cancelled: [] },
    sub_issues: [],
  });
  vi.spyOn(detail.relation, "fetchRelations").mockResolvedValue({
    blocking: [],
    blocked_by: [],
    duplicate: [],
    relates_to: [],
  });
  vi.spyOn(app.state, "fetchProjectStates").mockResolvedValue([]);
  get = vi.spyOn(APIService.prototype, "get").mockImplementation(async (url) => {
    if (failure && url.endsWith(failure)) throw { response: { data: { code: "resource-refused" } } };
    if (url.endsWith("/reactions/") || url.endsWith("/issue-links/")) return reply([]);
    if (url.endsWith("/issues/parent/")) return reply(issueOf("parent", project.id, project.workspace_id));
    return reply(core(url.split("/").at(-2)!));
  });
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
});
afterEach(async () => {
  await act(async () => root.unmount());
  container.remove();
});
async function render(source = "source", isOpen = true, withDraftIssueWrapper = false) {
  const props: IssuesModalProps = {
    data: { name: "Cloned", project_id: project.id, state_id: a.id },
    sourceIssueId: source,
    isOpen,
    onClose: ui.close,
    withDraftIssueWrapper,
  };
  await act(async () => root.render(<CreateUpdateIssueModalBase {...props} />));
}
async function title(value: string) {
  const input = container.querySelector<HTMLInputElement>("input[name=name]")!;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
async function submit() {
  await act(async () => {
    container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
  });
}
function retry() {
  const retryButton = [...container.querySelectorAll("button")].find((button) => button.textContent === "common.retry");
  expect(retryButton).toBeDefined();
  return retryButton!;
}
it.each(["/reactions/", "/issue-links/", "/issues/parent/"])(
  "real auxiliary refusal shows a retryable error in Base/Form, preserving input and blocking fake-empty submit (%s)",
  async (resource) => {
    failure = resource;
    await render();
    expect(container.querySelector("[role=alert]")).not.toBeNull();
    expect(container.querySelector("textarea[aria-label=description]")).toBeNull();
    expect(container.querySelector<HTMLButtonElement>("button[type=submit]")!.disabled).toBe(true);
    await title("Keep my title");
    await submit();
    expect(ui.create).not.toHaveBeenCalled();
    expect(ui.update).not.toHaveBeenCalled();
    failure = undefined;
    await act(async () => retry().click());
    expect(container.querySelector("[role=alert]")).toBeNull();
    expect(container.querySelector<HTMLInputElement>("input[name=name]")!.value).toBe("Keep my title");
    expect(container.querySelector<HTMLTextAreaElement>("textarea[aria-label=description]")!.value).toBe(
      "<p>source</p>"
    );
    await submit();
    expect(ui.create).toHaveBeenCalledExactlyOnceWith(
      project.id,
      expect.objectContaining({ name: "Keep my title", description_html: "<p>source</p>" })
    );
  }
);
it.each(["resolve", "reject"])(
  "an old source settling after replacement cannot change current form or load state (%s)",
  async (settling) => {
    const old = heldChange<AxiosResponse<Issue>>();
    const fallback = get.getMockImplementation()!;
    get.mockImplementation((url, ...args) => (url.endsWith("/issues/old/") ? old.sent : fallback(url, ...args)));
    await render("old");
    await title("Current input");
    sourceHtml = "<p>current</p>";
    await render("current");
    await act(async () => {
      if (settling === "resolve") old.answer(reply({ ...core("old"), description_html: "<p>stale</p>" }));
      else old.refuse({ response: { data: { code: "old-refused" } } });
    });
    expect(container.querySelector("[role=alert]")).toBeNull();
    expect(container.querySelector<HTMLInputElement>("input[name=name]")!.value).toBe("Current input");
    expect(container.querySelector<HTMLTextAreaElement>("textarea[aria-label=description]")!.value).toBe(
      "<p>current</p>"
    );
    await submit();
    expect(ui.create).toHaveBeenCalledWith(
      project.id,
      expect.objectContaining({ name: "Current input", description_html: "<p>current</p>" })
    );
  }
);
it("closing during load consumes its late failure and reopening can load another source", async () => {
  const old = heldChange<AxiosResponse<Issue>>();
  const fallback = get.getMockImplementation()!;
  get.mockImplementation((url, ...args) => (url.endsWith("/issues/old/") ? old.sent : fallback(url, ...args)));
  await render("old");
  const discard = [...container.querySelectorAll("button")].find((button) => button.textContent === "discard")!;
  await act(async () => discard.click());
  expect(ui.close).toHaveBeenCalledTimes(1);
  await render("old", false);
  await act(async () => old.refuse({ response: { data: { code: "closed-refused" } } }));
  expect(container.querySelector("form")).toBeNull();
  await render("current");
  expect(container.querySelector("[role=alert]")).toBeNull();
  expect(container.querySelector<HTMLTextAreaElement>("textarea[aria-label=description]")!.value).toBe("<p>source</p>");
});
it("a response without actual HTML is an error rather than a successful empty clone", async () => {
  sourceHasHtml = false;
  await render();
  expect(container.querySelector("[role=alert]")).not.toBeNull();
  await submit();
  expect(ui.create).not.toHaveBeenCalled();
});
it("a real empty HTML detail remains a valid editable description", async () => {
  sourceHtml = "";
  await render();
  expect(container.querySelector("[role=alert]")).toBeNull();
  await submit();
  expect(ui.create).toHaveBeenCalledExactlyOnceWith(project.id, expect.objectContaining({ description_html: "" }));
});

it.each(["<p>source</p>", ""])(
  "retry synchronizes real source HTML into the wrapper draft payload (%s)",
  async (html) => {
    sourceHtml = html;
    failure = "/reactions/";
    await render("source", true, true);
    await title("Keep for retry");
    const discard = [...container.querySelectorAll("button")].find((button) => button.textContent === "discard")!;
    await act(async () => discard.click());
    const save = [...container.querySelectorAll("button")].find((button) => button.textContent === "Save to Drafts")!;
    expect(save.disabled).toBe(true);
    expect(ui.draft).not.toHaveBeenCalled();
    const cancel = [...container.querySelectorAll("button")].find((button) => button.textContent === "Cancel")!;
    await act(async () => cancel.click());
    failure = undefined;
    await act(async () => retry().click());
    expect(container.querySelector<HTMLInputElement>("input[name=name]")!.value).toBe("Keep for retry");
    const discardAfterRetry = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "discard"
    )!;
    await act(async () => discardAfterRetry.click());
    const saveAfterRetry = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "Save to Drafts"
    )!;
    expect(saveAfterRetry.disabled).toBe(false);
    await act(async () => saveAfterRetry.click());
    expect(ui.draft).toHaveBeenCalledExactlyOnceWith(
      ws.slug,
      expect.objectContaining({ project_id: project.id, name: "Keep for retry", description_html: html })
    );
  }
);
