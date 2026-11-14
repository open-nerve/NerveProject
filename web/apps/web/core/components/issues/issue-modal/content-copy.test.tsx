// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act } from "react";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { DraftCreate, DraftUpdate, IssueCreate } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import { StoreContext } from "@/lib/store-context";
import { contentAssetOf } from "@/store/asset/fake-assets";
import { cycleRequests, rowOf } from "@/store/planning/fake-cycles";
import { moduleRequests, moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, moduleOf } from "@/store/project/fake-projects";
import { json, problem, type Call } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import { editor } from "../fake-body-images";
import { DraftIssueBlock } from "../workspace-draft/draft-issue-block";
import { CreateUpdateIssueModalBase } from "./base";
import type { IssuesModalProps } from "./modal";
import { draftFilesTab } from "./fake-draft-files";

type Tab = Awaited<ReturnType<typeof draftFilesTab>>;
const m = vi.hoisted(() => ({ projects: [] as string[], tab: undefined as unknown }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "copies" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "copies" }) }));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/components/editor/rich-text", () => import("@/components/editor/rich-text/editor-implementation"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/use-issue-layout-store", () => ({ useIssueStoreType: () => EIssuesStoreType.PROJECT }));
vi.mock("@/hooks/use-issues-actions", () => ({
  useIssuesActions: () => ({
    createIssueAsGiven: (project: string, body: IssueCreate) =>
      (m.tab as Tab).app.issue.projectIssues.createIssue("ws", project, body),
  }),
}));
vi.mock("@/hooks/context/use-issue-modal", () => ({
  useIssueModal: () => ({ allowedProjectIds: m.projects, selectedParentIssue: null, setSelectedParentIssue: () => {} }),
}));
vi.mock("@/hooks/store/use-app-theme", () => ({ useAppTheme: () => ({ sidebarCollapsed: false }) }));
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
vi.mock("./components/default-properties", () => ({ IssueDefaultProperties: () => null }));
vi.mock("./components/parent-tag", () => ({ IssueParentTag: () => null }));
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
vi.mock("@/components/dropdowns/project/dropdown", () => ({ ProjectDropdown: () => null }));
vi.mock("@nerve/ui", async (original) => ({
  ...(await original<typeof import("@nerve/ui")>()),
  ModalTitle: ({ children }: React.PropsWithChildren) => <h3>{children}</h3>,
  ModalCore: ({ children, isOpen }: React.PropsWithChildren<{ isOpen: boolean }>) =>
    isOpen ? <div role="dialog">{children}</div> : null,
}));

let tab: Tab;
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  const unlaidOut = document.createElement("div");
  // oxlint-disable-next-line no-extend-native -- declares what the test environment lacks; tests only
  Object.defineProperties(Range.prototype, {
    getClientRects: { configurable: true, value: () => unlaidOut.getClientRects() },
    getBoundingClientRect: { configurable: true, value: () => unlaidOut.getBoundingClientRect() },
  });
  toasts.length = 0;
  tab = await draftFilesTab();
  m.tab = tab;
  m.projects = [tab.project.id];
});
afterEach(async () => {
  await tab.dispose();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
const s1 = "44444444-4444-4444-8444-000000000001";
const s2 = "44444444-4444-4444-8444-000000000002";
const image = (id: string) => `<image-component src="${id}"></image-component>`;
const body = `<p data-id="a">First</p>${image(s1)}<p data-id="b">Second</p>${image(s2)}`;
const creates = () => tab.calls("POST", "/api/v0/workspaces/ws/drafts");
const duplicates = () => tab.calls("POST", /\/duplicate$/);
const publishes = () => tab.calls("POST", /\/drafts\/[^/]+\/publish$/);
const text = () => document.body.textContent ?? "";
/** The ordinary copy as the work item quick action opens it. */
async function copyIssue(html = body) {
  for (const id of [s1, s2]) tab.assets.set(id, contentAssetOf(id, tab.hosts.issue, "description"));
  await tab.render({
    isOpen: true,
    data: { name: "Copy", project_id: tab.project.id, description_html: html },
    sourceIssueId: tab.issue.id,
    storeType: EIssuesStoreType.PROJECT,
    onClose: tab.close,
    onSubmit: tab.confirmed,
  });
  await tab.waitFor(() => !!editor(), "copy editor");
}
/** The draft row's own copy action. */
async function copyDraft() {
  const source = await tab.seed({ project_id: tab.project.id, payload: { name: "Source", description_html: body } }, [
    s1,
    s2,
  ]);
  await act(async () =>
    tab.renderer.render(
      <StoreContext.Provider value={tab.app}>
        <DraftIssueBlock workspaceSlug="ws" issueId={source.id} />
      </StoreContext.Provider>
    )
  );
  await tab.click("make-a-copy");
  await tab.waitFor(() => !!editor(), "draft copy editor");
  return source;
}
/** The one target the copy created, and the server copies of the two images in document order. */
function created() {
  const target = (duplicates()[0]!.body as { scope: { id: string } }).scope.id;
  const copies = [...tab.assets.values()].filter((asset) => asset.scope.kind === "draft" && asset.scope.id === target);
  return { target, copies: copies.map((asset) => asset.id) };
}
function expectMapped(html: string, copies: string[]) {
  expect(html).not.toContain(s1);
  expect(html).not.toContain(s2);
  expect(html.indexOf(copies[0]!)).toBeGreaterThan(html.indexOf("First"));
  expect(html.indexOf(copies[1]!)).toBeGreaterThan(html.indexOf("Second"));
}

it("an ordinary copy with images makes one target draft, copies the images in order and publishes it once as an Issue", async () => {
  await copyIssue();
  await tab.submit();
  await tab.waitFor(() => tab.issues.length === 1, "published copy");
  expect(creates()).toHaveLength(1);
  // The target never references the source images; only its own server copies.
  expect((creates()[0]!.body as DraftCreate).payload.description_html).toBe(
    '<p data-id="a">First</p><p data-id="b">Second</p>'
  );
  expect(duplicates().map((call) => call.path)).toEqual([
    `/api/v0/assets/${s1}/duplicate`,
    `/api/v0/assets/${s2}/duplicate`,
  ]);
  const { target, copies } = created();
  expect(duplicates()[0]!.body).toEqual({ purpose: "DRAFT_ISSUE_DESCRIPTION", scope: { kind: "draft", id: target } });
  const saves = tab.calls("PATCH", `/api/v0/drafts/${target}`);
  expect(saves).toHaveLength(1);
  expectMapped((saves[0]!.body as DraftUpdate).payload!.description_html!, copies);
  expect(publishes().map((call) => call.path)).toEqual([`/api/v0/drafts/${target}/publish`]);
  expect(tab.calls("POST", /\/projects\/[^/]+\/issues$/)).toEqual([]);
  expect(tab.confirmed).toHaveBeenCalledWith(
    expect.objectContaining({ id: tab.issues[0]!.id, sequence_id: expect.any(Number) })
  );
  await tab.waitFor(() => tab.close.mock.calls.length === 1, "closed after the Issue");
});

// P7a spec 2.7, R15: the copy's form starts with its source's cycle and modules that it may join, which go into the
// target draft with the rest of the form and into the work item nerve publishes of it.
it("an ordinary copy with images carries the cycle and modules its form starts with into the target draft, which nerve publishes in them", async () => {
  const now = cycleOf(tab.project, "Now", { status: "current" });
  const core = moduleOf(tab.project, "Core");
  const gone = moduleOf(tab.project, "Gone", { status: "completed", archived_at: "2026-12-01T09:00:00Z" });
  const respond = tab.nerve.respond;
  tab.nerve.respond = (call) => {
    const listed = call.method === "GET" && call.query.archived !== "true";
    if (listed && call.path === cycleRequests.list(tab.project)) call.answer(json(200, { data: [rowOf(now)] }));
    else if (listed && call.path === moduleRequests.list(tab.project))
      call.answer(json(200, { data: [moduleRowOf(core)] }));
    else respond?.(call);
  };
  for (const id of [s1, s2]) tab.assets.set(id, contentAssetOf(id, tab.hosts.issue, "description"));
  // the copy as the quick action opens it, with its source's cycle and modules; an SWR cache of its own, for this
  // file's earlier forms read the same lists in the same session
  await act(async () =>
    tab.renderer.render(
      <StoreContext.Provider value={tab.app}>
        <SWRConfig value={{ provider: () => new Map() }}>
          <CreateUpdateIssueModalBase
            isOpen
            data={{
              name: "Copy",
              project_id: tab.project.id,
              description_html: body,
              cycle_id: now.id,
              module_ids: [core.id, gone.id],
            }}
            sourceIssueId={tab.issue.id}
            storeType={EIssuesStoreType.PROJECT}
            fetchIssueDetails={false}
            withDraftIssueWrapper={false}
            onClose={tab.close}
            onSubmit={tab.confirmed}
          />
        </SWRConfig>
      </StoreContext.Provider>
    )
  );
  await tab.waitFor(() => !!editor(), "copy editor");
  const lists = [cycleRequests.list(tab.project), moduleRequests.list(tab.project)];
  await tab.waitFor(() => lists.every((path) => tab.calls("GET", path).length === 1), "the project's two lists");
  await act(async () => vi.advanceTimersByTimeAsync(20));
  await tab.submit();
  await tab.waitFor(() => tab.issues.length === 1, "published copy");
  expect((creates()[0]!.body as DraftCreate).payload).toMatchObject({ cycle_id: now.id, module_ids: [core.id] });
  expect(publishes()).toHaveLength(1);
  expect(tab.issues[0]).toMatchObject({ cycle_id: now.id, module_ids: [core.id] });
  expect(tab.calls("POST", /\/projects\/[^/]+\/issues$/)).toEqual([]);
});

it("the draft row's copy with images ends as a Draft after one target, ordered copies and one save", async () => {
  const source = await copyDraft();
  await tab.submit();
  await tab.waitFor(() => tab.calls("PATCH", /\/api\/v0\/drafts\//).length === 1, "saved draft copy");
  const { target, copies } = created();
  expect(target).not.toBe(source.id);
  expect(creates()).toHaveLength(1);
  expectMapped(tab.drafts().find((row) => row.id === target)!.payload.description_html!, copies);
  expect(tab.drafts().find((row) => row.id === source.id)!.payload.description_html).toBe(body);
  expect(publishes()).toEqual([]);
  expect(tab.issues).toEqual([]);
});

it.each(["retry", "close"] as const)(
  "a refused image copy keeps the same target and finished copies (%s)",
  async (next) => {
    let refused = false;
    const duplicate = tab.control.duplicate;
    tab.control.duplicate = (call, source) => {
      if (source.id !== s2 || refused) return duplicate(call, source);
      refused = true;
      call.answer(problem(404, "asset.not_found"));
    };
    await copyIssue();
    await tab.submit();
    await tab.waitFor(() => duplicates().length === 2, "refused copy");
    await act(async () => vi.advanceTimersByTimeAsync(100));
    expect(text()).toContain("content_files.copy.kept");
    expect(document.querySelector('a[href="/ws/drafts"]')).not.toBeNull();
    expect(tab.calls("PATCH", /\/api\/v0\/drafts\//)).toEqual([]);
    if (next === "close") {
      await tab.click("discard");
      expect(tab.close).toHaveBeenCalledTimes(1);
      expect(toasts.map((toast) => toast.message)).toContain("content_files.copy.kept_closed");
      expect(creates()).toHaveLength(1);
      return;
    }
    await tab.submit();
    await tab.waitFor(() => tab.issues.length === 1, "published after the retry");
    expect(creates()).toHaveLength(1);
    expect(duplicates().map((call) => call.path)).toEqual([
      `/api/v0/assets/${s1}/duplicate`,
      `/api/v0/assets/${s2}/duplicate`,
      `/api/v0/assets/${s2}/duplicate`,
    ]);
    expect(publishes()).toHaveLength(1);
  }
);

it("an unconfirmed draft creation is checked against the drafts before the user allows another attempt", async () => {
  let lost = true;
  const create = tab.control.create;
  tab.control.create = (call) => {
    if (!lost) return create(call);
    lost = false;
    call.fail();
  };
  await copyIssue();
  await tab.submit();
  await tab.waitFor(() => text().includes("content_files.copy.unknown_create"), "unconfirmed creation");
  await tab.submit();
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(creates()).toHaveLength(1);
  const reads = tab.calls("GET", "/api/v0/workspaces/ws/drafts").length;
  await tab.click("content_files.copy.verify");
  await tab.waitFor(() => tab.calls("GET", "/api/v0/workspaces/ws/drafts").length === reads + 1, "drafts read");
  await tab.click("content_files.copy.allow_retry");
  await tab.submit();
  await tab.waitFor(() => tab.issues.length === 1, "published after the check");
  expect(creates()).toHaveLength(2);
  expect(duplicates()).toHaveLength(2);
});

it.each(["still a draft", "already published"] as const)(
  "an unconfirmed publish of the copy is checked before anything is sent again (%s)",
  async (outcome) => {
    let lost = true;
    const publish = tab.control.publish;
    tab.control.publish = (call, draft) => {
      if (!lost) return publish(call, draft);
      lost = false;
      if (outcome === "already published") publish({ ...call, answer: () => {} } as Call, draft);
      call.fail();
    };
    await copyIssue();
    await tab.submit();
    await tab.waitFor(() => text().includes("content_files.copy.unknown_publish"), "unconfirmed publish");
    await tab.click("content_files.copy.verify");
    if (outcome === "already published") {
      await tab.waitFor(() => text().includes("content_files.copy.published"), "published elsewhere");
      expect(text()).not.toContain("content_files.copy.allow_retry");
      await tab.submit();
      await act(async () => vi.advanceTimersByTimeAsync(100));
      expect(publishes()).toHaveLength(1);
      expect(tab.issues).toHaveLength(1);
      return;
    }
    await tab.waitFor(() => text().includes("content_files.copy.allow_retry"), "still a draft");
    await tab.click("content_files.copy.allow_retry");
    await tab.submit();
    await tab.waitFor(() => tab.issues.length === 1, "published after the check");
    expect(creates()).toHaveLength(1);
    expect(duplicates()).toHaveLength(2);
    expect(publishes()).toHaveLength(2);
  }
);

it.each(["unchecked", "already published"] as const)(
  "closing after an unconfirmed publish says where the copy may be instead of claiming the draft is kept (%s)",
  async (state) => {
    let lost = true;
    const publish = tab.control.publish;
    tab.control.publish = (call, draft) => {
      if (!lost) return publish(call, draft);
      lost = false;
      publish({ ...call, answer: () => {} } as Call, draft);
      call.fail();
    };
    await copyIssue();
    await tab.submit();
    await tab.waitFor(() => text().includes("content_files.copy.unknown_publish"), "unconfirmed publish");
    if (state === "already published") {
      await tab.click("content_files.copy.verify");
      await tab.waitFor(() => text().includes("content_files.copy.published"), "published elsewhere");
    }
    await tab.click("discard");
    expect(tab.close).toHaveBeenCalledTimes(1);
    const messages = toasts.map((toast) => toast.message);
    expect(messages).not.toContain("content_files.copy.kept_closed");
    expect(messages).toContain(
      state === "unchecked" ? "content_files.copy.unknown_publish" : "content_files.copy.published"
    );
  }
);

it("a copy without images keeps the direct work item creation and makes no draft", async () => {
  await copyIssue('<p data-id="a">Only text</p>');
  await tab.submit();
  await tab.waitFor(() => tab.calls("POST", /\/projects\/[^/]+\/issues$/).length === 1, "direct creation");
  expect(creates()).toEqual([]);
  expect(duplicates()).toEqual([]);
});
