// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { configure, runInAction } from "mobx";
// MobX actions remain real; configurable descriptors allow call-through observations.
configure({ safeDescriptors: false });
import React, { act, useEffect } from "react";
import { createRoot, type Root } from "react-dom/client";
import { InboxIssueCreateRoot } from "./create-root";
import type { InboxIssueDescription } from "./issue-description";
import type { TInboxIssue } from "@nerve/types";
import { EInboxIssueStatus, EInboxIssueSource } from "@nerve/types";
import { AxiosHeaders, type AxiosResponse } from "axios";
import { heldChange } from "@/lib/auth/fake-tab";
import { InboxIssueService } from "@/services/inbox";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { issueOf } from "@/store/issue/fake-issues";
import { workspaceOf, loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { projectOf, loadProjects } from "@/store/project/fake-projects";
// Public APIService replies include Axios metadata; entity data remains the complete real response fixture.
function reply<T>(data: T): AxiosResponse<T> {
  return { data, status: 200, statusText: "OK", headers: {}, config: { headers: new AxiosHeaders() } };
}
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
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

beforeEach(async () => {
  vi.restoreAllMocks();
  vi.useFakeTimers();
  const nerve = new FakeNerve();
  app = new RootStore(nerve.client());
  binding.app = app;
  runInAction(() => {
    app.router.setQuery({ workspaceSlug: ws.slug, projectId: project.id });
  });
  await loadWorkspaces(nerve, app.workspaceRoot, [ws]);
  await loadProjects(nerve, app.projectRoot.project, ws, [project]);
  vi.useRealTimers();
});
afterEach(() => {
  app.dispose();
  vi.useRealTimers();
});

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
const ui = vi.hoisted(() => ({ navigate: vi.fn(), close: vi.fn(), clear: vi.fn() }));
vi.mock("react-router", () => ({ useNavigate: () => ui.navigate }));
vi.mock("@/hooks/store/use-project-inbox", () => ({ useProjectInbox: () => app.projectInbox }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => app.workspaceRoot }));
vi.mock("./issue-properties", () => ({ InboxIssueProperties: () => null }));
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
class NoResize {
  observe() {}
  unobserve() {}
  disconnect() {}
}
let root: Root;
let container: HTMLElement;
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal("ResizeObserver", NoResize);
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
        isDuplicateModalOpen={false}
        handleDuplicateIssueModal={() => {}}
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
function envelope(): TInboxIssue {
  return {
    id: "inbox-row",
    status: EInboxIssueStatus.PENDING,
    snoozed_till: null,
    duplicate_to: undefined,
    source: EInboxIssueSource.IN_APP,
    issue: issueOf("R1", project.id, project.workspace_id),
    created_by: "actor",
    duplicate_issue_detail: undefined,
  };
}
it.each(["<p>create</p>", ""])(
  "sends form HTML through the real store/service POST with no provisional entity (%s)",
  async (html) => {
    await fill(html);
    const http = heldChange<AxiosResponse<TInboxIssue>>();
    const post = vi.spyOn(InboxIssueService.prototype, "post").mockReturnValue(http.sent);
    await act(async () => {
      container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    });
    expect(post).toHaveBeenCalledExactlyOnceWith(`/api/workspaces/${ws.slug}/projects/${project.id}/inbox-issues/`, {
      source: EInboxIssueSource.IN_APP,
      issue: {
        name: "New intake",
        description_html: html || "<p></p>",
        priority: "none",
        label_ids: [],
        assignee_ids: [],
        target_date: null,
      },
    });
    expect(app.issue.issues.getIssueById("R1")).toBeUndefined();
    expect(ui.navigate).not.toHaveBeenCalled();
    expect(ui.close).not.toHaveBeenCalled();
    await act(async () => {
      http.answer(reply(envelope()));
    });
    expect(app.issue.issues.getIssueById("R1")).toBeUndefined();
    expect(app.projectInbox.getIssueInboxByIssueId("R1")).toBeDefined();
    expect(app.projectInbox.inboxIssueIds).toEqual(["R1"]);
    expect(app.projectInbox.inboxIssuePaginationInfo?.total_results).toBe(1);
    expect(ui.navigate).toHaveBeenCalledTimes(1);
    expect(ui.close).toHaveBeenCalledTimes(1);
  }
);
it("real HTTP rejection is extracted and consumed by the component without success actions", async () => {
  await fill("<p>retry</p>");
  const error = { code: "create-refused" };
  const log = vi.spyOn(console, "error").mockImplementation(() => {});
  vi.spyOn(InboxIssueService.prototype, "post").mockRejectedValue({ response: { data: error } });
  await act(async () => {
    container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
  });
  expect(log).toHaveBeenCalledWith(error);
  expect(ui.navigate).not.toHaveBeenCalled();
  expect(ui.close).not.toHaveBeenCalled();
  expect(ui.clear).not.toHaveBeenCalled();
  expect(Object.keys(app.issue.issues.issuesMap)).toEqual([]);
  log.mockRestore();
});
