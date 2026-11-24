// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { runInAction } from "mobx";
import React, { act, useEffect } from "react";
import { createRoot, type Root } from "react-dom/client";
import { InboxIssueCreateRoot } from "./create-root";
import type { InboxIssueDescription } from "./issue-description";
import type { InboxIssueProperties } from "./issue-properties";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { detailOf, listPath, loadList, open, pageOf } from "@/store/issue/intake/fake-intake";
import { workspaceOf, loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { projectOf, loadProjects } from "@/store/project/fake-projects";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "intake" } }, publicClient: {} }));
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
/** nerve's answer to the submission: the new work item, its intake pending. */
const submitted = detailOf("new", { project_id: project.id, workspace_id: ws.id });

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
  nerve.calls.length = 0;
  toasts.length = 0;
});
afterEach(() => {
  app.dispose();
  vi.useRealTimers();
});

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/ui", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@nerve/ui")>()),
  ModalTitle: ({ children }: { children: React.ReactNode }) => <h3>{children}</h3>,
}));
const ui = vi.hoisted(() => ({ navigate: vi.fn(), close: vi.fn(), clear: vi.fn() }));
vi.mock("react-router", () => ({ useNavigate: () => ui.navigate }));
vi.mock("@/hooks/store/use-intake", () => ({ useIntake: () => app.issue.intake }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => app.workspaceRoot }));
vi.mock("./issue-properties", () => ({
  InboxIssueProperties: (props: React.ComponentProps<typeof InboxIssueProperties>) => (
    <button type="button" aria-label="due date" onClick={() => props.handleData("target_date", "2026-12-01")} />
  ),
}));
vi.mock("./issue-description", () => ({
  InboxIssueDescription: (props: React.ComponentProps<typeof InboxIssueDescription>) => {
    useEffect(() => {
      Object.assign(props.editorRef, { current: { isEditorReadyToDiscard: () => true, clearEditor: ui.clear } });
    }, [props.editorRef]);
    return (
      <textarea
        aria-label="description"
        value={props.data.description_html}
        onChange={(e) => props.handleData("description_html", e.target.value)}
      />
    );
  },
}));
let root: Root;
let container: HTMLElement;
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  ui.navigate.mockReset();
  ui.close.mockReset();
  ui.clear.mockReset();
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
  await act(async () =>
    root.render(
      <InboxIssueCreateRoot
        workspaceSlug={ws.slug}
        projectId={project.id}
        handleModalClose={ui.close}
        onSubmittingChange={() => {}}
      />
    )
  );
});
afterEach(async () => {
  await act(async () => root.unmount());
  container.remove();
});
async function fill(html: string) {
  const input = container.querySelector<HTMLInputElement>("input[name=name]")!;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "New intake");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  const description = container.querySelector<HTMLTextAreaElement>("textarea")!;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")!.set!.call(description, html);
    description.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
/** Submits the form; waits till its request is out. */
async function submit() {
  await act(async () => {
    container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await until(() => nerve.calls.length === 1, "the submission");
  });
}
/** nerve gives the submission response; waits till the form has followed it (the done condition). */
async function answer(response: Response, done: () => boolean) {
  await act(async () => {
    nerve.calls[0]!.answer(response);
    await until(done, "the form's follow-up");
    await vi.advanceTimersByTimeAsync(100);
  });
}
it.each(["<p>create</p>", ""])(
  "submits the form through the generated client as the intake's body, no date key without a date, and opens the answer (%s)",
  async (html) => {
    await loadList(nerve, app.issue.intake, project.id, open, pageOf([]));
    nerve.calls.length = 0;
    await fill(html);
    await submit();
    expect(nerve.calls[0]).toMatchObject({ method: "POST", path: listPath(project.id) });
    expect(nerve.calls[0]?.body).toStrictEqual({
      name: "New intake",
      description_html: html || "<p></p>",
      priority: "none",
      label_ids: [],
      assignee_ids: [],
    });
    expect(app.issue.issues.getIssueById(submitted.id)).toBeUndefined();
    expect(ui.navigate).not.toHaveBeenCalled();
    expect(ui.close).not.toHaveBeenCalled();
    await answer(json(201, submitted), () => ui.close.mock.calls.length > 0);
    expect(app.issue.issues.getIssueById(submitted.id)).toBeUndefined();
    // the intake list held reads its first page again
    expect(nerve.calls.slice(1)).toMatchObject([
      { method: "GET", path: listPath(project.id), query: { status: "pending" } },
    ]);
    expect(ui.navigate).toHaveBeenCalledExactlyOnceWith(
      `/${ws.slug}/projects/${project.id}/intake?currentTab=open&inboxIssueId=${submitted.id}`
    );
    expect(ui.close).toHaveBeenCalledTimes(1);
  }
);
it("sends the day chosen as a date string", async () => {
  await fill("<p>due</p>");
  await act(async () => container.querySelector<HTMLButtonElement>("button[aria-label='due date']")!.click());
  await submit();
  expect(nerve.calls[0]?.body).toStrictEqual({
    name: "New intake",
    description_html: "<p>due</p>",
    priority: "none",
    label_ids: [],
    assignee_ids: [],
    target_date: "2026-12-01",
  });
});
it("says nerve's refusal through the refusal toast, without console.error and without a success's actions", async () => {
  await fill("<p>retry</p>");
  const log = vi.spyOn(console, "error");
  await submit();
  await answer(problem(409, "issue.intake_disabled"), () => toasts.length > 0);
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.issue_intake_disabled" }]);
  expect(log).not.toHaveBeenCalled();
  expect(ui.navigate).not.toHaveBeenCalled();
  expect(ui.close).not.toHaveBeenCalled();
  expect(ui.clear).not.toHaveBeenCalled();
  expect(Object.keys(app.issue.issues.issuesMap)).toEqual([]);
  log.mockRestore();
});
