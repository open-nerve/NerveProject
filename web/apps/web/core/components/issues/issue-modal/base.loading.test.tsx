// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */

import { heldChange } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { ApiError } from "@/lib/api-error";
import type { Issue } from "@nerve/api-client";
import React, { act, forwardRef, useImperativeHandle } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi, type Mock } from "vitest";
import { CreateUpdateIssueModalBase } from "./base";
import type { IssuesModalProps } from "./modal";
import type { RichTextEditor } from "@/components/editor/rich-text";
import { configure, runInAction } from "mobx";
// MobX actions remain real; configurable descriptors allow call-through observations.
configure({ safeDescriptors: false });
import { EIssuesStoreType } from "@nerve/types";
import { FakeNerve, answered, json, problem, type Call } from "@/lib/auth/fake-nerve";
import { issueOf } from "@/store/issue/fake-issues";
import { workspaceOf, loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { projectOf, loadProjects, stateOf } from "@/store/project/fake-projects";
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
let nerve: FakeNerve;
const ws = workspaceOf("ws");
const project = projectOf("A", ws.id);
const a = stateOf(project, "A", "unstarted", 1000);
const b = stateOf(project, "B", "started", 2000);
beforeEach(async () => {
  vi.restoreAllMocks();
  vi.useFakeTimers();
  nerve = new FakeNerve();
  app = new RootStore(nerve.client());
  binding.app = app;
  runInAction(() => {
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
  ModalTitle: ({ children }: React.PropsWithChildren) => <h3>{children}</h3>,
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
let failure: string | undefined;
let native: Mock<(call: Call) => void>;
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
  vi.spyOn(detail.subIssues, "fetchSubIssues").mockResolvedValue([]);
  vi.spyOn(detail.relation, "fetchRelations").mockResolvedValue([]);
  vi.spyOn(app.state, "fetchProjectStates").mockResolvedValue([]);
  native = vi.fn((call: Call) => {
    if (call.method !== "GET" || !call.path.startsWith("/api/v0/issues/")) return;
    const id = call.path.split("/").at(-1)!;
    if (id === "links") call.answer(failure === "/links" ? problem(403, "resource-refused") : json(200, { data: [] }));
    else if (id === "reactions")
      call.answer(failure === "/reactions" ? problem(403, "resource-refused") : json(200, { data: [] }));
    else if (failure === "/issues/parent/" && id === "parent") call.answer(problem(403, "resource-refused"));
    else if (id === "parent")
      call.answer(
        json(200, { ...issueOf("parent", project.id, project.workspace_id), description_html: "<p>parent</p>" })
      );
    else if (!app.issue.verification.isUncertain(id)) call.answer(json(200, core(id)));
  });
  nerve.respond = (call) => native(call);
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
it.each(["/reactions", "/links", "/issues/parent/"])(
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
    const fallback = native.getMockImplementation()!;
    native.mockImplementation((call) => {
      if (call.path !== "/api/v0/issues/old") fallback(call);
    });
    await render("old");
    await title("Current input");
    sourceHtml = "<p>current</p>";
    await render("current");
    await act(async () => {
      if (settling === "resolve")
        nerve.to("/api/v0/issues/old")[0]!.answer(json(200, { ...core("old"), description_html: "<p>stale</p>" }));
      else nerve.to("/api/v0/issues/old")[0]!.answer(problem(403, "old-refused"));
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
  const fallback = native.getMockImplementation()!;
  native.mockImplementation((call) => {
    if (call.path !== "/api/v0/issues/old") fallback(call);
  });
  await render("old");
  const discard = [...container.querySelectorAll("button")].find((button) => button.textContent === "discard")!;
  await act(async () => discard.click());
  expect(ui.close).toHaveBeenCalledTimes(1);
  await render("old", false);
  await act(async () => nerve.to("/api/v0/issues/old")[0]!.answer(problem(403, "closed-refused")));
  expect(container.querySelector("form")).toBeNull();
  await render("current");
  expect(container.querySelector("[role=alert]")).toBeNull();
  expect(container.querySelector<HTMLTextAreaElement>("textarea[aria-label=description]")!.value).toBe("<p>source</p>");
});
it("a response without actual HTML is an error rather than a successful empty clone", async () => {
  sourceHasHtml = false;
  await render();
  const authority = nerve.calls.at(-1)!;
  expect(authority.path).toBe("/api/v0/issues/source");
  await act(async () => authority.answer(json(200, core("source"))));
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
    failure = "/reactions";
    await render("source", true, true);
    await title("Keep for retry");
    const discard = [...container.querySelectorAll("button")].find((button) => button.textContent === "discard")!;
    await act(async () => discard.click());
    const save = [...container.querySelectorAll("button")].find((button) => button.textContent === "save_to_drafts")!;
    expect(save.disabled).toBe(true);
    expect(ui.draft).not.toHaveBeenCalled();
    const cancel = [...container.querySelectorAll("button")].find((button) => button.textContent === "cancel")!;
    await act(async () => cancel.click());
    failure = undefined;
    await act(async () => retry().click());
    expect(container.querySelector<HTMLInputElement>("input[name=name]")!.value).toBe("Keep for retry");
    const discardAfterRetry = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "discard"
    )!;
    await act(async () => discardAfterRetry.click());
    const saveAfterRetry = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "save_to_drafts"
    )!;
    expect(saveAfterRetry.disabled).toBe(false);
    await act(async () => saveAfterRetry.click());
    expect(ui.draft).toHaveBeenCalledExactlyOnceWith(
      ws.slug,
      expect.objectContaining({
        project_id: project.id,
        payload: expect.objectContaining({ name: "Keep for retry", description_html: html }),
      })
    );
  }
);

it.each(["closed", "source", "unmounted"])(
  "does not follow an old ordinary creation after its form is %s",
  async (scope) => {
    toasts.length = 0;
    const pending = heldChange<Issue>();
    ui.create.mockImplementation(() => pending.sent);
    await render();
    await submit();
    expect(ui.create).toHaveBeenCalledTimes(1);
    if (scope === "closed") await render("source", false);
    else if (scope === "source") await render("another");
    else await act(async () => root.render(null));
    await act(async () => pending.answer(issueOf("created", project.id, project.workspace_id)));
    expect(ui.close).not.toHaveBeenCalled();
    expect(toasts).toEqual([]);
  }
);
it("does not report an ordinary creation refusal from a previous source", async () => {
  toasts.length = 0;
  const pending = heldChange<Issue>();
  ui.create.mockImplementation(() => pending.sent);
  await render();
  await submit();
  await render("another");
  await act(async () => pending.refuse(new ApiError(403, { status: 403, code: "forbidden", title: "Forbidden" })));
  expect(toasts).toEqual([]);
  expect(ui.close).not.toHaveBeenCalled();
});
