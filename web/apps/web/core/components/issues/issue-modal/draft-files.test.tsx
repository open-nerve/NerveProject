// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Draft } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import { imageFile } from "@/store/asset/fake-assets";
import { toasts } from "@/lib/fake-toast";
import { editor, paste } from "../fake-body-images";
import { draftFilesTab } from "./fake-draft-files";

const m = vi.hoisted(() => ({ projects: [] as string[] }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "drafts" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "drafts" }) }));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/components/editor/rich-text", () => import("@/components/editor/rich-text/editor-implementation"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/use-issue-layout-store", () => ({ useIssueStoreType: () => EIssuesStoreType.WORKSPACE_DRAFT }));
vi.mock("@/hooks/use-issues-actions", () => ({ useIssuesActions: () => ({}) }));
vi.mock("@/hooks/context/use-issue-modal", () => ({
  useIssueModal: () => ({ allowedProjectIds: m.projects, selectedParentIssue: null, setSelectedParentIssue: () => {} }),
}));
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
vi.mock("./components/default-properties", () => ({ IssueDefaultProperties: () => null }));
vi.mock("./components/parent-tag", () => ({ IssueParentTag: () => null }));
vi.mock("@/components/dropdowns/project/dropdown", () => ({
  ProjectDropdown: ({ value, onChange }: { value?: string | null; onChange: (value: string | null) => void }) => (
    <select aria-label="project" value={value ?? ""} onChange={(event) => onChange(event.target.value || null)}>
      <option value="">No project</option>
      {m.projects.map((project) => (
        <option key={project} value={project}>
          {project}
        </option>
      ))}
    </select>
  ),
}));
vi.mock("@nerve/ui", async (original) => ({
  ...(await original<typeof import("@nerve/ui")>()),
  ModalTitle: ({ children }: React.PropsWithChildren) => <h3>{children}</h3>,
  ModalCore: ({ children, isOpen }: React.PropsWithChildren<{ isOpen: boolean }>) =>
    isOpen ? <div role="dialog">{children}</div> : null,
}));

let tab: Awaited<ReturnType<typeof draftFilesTab>>;
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
  m.projects = [tab.project.id];
});
afterEach(async () => {
  await tab.dispose();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
const image = (id: string) => `<image-component src="${id}"></image-component>`;
type Node = { type: { name: string }; attrs: Record<string, unknown>; nodeSize: number };
type Mounted = {
  state: {
    doc: { descendants(visit: (node: Node, pos: number) => void): void };
    tr: { delete(from: number, to: number): unknown };
  };
  view: { dispatch(tr: unknown): void };
};
const removePlaceholder = () =>
  act(async () => {
    const mounted = (editor() as HTMLElement & { editor: Mounted }).editor;
    let found: { node: Node; pos: number } | undefined;
    mounted.state.doc.descendants((node, pos) => {
      if (node.type.name === "imageComponent" && !tab.imageID.test(String(node.attrs.src ?? ""))) found = { node, pos };
    });
    expect(found, "image placeholder").toBeDefined();
    mounted.view.dispatch(mounted.state.tr.delete(found!.pos, found!.pos + found!.node.nodeSize));
  });
const fileRequests = () =>
  tab.nerve.calls.filter(
    (call) => call.path.startsWith("/api/v0/assets") || (call.method === "POST" && call.path.includes("/drafts"))
  );

it.each(["work item", "draft"] as const)(
  "a new %s form sends no file request and creates no draft before one is explicitly saved",
  async (kind) => {
    await tab.render(
      kind === "draft"
        ? { isOpen: true, isDraft: true, onClose: tab.close }
        : {
            isOpen: true,
            data: { name: "New", project_id: tab.project.id, description_html: "<p>New</p>" },
            storeType: EIssuesStoreType.PROJECT,
            onClose: tab.close,
          }
    );
    await tab.waitFor(() => !!editor(), "new form editor");
    const pasted = await paste(undefined, [imageFile()]);
    await act(async () => vi.advanceTimersByTimeAsync(2000));
    expect(pasted.defaultPrevented).toBe(false);
    expect(tab.files()).toBeNull();
    expect(document.body.textContent).toContain("content_files.draft.files_after_save");
    expect(fileRequests()).toEqual([]);
  }
);

it("an existing draft without a project uploads an attachment and an image, saves the image by UUID and shows both again", async () => {
  const draft = await tab.seed({ project_id: null });
  await tab.openDraft(draft.id);
  await tab.waitFor(() => tab.calls("GET", `/api/v0/drafts/${draft.id}/attachments`).length === 1, "attachments");
  await tab.uploadAttachment();
  await tab.waitFor(() => tab.puts.length === 1, "attachment PUT");
  expect(tab.calls("POST", "/api/v0/assets")[0]!.body).toMatchObject({
    purpose: "DRAFT_ISSUE_ATTACHMENT",
    scope: { kind: "draft", id: draft.id },
  });
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 200 })));
  await tab.waitFor(() => tab.files()!.textContent!.includes("notes.txt"), "listed attachment");
  await paste(undefined, [imageFile()]);
  await tab.waitFor(() => tab.puts.length === 2, "image PUT");
  expect(tab.calls("POST", "/api/v0/assets")[1]!.body).toMatchObject({
    purpose: "DRAFT_ISSUE_DESCRIPTION",
    scope: { kind: "draft", id: draft.id },
  });
  await act(async () => tab.puts[1]!.answer(new Response(null, { status: 200 })));
  await tab.waitFor(() => !!document.querySelector("img.image-component"), "confirmed image");
  await tab.submit();
  await tab.waitFor(() => tab.calls("PATCH", `/api/v0/drafts/${draft.id}`).length === 1, "draft save");
  const saved = tab.calls("PATCH", `/api/v0/drafts/${draft.id}`)[0]!.body as Draft;
  const uploaded = [...tab.assets.values()].find((asset) => asset.purpose === "DRAFT_ISSUE_DESCRIPTION")!;
  expect(saved.payload.description_html).toContain(`src="${uploaded.id}"`);
  expect(saved.payload.description_html).not.toMatch(/grant=|blob:|\/static\//);
  await tab.waitFor(() => tab.close.mock.calls.length === 1, "closed after save");
  await act(async () => tab.renderer.render(null));
  tab.nerve.calls.length = 0;
  await tab.openDraft(draft.id);
  await tab.waitFor(() => !!document.querySelector("img.image-component"), "reopened image");
  await tab.waitFor(() => tab.files()?.textContent?.includes("notes.txt") === true, "reopened attachment");
  expect(tab.writes()).toEqual([]);
});

it.each(["attachment", "image"] as const)(
  "an unfinished %s transfer blocks moving and publishing until it finishes or is explicitly cancelled",
  async (kind) => {
    const draft = await tab.seed({ project_id: null });
    await tab.openDraft(draft.id, true);
    await tab.waitFor(() => !!tab.files(), "draft files");
    if (kind === "attachment") await tab.uploadAttachment();
    else await paste(undefined, [imageFile()]);
    await tab.waitFor(() => tab.puts.length === 1, "pending PUT");
    await tab.chooseProject(tab.project.id);
    await tab.submit();
    await tab.click("add_to_project");
    await act(async () => vi.advanceTimersByTimeAsync(100));
    expect(tab.calls("PATCH", /\/drafts\//)).toEqual([]);
    expect(tab.calls("POST", /\/publish$/)).toEqual([]);
    expect(toasts.map((toast) => toast.message)).toContain(
      kind === "attachment" ? "content_files.draft.transfer_busy" : "editor_is_not_ready_to_discard_changes"
    );
    expect(tab.files()).not.toBeNull();
    if (kind === "attachment") await tab.click("content_files.cancel_upload", tab.files()!);
    else await tab.click("content_files.image.cancel");
    expect(tab.puts[0]!.signal?.aborted).toBe(true);
    // A cancelled image leaves its empty placeholder, which the user removes before the draft can be saved.
    if (kind === "image") await removePlaceholder();
    await tab.submit();
    await tab.waitFor(() => tab.calls("PATCH", `/api/v0/drafts/${draft.id}`).length === 1, "move after cancel");
    expect(tab.calls("PATCH", `/api/v0/drafts/${draft.id}`)[0]!.body).toMatchObject({ project_id: tab.project.id });
    expect(tab.calls("DELETE", /\/api\/v0\/assets\//)).toEqual([]);
  }
);

it("moving a draft A→B→A retires each content lifetime before the request and never reuses an earlier read", async () => {
  const stored = "33333333-3333-4333-8333-000000000001";
  const draft = await tab.seed(
    {
      project_id: tab.project.id,
      payload: { name: "Draft", description_html: `<p data-id="base">Body</p>${image(stored)}` },
    },
    [stored]
  );
  tab.control.hold = (call) => call.path.endsWith("/content") || call.method === "PATCH";
  const move = async (target: string | null) => {
    tab.held.length = 0;
    await tab.openDraft(draft.id);
    await tab.waitFor(() => tab.held.some((call) => call.path.endsWith("/content")), "held image read");
    const read = tab.held.find((call) => call.path.endsWith("/content"))!;
    await tab.chooseProject(target);
    await tab.submit();
    await tab.waitFor(() => tab.held.some((call) => call.method === "PATCH"), "held move");
    // The old lifetime ended before the move left: its read is cancelled and nothing of it remains on the page.
    expect(read.aborted()).toBe(true);
    expect(tab.files()).toBeNull();
    expect(document.querySelector('img[src*="grant="]')).toBeNull();
    const sent = tab.held.find((call) => call.method === "PATCH")!;
    expect(sent.body).toMatchObject({ project_id: target });
    await act(async () => tab.control.update(sent, tab.drafts().find((row) => row.id === draft.id)!));
    await tab.waitFor(() => tab.close.mock.calls.length > 0, "closed after move");
    tab.close.mockClear();
    await act(async () => tab.renderer.render(null));
  };
  await move(null);
  await move(tab.project.id);
  expect(tab.calls("GET", `/api/v0/assets/${stored}/content`)).toHaveLength(2);
  expect(tab.calls("PATCH", `/api/v0/drafts/${draft.id}`)).toHaveLength(2);
});

it("a lost prepare answer is found and explicitly discarded before the saved draft publishes once", async () => {
  const stored = "33333333-3333-4333-8333-000000000002";
  const draft = await tab.seed(
    {
      project_id: tab.project.id,
      payload: { name: "Draft", description_html: `<p data-id="base">Body</p>${image(stored)}` },
    },
    [stored]
  );
  let lost = true;
  const prepare = tab.control.prepare;
  tab.control.prepare = (call) => {
    if (!lost) return prepare(call);
    lost = false;
    // The server keeps the pending row; its answer never reaches the page.
    prepare({ ...call, answer: () => {} } as typeof call);
    call.fail();
  };
  await tab.openDraft(draft.id, true);
  await tab.waitFor(() => !!tab.files(), "draft files");
  await tab.uploadAttachment();
  await tab.waitFor(() => tab.calls("POST", "/api/v0/assets").length === 1, "lost prepare");
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(tab.calls("DELETE", /\/api\/v0\/assets\//)).toEqual([]);
  const attachmentsRead = () => tab.calls("GET", `/api/v0/drafts/${draft.id}/attachments`).length;
  const before = attachmentsRead();
  await tab.click("add_to_project");
  await tab.waitFor(() => tab.calls("POST", `/api/v0/drafts/${draft.id}/publish`).length === 1, "refused publish");
  expect(tab.calls("PATCH", `/api/v0/drafts/${draft.id}`)).toHaveLength(1);
  // The refusal keeps the form; the draft's files come back under a fresh lifetime and point to the review.
  await tab.waitFor(() => attachmentsRead() === before + 1, "fresh attachment read");
  expect(document.body.textContent).toContain("content_files.draft.publish_blocked");
  await tab.click("content_files.preparations.open", tab.files()!);
  await tab.waitFor(() => tab.files()!.textContent!.includes("notes.txt"), "unknown pending file");
  await tab.click("content_files.discard notes.txt");
  await tab.click("content_files.confirm_discard");
  await tab.waitFor(() => tab.calls("DELETE", /\/api\/v0\/assets\//).length === 1, "explicit discard");
  await tab.click("add_to_project");
  await tab.waitFor(() => tab.issues.length === 1, "published");
  expect(tab.calls("POST", `/api/v0/drafts/${draft.id}/publish`)).toHaveLength(2);
  expect(tab.calls("PATCH", `/api/v0/drafts/${draft.id}`)).toHaveLength(2);
  // The save before publishing carries the complete payload, the draft's own image identity included.
  expect(tab.calls("PATCH", `/api/v0/drafts/${draft.id}`)[1]!.body).toMatchObject({
    project_id: tab.project.id,
    payload: { name: "Draft", description_html: expect.stringContaining(`src="${stored}"`) },
  });
  // Publishing moves the files on the server; the page sends no per-file copy or move.
  expect(tab.calls("POST", /\/duplicate$/)).toEqual([]);
  await tab.waitFor(() => tab.close.mock.calls.length === 1, "closed after publish");
  const after = tab.nerve.calls.length;
  await act(async () => vi.advanceTimersByTimeAsync(2000));
  expect(tab.nerve.calls.slice(after).filter((call) => call.path.includes(draft.id))).toEqual([]);
});
