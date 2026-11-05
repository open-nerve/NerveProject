// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { runInAction } from "mobx";
import { AxiosHeaders, type AxiosResponse } from "axios";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { TInboxIssue } from "@nerve/types";
import { EInboxIssueSource, EInboxIssueStatus } from "@nerve/types";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { StoreContext } from "@/lib/store-context";
import { APIService } from "@/services/api.service";
import { issueOf } from "@/store/issue/fake-issues";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import type { RootStore } from "@/store/root.store";
import { InboxIssueCreateRoot } from "./create-root";

const binding = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return binding.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "inbox" } }, publicClient: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
// The real editor, without the lazy boundary the intake form loads it through.
vi.mock("@/components/editor/rich-text/editor", async () => ({
  RichTextEditor: (await import("@/components/editor/rich-text/editor-implementation")).RichTextEditor,
}));
vi.mock("./issue-properties", () => ({ InboxIssueProperties: () => null }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useNavigate: () => () => undefined,
}));
vi.mock("@/hooks/store/use-project-inbox", () => ({ useProjectInbox: () => binding.app!.projectInbox }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => binding.app!.workspaceRoot }));
class NoResize {
  observe() {}
  unobserve() {}
  disconnect() {}
}
const ws = workspaceOf("ws");
const project = projectOf("A", ws.id);
let nerve: FakeNerve;
let app: RootStore;
let host: HTMLElement;
let root: Root;
const raw: unknown[] = [];
const editor = () => host.querySelector<HTMLElement>('.tiptap[contenteditable="true"]');
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.stubGlobal("ResizeObserver", NoResize);
  vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
  raw.length = 0;
  vi.stubGlobal("fetch", (...args: unknown[]) => {
    raw.push(args);
    return new Promise<Response>(() => {});
  });
  nerve = new FakeNerve();
  const { RootStore } = await import("@/store/root.store");
  app = new RootStore(nerve.client());
  binding.app = app;
  runInAction(() => app.router.setQuery({ workspaceSlug: ws.slug, projectId: project.id }));
  await loadWorkspaces(nerve, app.workspaceRoot, [ws]);
  await loadProjects(nerve, app.projectRoot.project, ws, [project]);
  nerve.calls.length = 0;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  app.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
function reply(data: TInboxIssue): AxiosResponse<TInboxIssue> {
  return { data, status: 200, statusText: "OK", headers: {}, config: { headers: new AxiosHeaders() } };
}
async function pasteImage() {
  const file = new File([new Uint8Array([137, 80, 78, 71])], "diagram.png", { type: "image/png" });
  const event = new Event("paste", { bubbles: true, cancelable: true });
  Object.defineProperty(event, "clipboardData", {
    value: {
      files: [file],
      items: [
        { kind: "file", type: file.type, getAsFile: () => file, webkitGetAsEntry: () => ({ isDirectory: false }) },
      ],
      getData: () => "",
    },
  });
  await act(async () => editor()!.dispatchEvent(event));
}

it("the unopened intake form keeps a pasted image out and sends no file request of any kind", async () => {
  const calls = (["get", "post", "put", "patch", "delete", "request"] as const).map((method) =>
    vi.spyOn(APIService.prototype, method)
  );
  const post = calls[1]!.mockResolvedValue(
    reply({
      id: "inbox-row",
      status: EInboxIssueStatus.PENDING,
      snoozed_till: null,
      duplicate_to: undefined,
      source: EInboxIssueSource.IN_APP,
      issue: issueOf("R1", project.id, project.workspace_id),
      created_by: "actor",
      duplicate_issue_detail: undefined,
    })
  );
  await act(async () =>
    root.render(
      <StoreContext.Provider value={app}>
        <InboxIssueCreateRoot
          workspaceSlug={ws.slug}
          projectId={project.id}
          handleModalClose={() => undefined}
          isDuplicateModalOpen={false}
          handleDuplicateIssueModal={() => undefined}
        />
      </StoreContext.Provider>
    )
  );
  await act(async () => until(() => !!editor(), "description editor"));
  const title = host.querySelector<HTMLInputElement>("input[name=name]")!;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(title, "New intake");
    title.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await pasteImage();
  expect(editor()!.querySelector("image-component, img")).toBeNull();
  await act(async () => {
    host.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
  });
  await act(async () => until(() => post.mock.calls.length === 1, "intake POST"));
  expect(post).toHaveBeenCalledExactlyOnceWith(
    `/api/workspaces/${ws.slug}/projects/${project.id}/inbox-issues/`,
    expect.objectContaining({ issue: expect.objectContaining({ name: "New intake" }) })
  );
  const sent = (post.mock.calls[0]![1] as { issue: { description_html: string } }).issue.description_html;
  expect(sent).not.toMatch(/image-component|<img/);
  // No other old-service call, no generated-client request and no raw transfer happened at any point.
  for (const spy of calls) if (spy !== post) expect(spy).not.toHaveBeenCalled();
  expect(nerve.calls.map((call) => `${call.method} ${call.path}`)).toEqual([]);
  expect(raw).toEqual([]);
});
