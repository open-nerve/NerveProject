// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useRef } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { EditorRefApi } from "@nerve/editor";
import type { RootStore } from "@/store/root.store";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { IssueMainContent } from "./issue-detail/main-content";
import { PeekOverviewIssueDetails } from "./peek-overview/issue-detail";
import { useIssueOperations } from "./issue-detail/use-issue-operations";
import { IssueDetailHost } from "./detail-host";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "parents" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "parents" }) }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: undefined,
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/use-window-size", () => ({ default: () => [1024, 800] }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
// Keep the real content owners, breadcrumb, sibling menu, stores and native request path; unrelated editors are leaves.
vi.mock("@/components/editor/rich-text/description-input", () => ({ DescriptionInput: () => null }));
vi.mock("./title-input", () => ({ IssueTitleInput: () => null }));
vi.mock("./issue-detail/description-versions", () => ({ IssueDescriptionVersions: () => null }));
vi.mock("./issue-detail/issue-activity", () => ({ IssueActivity: () => null }));
vi.mock("./issue-detail/reactions", () => ({ IssueReaction: () => null }));
vi.mock("./issue-detail-widgets", () => ({ IssueDetailWidgets: () => null }));
vi.mock("./peek-overview/properties", () => ({ PeekOverviewProperties: () => null }));

type Kind = "detail" | "peek";
type Mode = "editable" | "archived" | "guest";
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
const noop = () => {};
const cache = () => new Map();
function Content({ kind, mode }: { kind: Kind; mode: Mode }) {
  const editorRef = useRef<EditorRefApi>(null);
  const issueOperations = useIssueOperations("ws", "A", "child");
  const context = {
    workspaceSlug: "ws",
    projectId: "A",
    issueId: "child",
    issueOperations,
    isArchived: mode === "archived",
  };
  return kind === "detail" ? (
    <IssueMainContent {...context} isEditable={mode !== "guest"} />
  ) : (
    <PeekOverviewIssueDetails
      {...context}
      editorRef={editorRef}
      disabled={mode === "guest"}
      isSubmitting="saved"
      setTitleStatus={noop}
      setDescriptionStatus={noop}
    />
  );
}
beforeEach(async () => {
  tab = await issueSelectionTab();
});
afterEach(async () => tab.close());

async function showParent(kind: Kind, mode: Mode = "editable", holdParent = false) {
  if (mode === "guest")
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
      { ...tab.project, member_role: 5, guest_view_all_features: true },
    ]);
  const child = {
    ...issueDetailOf("child"),
    parent_id: "parent",
    archived_at: mode === "archived" ? "2026-10-31T00:00:00Z" : null,
  };
  const parent = { ...issueDetailOf("parent"), sequence_id: 2 };
  const sibling = { ...issueDetailOf("sibling"), parent_id: parent.id, sequence_id: 3 };
  for (const issue of [parent, child, sibling]) tab.app.issue.issues.acceptRead(issue, tab.app.issue.captureRead());
  const native = tab.nerve.respond;
  let detached = false;
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && call.path === "/api/v0/issues/parent/sub-issues")
      call.answer(json(200, { data: detached ? [sibling] : [child, sibling] }));
    else if (call.method === "GET" && call.path === "/api/v0/issues/parent") {
      if (!holdParent) call.answer(json(200, parent));
    } else if (call.method === "PATCH" && call.path === "/api/v0/issues/child") {
      detached = true;
      call.answer(json(200, { ...child, parent_id: null }));
    } else native?.(call);
  };
  await tab.render(
    <MemoryRouter>
      <SWRConfig value={{ provider: cache }}>
        <IssueDetailHost workspaceSlug="ws" projectId="A" issueId="child">
          <Content kind={kind} mode={mode} />
        </IssueDetailHost>
      </SWRConfig>
    </MemoryRouter>
  );
  return { parent, child, sibling };
}

for (const kind of ["detail", "peek"] as const) {
  it.each(["editable", "archived", "guest"] as const)(
    `R07 ${kind} parent menu preserves sibling navigation and offers removal only for editable content (%s)`,
    async (mode) => {
      await showParent(kind, mode);
      const trigger = document.querySelector<HTMLButtonElement>('button[aria-haspopup="menu"]');
      expect(trigger).not.toBeNull();
      await act(async () => trigger?.click());
      await act(async () =>
        until(
          () => tab.app.issue.issueDetail.subIssues.subIssuesByIssueId("parent")?.includes("sibling") === true,
          "sibling read"
        )
      );
      expect(document.querySelector('button[aria-haspopup="menu"]')).toBe(trigger);
      expect(trigger?.getAttribute("aria-expanded")).toBe("true");
      expect(document.querySelector('[role="menu"]')?.textContent).toContain("A-3");
      expect(document.querySelector("a")?.getAttribute("href")).toBe("/ws/browse/A-2");
      const open = vi.spyOn(window, "open").mockReturnValue(null);
      const sibling = [...document.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')].find((button) =>
        button.textContent?.includes("A-3")
      )!;
      await act(async () => sibling.click());
      expect(open).toHaveBeenCalledExactlyOnceWith("/ws/browse/A-3", "_blank", "noopener,noreferrer");
      await act(async () => trigger?.click());
      const remove = [...document.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')].find((button) =>
        button.textContent?.includes("issue.remove.parent.label")
      );
      if (mode === "editable") {
        expect(remove).toBeDefined();
        await act(async () => remove?.click());
        await act(async () =>
          until(() => tab.app.issue.issues.getIssueById("child")?.parent_id === null, "confirmed removal")
        );
        expect(tab.nerve.calls.filter((call) => call.method === "PATCH").map((call) => [call.path, call.body])).toEqual(
          [["/api/v0/issues/child", { parent_id: null }]]
        );
      } else {
        expect(remove).toBeUndefined();
        expect(tab.nerve.calls.filter((call) => call.method !== "GET")).toEqual([]);
        expect(tab.app.issue.issues.getIssueById("child")?.parent_id).toBe("parent");
      }
    }
  );
  it.each([200, 404, 503])(`R08 ${kind} cold sibling menu survives its parent verification (%i)`, async (status) => {
    const { parent } = await showParent(kind, "editable", true);
    const trigger = document.querySelector<HTMLButtonElement>('button[aria-haspopup="menu"]')!;
    expect(document.querySelector("a")?.textContent).toContain(parent.name);
    await act(async () => trigger.click());
    await act(async () => until(() => tab.nerve.to("/api/v0/issues/parent").length === 1, "parent verification"));
    expect(tab.app.issue.issues.getIssueById("parent")).toBeUndefined();
    // The still-visible child's relationship owns the shell, but uncertain parent data must not be displayed.
    expect(document.querySelector('button[aria-haspopup="menu"]')).toBe(trigger);
    expect(trigger.getAttribute("aria-expanded")).toBe("true");
    expect(document.querySelector('[role="menu"]')?.textContent).toContain("Loading");
    expect(document.querySelector("a")).toBeNull();
    const reply = status === 200 ? json(200, parent) : problem(status, status === 404 ? "not_found" : "server_busy");
    await act(async () => tab.nerve.to("/api/v0/issues/parent")[0]!.answer(reply));
    await act(async () =>
      until(() => !tab.app.issue.issueDetail.subIssues.getLoading("parent"), "sibling read settled")
    );
    expect(document.querySelector('button[aria-haspopup="menu"]')).toBe(trigger);
    expect(trigger.getAttribute("aria-expanded")).toBe("true");
    if (status === 200) {
      expect(document.querySelector('[role="menu"]')?.textContent).toContain("A-3");
      expect(document.querySelector("a")?.textContent).toContain(parent.name);
    } else {
      expect(document.querySelector("a")).toBeNull();
      expect(document.querySelector('[role="menu"]')?.textContent).not.toContain("A-3");
      expect(document.querySelector('[role="menu"]')?.textContent).not.toContain("issue.remove.parent.label");
      if (status === 503) {
        const retry = [...document.querySelectorAll<HTMLButtonElement>("button")].find(
          (button) => button.textContent === "common.retry"
        );
        expect(retry).toBeDefined();
        await act(async () => retry!.click());
        await act(async () =>
          until(() => tab.nerve.to("/api/v0/issues/parent").length === 2, "parent verification retry")
        );
        await act(async () => tab.nerve.to("/api/v0/issues/parent")[1]!.answer(json(200, parent)));
        await act(async () => until(() => !tab.app.issue.issueDetail.subIssues.getLoading("parent"), "retry settled"));
        expect(document.querySelector('[role="menu"]')?.textContent).toContain("A-3");
        expect(trigger.getAttribute("aria-expanded")).toBe("true");
      }
    }
    // A real project permission loss still removes the whole host and all navigation/action data.
    await act(async () => loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []));
    expect(document.querySelector('[role="menu"]')).toBeNull();
    expect(document.querySelector('button[aria-haspopup="menu"]')).toBeNull();
    expect(tab.nerve.calls.filter((call) => call.method !== "GET")).toEqual([]);
  });
  it.each(["parent", "permission"])(
    `R08 ${kind} discards the old menu when %s changes during its read`,
    async (change) => {
      const { parent, child } = await showParent(kind, "editable", true);
      const trigger = document.querySelector<HTMLButtonElement>('button[aria-haspopup="menu"]')!;
      await act(async () => trigger.click());
      await act(async () => until(() => tab.nerve.to("/api/v0/issues/parent").length === 1, "held parent read"));
      expect(trigger.getAttribute("aria-expanded")).toBe("true");
      await act(async () => {
        if (change === "permission") await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
        else {
          tab.app.issue.issues.acceptRead({ ...issueDetailOf("next"), sequence_id: 4 }, tab.app.issue.captureRead());
          tab.app.issue.issues.acceptRead(
            { ...child, parent_id: "next", updated_at: "2026-11-01T00:00:00Z" },
            tab.app.issue.captureRead()
          );
        }
      });
      expect(document.querySelector('[role="menu"]')).toBeNull();
      expect(trigger.isConnected).toBe(false);
      await act(async () => tab.nerve.to("/api/v0/issues/parent")[0]!.answer(json(200, parent)));
      await act(async () =>
        until(() => !tab.app.issue.issueDetail.subIssues.getLoading("parent"), "retired menu read settled")
      );
      expect(document.querySelector('[role="menu"]')).toBeNull();
      expect(document.body.textContent).not.toContain("A-3");
      if (change === "parent") {
        expect(document.querySelector("a")?.getAttribute("href")).toBe("/ws/browse/A-4");
        expect(document.querySelector('button[aria-haspopup="menu"]')?.getAttribute("aria-expanded")).toBe("false");
      } else expect(document.querySelector('button[aria-haspopup="menu"]')).toBeNull();
      expect(tab.nerve.calls.filter((call) => call.method !== "GET")).toEqual([]);
    }
  );
}
