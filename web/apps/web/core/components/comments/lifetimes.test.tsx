// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { loadProjects } from "@/store/project/fake-projects";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { attachmentOf } from "@/store/asset/fake-assets";
import { commentFilesTab } from "./fake-comment-files";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "host" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/components/editor/lite-text", async () => ({
  LiteTextEditor: (await import("./fake-comment-editor")).commentEditor(),
}));
vi.mock("@/components/assets/profile-image", () => ({ ProfileAvatar: () => null }));

let tab: Awaited<ReturnType<typeof commentFilesTab>>;
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
  tab = await commentFilesTab();
});
afterEach(async () => {
  await tab?.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
async function openFiles(card: HTMLElement) {
  const trigger = [...card.querySelectorAll<HTMLButtonElement>("button")].find(
    (button) => button.textContent === "content_files.title"
  )!;
  await act(async () => trigger.click());
}
async function issueUpload() {
  const trigger = [...document.querySelectorAll<HTMLButtonElement>("button")].find(
    (button) => button.textContent === `content_files.title${tab.issue.attachment_count}`
  )!;
  await act(async () => trigger.click());
  await waitForPickerDOM(() => document.querySelectorAll('input[type="file"]').length === 2, "issue picker");
  const picker = document.querySelectorAll('input[type="file"]')[1]!.parentElement!;
  await tab.select([new File(["body"], "issue.txt", { type: "text/plain" })], picker);
  await tab.click("content_files.upload", picker);
  await act(async () => until(() => tab.puts.length === 1, "issue upload"));
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 200 })));
}
it("a failed comment edit and an independent new comment survive another attachment's complete and failed owner read", async () => {
  await tab.render();
  await tab.fill("<p>unsent new comment</p>");
  const card = document.getElementById(`comment-${tab.comment.id}`)!;
  await act(async () => card.querySelector<HTMLButtonElement>('button[aria-haspopup="menu"]')!.click());
  await waitForPickerDOM(() => !!document.querySelector('[role="menuitem"]'), "comment menu");
  await act(async () =>
    [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')]
      .find((item) => item.textContent === "common.actions.edit")!
      .click()
  );
  await waitForPickerDOM(() => !!card.querySelector<HTMLTextAreaElement>("textarea:not([readonly])"), "edit form");
  const edited = card.querySelector<HTMLTextAreaElement>("textarea")!;
  await tab.fill("<p>keep failed edit</p>", edited);
  await tab.enter(edited);
  const path = `/api/v0/issue-comments/${tab.comment.id}`;
  await act(async () => until(() => tab.nerve.to(path).length === 1, "edit"));
  await act(async () =>
    tab.nerve
      .to(path)[0]!
      .answer(
        json(422, { status: 422, code: "validation_failed", errors: [{ field: "comment_html", code: "required" }] })
      )
  );
  tab.control.core = (call) => call.answer(problem(503, "server_busy"));
  await issueUpload();
  await waitForPickerDOM(() => !tab.text(), "body withdrawn");
  tab.control.core = (call) => call.answer(json(200, tab.issue));
  await tab.click("read owner");
  await waitForPickerDOM(() => !!tab.text(), "restored body");
  const fresh = document.getElementById(`comment-${tab.comment.id}`)!.querySelector<HTMLTextAreaElement>("textarea")!;
  expect(fresh.value).toBe("<p>keep failed edit</p>");
  expect(fresh.readOnly).toBe(false);
  expect(tab.text()!.value).toBe("<p>unsent new comment</p>");
  expect(fresh.getAttribute("aria-invalid")).toBe("true");
  expect(document.getElementById(fresh.getAttribute("aria-describedby")!)?.textContent).toBe("errors.field.required");
  expect(tab.nerve.to(path)).toHaveLength(1);
  expect(tab.posts()).toHaveLength(0);
});
it.each([20, 15] as const)(
  "a non-author with effective role %i can read files, only admin can delete, and neither can add",
  async (role) => {
    const responder = tab.nerve.respond;
    tab.nerve.respond = undefined;
    await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 15 }]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: role }]);
    tab.nerve.respond = responder;
    tab.control.comments = () => [{ ...tab.comment, actor_id: "another-author" }];
    await tab.app.issue.issues.refreshAuthoritative(tab.issue.id);
    await tab.app.issue.issueDetail.comment.fetchComments("ws", "A", tab.issue.id);
    const path = `/api/v0/issue-comments/${tab.comment.id}/attachments`;
    tab.nerve.respond = (call) =>
      call.path === path
        ? call.answer(json(200, { data: [attachmentOf("other-file")], next_cursor: null }))
        : responder?.(call);
    await tab.render();
    const card = document.getElementById(`comment-${tab.comment.id}`)!;
    await openFiles(card);
    await waitForPickerDOM(() => card.textContent?.includes("other-file.txt") ?? false, "files read");
    expect(card.querySelector('input[type="file"]')).toBeNull();
    expect(!!card.querySelector('button[aria-label="content_files.remove other-file.txt"]')).toBe(role === 20);
    expect(tab.nerve.calls.some((call) => call.path === "/api/v0/assets")).toBe(false);
  }
);
it("a deleted canonical comment cannot publish its late attachment page", async () => {
  const path = `/api/v0/issue-comments/${tab.comment.id}/attachments`;
  const respond = tab.nerve.respond;
  tab.nerve.respond = (call) => {
    if (call.path !== path) respond?.(call);
  };
  await tab.render();
  const card = document.getElementById(`comment-${tab.comment.id}`)!;
  await openFiles(card);
  await act(async () => until(() => tab.nerve.to(path).length === 1, "page"));
  tab.control.comments = () => [];
  await act(async () => {
    await tab.app.issue.issueDetail.comment.fetchComments("ws", "A", tab.issue.id);
  });
  await act(async () => tab.nerve.to(path)[0]!.answer(json(200, { data: [attachmentOf("late")], next_cursor: null })));
  expect(document.getElementById(`comment-${tab.comment.id}`)).toBeNull();
  expect(document.body.textContent).not.toContain("late.txt");
  expect(tab.nerve.to(path)[0]!.aborted()).toBe(true);
});

it("two mounted comment forms keep distinct editor IDs, text and selected ordinary files", async () => {
  await tab.render(2);
  const hosts = [...document.querySelectorAll<HTMLElement>("[data-comment-host]")];
  const first = hosts[0]!.querySelector<HTMLTextAreaElement>('textarea[id^="add_comment_"]')!;
  const second = hosts[1]!.querySelector<HTMLTextAreaElement>('textarea[id^="add_comment_"]')!;
  expect(first.id).not.toBe(second.id);
  await tab.fill("<p>first</p>", first);
  await tab.fill("<p>second</p>", second);
  await tab.upload();
  await tab.enter(second);
  await act(async () => until(() => tab.posts().length === 1, "second form POST"));
  expect(tab.posts()[0]!.body).toEqual({ comment_html: "<p>second</p>", attachment_ids: [] });
  expect(first.value).toBe("<p>first</p>");
  expect(hosts[0]!.textContent).toContain("comment.txt");
  expect(hosts[1]!.textContent).not.toContain("comment.txt");
});
