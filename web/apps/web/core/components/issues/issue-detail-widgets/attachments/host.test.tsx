// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, StrictMode, type ReactNode } from "react";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, onTestFinished, vi } from "vitest";
import { IssueDetailModals } from "@/store/issue/issue-details/modals";
import type { RootStore } from "@/store/root.store";
import { StoreContext } from "@/lib/store-context";
import { attachmentOf, contentAssetTab, watchLegacyFileTraffic } from "@/store/asset/fake-assets";
import { pickerDOM, waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { json, noContent, problem, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { loadProjects } from "@/store/project/fake-projects";
import { IssueDetailRoot } from "../../issue-detail/root";
import { IssuePeekOverview } from "../../peek-overview/root";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "host" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "host" }) }));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({
  useNavigate: () => vi.fn(),
  Link: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => tab.app.user.permission,
  useUser: () => ({ data: undefined }),
}));
vi.mock("@/hooks/store/use-app-theme", () => ({ useAppTheme: () => ({ issueDetailSidebarCollapsed: true }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/use-window-size", () => ({ default: () => [1024, 768] }));
vi.mock("@/hooks/use-reload-confirmation", () => ({ default: () => ({ setShowAlert: () => undefined }) }));
// Keep the actual main/peek conditional bodies, title, widgets and asset controls; unrelated leaves are inert.
vi.mock("@/components/editor/rich-text/description-input", () => ({ DescriptionInput: () => null }));
vi.mock("@/components/issues/issue-detail/description-versions", () => ({ IssueDescriptionVersions: () => null }));
vi.mock("@/components/issues/issue-detail/issue-identifier", () => ({ IssueIdentifier: () => null }));
vi.mock("@/components/issues/issue-detail/issue-activity", () => ({ IssueActivity: () => null }));
vi.mock("@/components/issues/issue-detail/reactions", () => ({ IssueReaction: () => null }));
vi.mock("@/components/issues/issue-detail/sidebar", () => ({ IssueDetailsSidebar: () => null }));
vi.mock("@/components/issues/peek-overview/properties", () => ({ PeekOverviewProperties: () => null }));
vi.mock("@/components/issues/peek-overview/header", () => ({
  IssuePeekOverviewHeader: ({ removeRoutePeekId }: { removeRoutePeekId(): void }) => (
    <button onClick={removeRoutePeekId}>close peek</button>
  ),
}));
vi.mock("@/components/issues/issue-detail-widgets/action-buttons", () => ({
  IssueDetailWidgetActionButtons: () => null,
}));
vi.mock("@/components/issues/issue-detail-widgets/issue-detail-widget-modals", () => ({
  IssueDetailWidgetModals: () => null,
}));
let tab: Awaited<ReturnType<typeof fixture>>;
let legacy: ReturnType<typeof watchLegacyFileTraffic>;
async function fixture() {
  return pickerDOM(await contentAssetTab());
}
let core: (call: Call) => void;
let page: { data: ReturnType<typeof attachmentOf>[]; next_cursor: string | null };
const path = "/api/v0/issues/issue-1/attachments";
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
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  tab = await fixture();
  legacy = watchLegacyFileTraffic(tab);
  page = { data: [attachmentOf("file-a")], next_cursor: null };
  core = (call) => call.answer(json(200, tab.issue));
  tab.nerve.respond = (call) => {
    if (call.path === "/api/v0/issues/issue-1" && call.method === "GET") core(call);
    else if (call.path === path) call.answer(json(200, page));
    else if (call.path.endsWith("/complete"))
      call.answer(
        json(200, { ...tab.assets.get("asset-1"), state: "uploaded", is_bound: true, width: null, height: null })
      );
    else if (call.path === "/api/v0/assets") tab.answer(call);
    else if (call.method === "GET") call.answer(json(200, { data: [], next_cursor: null }));
  };
});
afterEach(async () => {
  const leaks = legacy.leaks();
  await tab?.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
  expect(leaks).toEqual([]);
});
type Kind = "detail" | "peek";
async function render(kind: Kind, strict = false) {
  if (kind === "peek")
    tab.app.issue.issueDetail.setPeekIssue({ workspaceSlug: "ws", projectId: "A", issueId: tab.issue.id });
  const content = (
    <StoreContext.Provider value={tab.app}>
      <SWRConfig value={{ provider: () => new Map() }}>
        {kind === "detail" ? (
          <IssueDetailRoot workspaceSlug="ws" projectId="A" issueId={tab.issue.id} />
        ) : (
          <IssuePeekOverview hostScope="files" />
        )}
      </SWRConfig>
    </StoreContext.Provider>
  );
  await act(async () => tab.renderer.render(strict ? <StrictMode>{content}</StrictMode> : content));
  await waitForPickerDOM(() => !!document.querySelector('textarea[aria-label="issue.title.label"]'), "title");
}
async function openFiles() {
  const button = [...document.querySelectorAll<HTMLButtonElement>("button")].find((b) =>
    b.textContent?.includes("content_files.title")
  );
  expect(button, "actual attachment widget").toBeDefined();
  await act(async () => button!.click());
  await waitForPickerDOM(() => document.body.textContent?.includes("file-a.txt") ?? false, "files");
}
async function title(value: string) {
  await act(async () => {
    const input = document.querySelector<HTMLTextAreaElement>('textarea[aria-label="issue.title.label"]')!;
    Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")!.set!.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
async function complete() {
  await tab.select([new File(["body"], "note.txt", { type: "text/plain" })]);
  await tab.click("content_files.upload");
  await act(async () => until(() => tab.puts.length === 1, "raw upload"));
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 200 })));
  await waitForPickerDOM(() => tab.nerve.calls.some((call) => call.path.endsWith("/complete")), "complete");
}
for (const kind of ["detail", "peek"] as const) {
  it(`${kind}: complete survives pending and a failed canonical refresh, preserving title and exposing only a read retry`, async () => {
    await render(kind);
    await openFiles();
    await title("unsaved local title");
    let pending: Call | undefined;
    core = (call) => {
      pending = call;
    };
    await complete();
    await waitForPickerDOM(() => !!pending, "canonical verification");
    expect(document.querySelector('textarea[aria-label="issue.title.label"]')).toBeNull();
    expect(document.body.textContent).toContain("content_files.confirmed_upload");
    expect(tab.nerve.calls.filter((c) => c.method === "PATCH")).toHaveLength(0);
    await act(async () => pending!.answer(problem(503, "server_busy")));
    await waitForPickerDOM(() => document.body.textContent?.includes("errors.server_busy") ?? false, "read error");
    core = (call) => call.answer(json(200, { ...tab.issue, attachment_count: 2 }));
    await tab.click("common.retry");
    await waitForPickerDOM(() => !!document.querySelector('textarea[aria-label="issue.title.label"]'), "fresh body");
    expect(document.querySelector<HTMLTextAreaElement>('textarea[aria-label="issue.title.label"]')!.value).toBe(
      "unsaved local title"
    );
    await act(async () => vi.advanceTimersByTimeAsync(3000));
    expect(tab.nerve.calls.filter((c) => c.method === "PATCH")).toHaveLength(0);
    expect(tab.nerve.calls.filter((c) => c.path.endsWith("/complete"))).toHaveLength(1);
    expect(tab.nerve.to("/api/v0/assets")).toHaveLength(1);
  });
  it(`${kind}: a confirmed upload and deletion return the focus the re-read took to the remounted widget`, async () => {
    await render(kind);
    await openFiles();
    await complete();
    await waitForPickerDOM(() => !!document.querySelector('input[type="file"]'), "remounted picker");
    await waitForPickerDOM(() => document.activeElement === document.querySelector('input[type="file"]'), "picker");
    await tab.click("content_files.remove file-a.txt");
    await waitForPickerDOM(() => !!document.querySelector('[role="dialog"]'), "dialog");
    await tab.click("content_files.confirm_delete");
    await act(async () => until(() => tab.nerve.to("/api/v0/assets/file-a").length === 1, "delete"));
    await act(async () => tab.nerve.to("/api/v0/assets/file-a")[0]!.answer(noContent()));
    await waitForPickerDOM(() => !document.querySelector('[role="dialog"]'), "confirmed");
    await waitForPickerDOM(() => document.activeElement === tab.button("content_files.refresh"), "refresh");
  });
  it(`${kind}: a control chosen during the re-read keeps the focus when the widget remounts`, async () => {
    await render(kind);
    await openFiles();
    let pending: Call | undefined;
    core = (call) => {
      pending = call;
    };
    await complete();
    await waitForPickerDOM(() => !!pending, "canonical verification");
    const chosen = document.body.appendChild(document.createElement("button"));
    chosen.focus();
    await act(async () => pending!.answer(json(200, tab.issue)));
    await waitForPickerDOM(() => !!document.querySelector('input[type="file"]'), "remounted picker");
    await act(async () => vi.advanceTimersByTimeAsync(100));
    expect(document.activeElement).toBe(chosen);
    chosen.remove();
  });
  it(`${kind}: an unknown deletion found gone leaves only Close, resends nothing and leaves no unknown notice`, async () => {
    await render(kind);
    await openFiles();
    await tab.click("content_files.remove file-a.txt");
    await waitForPickerDOM(() => !!document.querySelector('[role="dialog"]'), "dialog");
    await tab.click("content_files.confirm_delete");
    await act(async () => until(() => tab.nerve.to("/api/v0/assets/file-a").length === 1, "delete"));
    await act(async () => tab.nerve.to("/api/v0/assets/file-a")[0]!.fail());
    await waitForPickerDOM(() => !!document.querySelector('[role="alert"]'), "unknown");
    page = { data: [], next_cursor: null };
    const respond = tab.nerve.respond!;
    tab.nerve.respond = (call) => {
      if (call.path === "/api/v0/assets/file-a" && call.method === "GET") call.answer(problem(404, "asset.not_found"));
      else respond(call);
    };
    await tab.click("content_files.verify");
    await waitForPickerDOM(() => document.activeElement?.textContent === "content_files.delete_gone", "gone");
    const dialog = document.querySelector('[role="dialog"]')!;
    expect([...dialog.querySelectorAll("button")].map((button) => button.textContent)).toEqual(["close"]);
    await tab.click("close", dialog);
    await waitForPickerDOM(() => !document.querySelector('[role="dialog"]'), "closed");
    expect(document.body.textContent).not.toContain("content_files.unknown_delete");
    expect(document.body.textContent).not.toContain("content_files.confirmed_delete");
    expect(document.body.textContent).not.toContain("file-a.txt");
    expect(tab.nerve.to("/api/v0/assets/file-a").map((call) => call.method)).toEqual(["DELETE", "GET"]);
  });
  it(`${kind}: the confirmed notice is announced once, stays through the remount that returns the focus and ends at the next`, async () => {
    await render(kind);
    await openFiles();
    const shown: Node[] = [];
    const watch = new MutationObserver((records) => {
      for (const record of records)
        for (const node of record.addedNodes)
          if (node.textContent?.includes("content_files.confirmed_upload")) shown.push(node);
    });
    watch.observe(document.body, { childList: true, subtree: true, characterData: true });
    let pending: Call | undefined;
    core = (call) => {
      pending = call;
    };
    await complete();
    await waitForPickerDOM(() => !!pending, "canonical verification");
    expect(document.body.textContent).toContain("content_files.confirmed_upload");
    await act(async () => pending!.answer(json(200, tab.issue)));
    await waitForPickerDOM(() => document.activeElement === document.querySelector('input[type="file"]'), "picker");
    expect(document.body.textContent).toContain("content_files.confirmed_upload");
    pending = undefined;
    let reread!: Promise<unknown>;
    await act(async () => {
      reread = tab.app.issue.issues.refreshAuthoritative(tab.issue.id, { afterWrite: true });
    });
    await waitForPickerDOM(() => !!pending, "later re-read");
    expect(document.body.textContent).not.toContain("content_files.confirmed_upload");
    await act(async () => {
      pending!.answer(json(200, tab.issue));
      await reread;
    });
    await waitForPickerDOM(() => !!document.querySelector('input[type="file"]'), "later remount");
    expect(document.body.textContent).not.toContain("content_files.confirmed_upload");
    watch.disconnect();
    expect(shown).toHaveLength(1);
  });
  it(`${kind}: a new command on the host ends the notice whose focus the user had moved elsewhere`, async () => {
    await render(kind);
    await openFiles();
    const elsewhere = document.body.appendChild(document.createElement("button"));
    onTestFinished(() => elsewhere.remove());
    let pending: Call | undefined;
    core = (call) => {
      pending = call;
    };
    await complete();
    await waitForPickerDOM(() => !!pending, "upload re-read");
    elsewhere.focus();
    await act(async () => pending!.answer(json(200, tab.issue)));
    await waitForPickerDOM(() => !!document.querySelector('input[type="file"]'), "remounted after upload");
    expect(document.activeElement).toBe(elsewhere);
    expect(document.body.textContent).toContain("content_files.confirmed_upload");
    await tab.click("content_files.remove file-a.txt");
    await waitForPickerDOM(() => !!document.querySelector('[role="dialog"]'), "dialog");
    expect(document.body.textContent).toContain("content_files.confirmed_upload");
    await tab.click("content_files.confirm_delete");
    await act(async () => until(() => tab.nerve.to("/api/v0/assets/file-a").length === 1, "delete"));
    expect(document.body.textContent).not.toContain("content_files.confirmed_upload");
    pending = undefined;
    await act(async () => tab.nerve.to("/api/v0/assets/file-a")[0]!.answer(noContent()));
    await waitForPickerDOM(() => !!pending, "delete re-read");
    elsewhere.focus();
    await act(async () => pending!.answer(json(200, tab.issue)));
    await waitForPickerDOM(() => !!document.querySelector('input[type="file"]'), "remounted after delete");
    expect(document.activeElement).toBe(elsewhere);
    expect(document.body.textContent).toContain("content_files.confirmed_delete");
    await tab.select([new File(["next"], "next.txt", { type: "text/plain" })]);
    await tab.click("content_files.upload");
    await act(async () => until(() => tab.puts.length === 2, "second upload"));
    expect(document.body.textContent).not.toContain("content_files.confirmed_delete");
  });
  it.each([401, 403, 404])(
    `${kind}: canonical %i clears retained input and receipts, and a later fresh owner never revives either`,
    async (status) => {
      await render(kind);
      await openFiles();
      await title("must be cleared");
      core = (call) => call.answer(problem(status, status === 404 ? "not_found" : "forbidden"));
      await complete();
      await waitForPickerDOM(
        () =>
          status === 404
            ? tab.app.issue.verification.isMissing(tab.issue.id)
            : !!tab.app.issue.verification.getError(tab.issue.id),
        "refused owner"
      );
      expect(document.body.textContent).not.toContain("content_files.confirmed_upload");
      core = (call) => call.answer(json(200, tab.issue));
      await act(async () => {
        await tab.app.issue.issues.refreshAuthoritative(tab.issue.id);
      });
      await waitForPickerDOM(() => !!document.querySelector('textarea[aria-label="issue.title.label"]'), "new owner");
      expect(document.querySelector<HTMLTextAreaElement>('textarea[aria-label="issue.title.label"]')!.value).toBe(
        tab.issue.name
      );
      expect(tab.nerve.calls.filter((c) => c.method === "PATCH")).toHaveLength(0);
    }
  );
}
it("the attachment widget keeps the authoritative count of 51 while reading only the first 50 rows", async () => {
  page = {
    data: Array.from({ length: 50 }, (_, i) => attachmentOf(i === 0 ? "file-a" : `file-${i}`)),
    next_cursor: "next",
  };
  core = (call) => call.answer(json(200, { ...tab.issue, attachment_count: 51 }));
  await tab.app.issue.issues.refreshAuthoritative(tab.issue.id);
  await render("detail");
  await openFiles();
  const label = [...document.querySelectorAll("button")].find((b) => b.textContent?.includes("content_files.title"))!;
  expect(label.textContent).toContain("51");
  expect(tab.nerve.to(path)).toHaveLength(1);
  expect(document.querySelectorAll('button[aria-label^="content_files.download"]')).toHaveLength(50);
  expect(tab.button("content_files.more")).toBeDefined();
});
it("a busy attachment confirmation prevents the underlying peek from closing on Escape or backdrop", async () => {
  await render("peek");
  await openFiles();
  await tab.click("content_files.remove file-a.txt");
  await waitForPickerDOM(() => !!document.querySelector('[role="dialog"]'), "dialog");
  await tab.click("content_files.confirm_delete");
  await act(async () => until(() => tab.nerve.to("/api/v0/assets/file-a").length === 1, "delete"));
  await act(async () =>
    document.activeElement!.dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
    )
  );
  const backdrop = document.querySelector<HTMLElement>('[data-headlessui-state][class*="bg-backdrop"]')!;
  await act(async () => {
    backdrop.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
    backdrop.click();
  });
  expect(tab.app.issue.issueDetail.peekIssue?.issueId).toBe(tab.issue.id);
  expect(document.querySelector('[role="dialog"]')).not.toBeNull();
  await act(async () => tab.nerve.to("/api/v0/assets/file-a")[0]!.answer(noContent()));
  await waitForPickerDOM(() => !document.querySelector('[role="dialog"]'), "confirmed");
  expect(tab.nerve.to("/api/v0/assets/file-a")).toHaveLength(1);
});
it("closing a peek during transfer prevents a late complete from publishing into a reopened owner", async () => {
  await render("peek");
  await openFiles();
  await tab.select([new File(["body"], "note.txt", { type: "text/plain" })]);
  await tab.click("content_files.upload");
  await act(async () => until(() => tab.puts.length === 1, "raw upload"));
  await tab.click("close peek");
  await render("peek");
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 200 })));
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(tab.nerve.calls.filter((c) => c.path.endsWith("/complete"))).toHaveLength(0);
  expect(document.body.textContent).not.toContain("content_files.confirmed_upload");
});
it.each(["guest", "archived"] as const)("%s can read attachments but receives no write controls", async (kind) => {
  if (kind === "guest") {
    const responder = tab.nerve.respond;
    tab.nerve.respond = undefined;
    await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 5 }]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
    tab.nerve.respond = responder;
  }
  core = (call) =>
    call.answer(json(200, { ...tab.issue, archived_at: kind === "archived" ? "2026-11-01T00:00:00Z" : null }));
  await tab.app.issue.issues.refreshAuthoritative(tab.issue.id);
  await act(async () =>
    tab.renderer.render(
      <StoreContext.Provider value={tab.app}>
        <IssueDetailRoot workspaceSlug="ws" projectId="A" issueId={tab.issue.id} />
      </StoreContext.Provider>
    )
  );
  await openFiles();
  expect(document.querySelector('input[type="file"]')).toBeNull();
  expect(document.querySelector('button[aria-label^="content_files.remove"]')).toBeNull();
  expect(tab.button("content_files.download file-a.txt")).toBeDefined();
  expect(document.querySelector('textarea[aria-label="issue.title.label"]')).toBeNull();
});

it("StrictMode preserves local input through verification, and a changed base requires an explicit decision", async () => {
  await render("detail", true);
  await openFiles();
  await title("my pending edit");
  core = (call) => call.answer(json(200, { ...tab.issue, name: "concurrent title" }));
  await complete();
  await waitForPickerDOM(
    () => document.body.textContent?.includes("content_files.input.conflict") ?? false,
    "conflict"
  );
  expect(document.querySelector<HTMLTextAreaElement>('textarea[aria-label="issue.title.label"]')!.value).toBe(
    "my pending edit"
  );
  await act(async () => vi.advanceTimersByTimeAsync(3000));
  expect(tab.nerve.calls.filter((c) => c.method === "PATCH")).toHaveLength(0);
  await tab.click("content_files.input.keep");
  await waitForPickerDOM(() => tab.nerve.calls.some((c) => c.method === "PATCH"), "explicit save");
  expect(tab.nerve.calls.filter((c) => c.method === "PATCH")).toHaveLength(1);
});
it("a known title 422 survives attachment verification and is not resent by remounting", async () => {
  await render("detail");
  await openFiles();
  await title("rejected edit");
  await act(async () => vi.advanceTimersByTimeAsync(1500));
  const patch = tab.nerve.calls.find((c) => c.method === "PATCH");
  expect(patch).toBeDefined();
  await act(async () => patch!.answer(problem(422, "validation_error")));
  await waitForPickerDOM(() => !!document.querySelector('[role="alert"]'), "title refusal");
  core = (call) => call.answer(json(200, { ...tab.issue, name: "changed elsewhere" }));
  await complete();
  await waitForPickerDOM(
    () => document.body.textContent?.includes("content_files.input.conflict") ?? false,
    "preserved refusal"
  );
  expect(document.querySelector<HTMLTextAreaElement>('textarea[aria-label="issue.title.label"]')!.value).toBe(
    "rejected edit"
  );
  expect(document.querySelector('[role="alert"]')).not.toBeNull();
  expect([...document.querySelectorAll("button")].some((button) => button.textContent === "common.retry")).toBe(false);
  await act(async () => vi.advanceTimersByTimeAsync(3000));
  expect(tab.nerve.calls.filter((c) => c.method === "PATCH")).toHaveLength(1);
});
it("a newly archived owner displays its canonical title while keeping local input visibly read-only", async () => {
  await render("detail");
  await openFiles();
  await title("not saved");
  core = (call) => call.answer(json(200, { ...tab.issue, archived_at: "2026-11-01T00:00:00Z" }));
  await complete();
  await waitForPickerDOM(() => !!tab.app.issue.issues.getIssueDetailById(tab.issue.id)?.archived_at, "archived owner");
  expect(document.querySelector('textarea[aria-label="issue.title.label"]')).toBeNull();
  expect(document.body.textContent).toContain("content_files.input.read_only");
  expect(document.body.textContent).toContain(tab.issue.name);
  expect(document.body.textContent).not.toContain("not saved");
  expect(tab.nerve.calls.filter((c) => c.method === "PATCH")).toHaveLength(0);
});

it("cancel releases only its attachment dialog and Escape can then close the peek", async () => {
  await render("peek");
  await openFiles();
  await tab.click("content_files.remove file-a.txt");
  await waitForPickerDOM(() => !!document.querySelector('[role="dialog"]'), "dialog");
  await tab.click("cancel");
  await waitForPickerDOM(() => !document.querySelector('[role="dialog"]'), "cancelled");
  await act(async () =>
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }))
  );
  expect(tab.app.issue.issueDetail.peekIssue).toBeUndefined();
  expect(tab.nerve.calls.filter((c) => c.method === "DELETE")).toHaveLength(0);
});

it("content dialog leases release by identity and never erase a second open dialog", () => {
  const modals = new IssueDetailModals(() => undefined);
  modals.activate();
  const first = modals.retainContentDialog();
  const second = modals.retainContentDialog();
  first();
  expect(modals.isAnyModalOpen).toBe(true);
  second();
  expect(modals.isAnyModalOpen).toBe(false);
});
