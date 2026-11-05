// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { watchLegacyFileTraffic } from "@/store/asset/fake-assets";
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
let legacy: ReturnType<typeof watchLegacyFileTraffic>;
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
  legacy = watchLegacyFileTraffic(tab);
});
afterEach(async () => {
  const leaks = legacy.leaks();
  await tab?.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
  expect(leaks).toEqual([]);
});
const render = () => tab.render();
const fill = (value: string) => tab.fill(value);
const enter = () => tab.enter();
const upload = () => tab.upload();
const text = () => tab.text();
const posts = () => tab.posts();

it("uploads privately and sends HTML plus the selected ordinary ID once, without the legacy bind request", async () => {
  await render();
  await fill("<p>my comment</p>");
  await upload();
  expect(posts()).toHaveLength(0);
  await enter();
  await enter();
  await act(async () => until(() => posts().length === 1, "atomic comment"));
  expect(posts()[0]!.body).toEqual({ comment_html: "<p>my comment</p>", attachment_ids: ["asset-1"] });
  tab.control.comments = () => [{ ...tab.comment, id: "created", comment_html: "<p>my comment</p>" }];
  await act(async () => posts()[0]!.answer(json(201, tab.control.comments()[0])));
  await waitForPickerDOM(() => !!text() && text()!.value === "<p></p>", "fresh empty form");
  expect(document.activeElement).toBe(text());
  expect(document.body.textContent).not.toContain("comment.txt");
  expect(tab.nerve.calls.some((c) => c.path.includes("bulk") || c.path.includes("/api/assets"))).toBe(false);
  expect(posts()).toHaveLength(1);
});
it("a selected but incomplete ordinary file blocks comment submission and removing a ready selection is local", async () => {
  await render();
  await fill("<p>wait for file</p>");
  await tab.select([new File(["body"], "comment.txt", { type: "text/plain" })]);
  await enter();
  expect(posts()).toHaveLength(0);
  await tab.click("content_files.upload");
  await act(async () => until(() => tab.puts.length === 1, "PUT"));
  await enter();
  expect(posts()).toHaveLength(0);
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 200 })));
  await waitForPickerDOM(
    () => !!document.querySelector('button[aria-label="content_files.remove comment.txt"]'),
    "ready selection"
  );
  await tab.click("content_files.remove comment.txt");
  await enter();
  await act(async () => until(() => posts().length === 1, "comment without removed file"));
  expect(posts()[0]!.body).toEqual({ comment_html: "<p>wait for file</p>", attachment_ids: [] });
  expect(tab.nerve.calls.some((c) => c.method === "DELETE")).toBe(false);
});
it.each(["network", "proxy"] as const)(
  "an unknown %s result must be checked with GET and never auto-reposts",
  async (kind) => {
    await render();
    await fill("<p>possibly sent</p>");
    await upload();
    await enter();
    await act(async () => until(() => posts().length === 1, "POST"));
    await act(async () =>
      kind === "network" ? posts()[0]!.fail() : posts()[0]!.answer(new Response("Bad Gateway", { status: 502 }))
    );
    await waitForPickerDOM(
      () => document.body.textContent?.includes("content_files.comment.unknown") ?? false,
      "unknown result"
    );
    await enter();
    expect(posts()).toHaveLength(1);
    await tab.click("content_files.comment.verify");
    await waitForPickerDOM(
      () => document.body.textContent?.includes("content_files.comment.allow_resend") ?? false,
      "verified list"
    );
    expect(posts()).toHaveLength(1);
    expect(text()!.value).toBe("<p>possibly sent</p>");
    await tab.click("content_files.comment.allow_resend");
    expect(posts()).toHaveLength(1);
    await enter();
    await act(async () => until(() => posts().length === 2, "explicit resend"));
    expect(posts()[1]!.body).toEqual({ comment_html: "<p>possibly sent</p>", attachment_ids: ["asset-1"] });
  }
);
it("an unknown create the server committed is verified by its comment list, and the file it bound blocks a resend", async () => {
  await render();
  await fill("<p>committed</p>");
  await upload();
  await enter();
  await act(async () => until(() => posts().length === 1, "POST"));
  // The server created the comment and bound the selected file; only its answer is lost.
  tab.control.comments = () => [tab.comment, { ...tab.comment, id: "committed", comment_html: "<p>committed</p>" }];
  tab.assets.set("asset-1", { ...tab.assets.get("asset-1")!, scope: { kind: "comment", id: "committed" } });
  await act(async () => posts()[0]!.fail());
  await waitForPickerDOM(
    () => document.body.textContent?.includes("content_files.comment.unknown") ?? false,
    "unknown result"
  );
  await tab.click("content_files.comment.verify");
  await waitForPickerDOM(
    () => document.body.textContent?.includes("content_files.comment.allow_resend") ?? false,
    "verified by the comment list"
  );
  expect(document.body.textContent).toContain("content_files.comment.unverified");
  await tab.click("content_files.comment.allow_resend");
  await enter();
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(posts()).toHaveLength(1);
  expect(text()!.value).toBe("<p>committed</p>");
});
it("an unsent comment and selected IDs survive temporary owner failure, with fresh asset authorization and no POST", async () => {
  await render();
  await fill("<p>keep this draft</p>");
  await upload();
  tab.control.core = (call) => call.answer(problem(503, "server_busy"));
  await act(async () => {
    await tab.app.issue.issues.refreshAuthoritative(tab.issue.id, { afterWrite: true }).catch(() => undefined);
  });
  expect(text()).toBeNull();
  tab.control.core = (call) => call.answer(json(200, tab.issue));
  await tab.click("read owner");
  await waitForPickerDOM(() => !!text() && text()!.value === "<p>keep this draft</p>", "retained comment");
  await waitForPickerDOM(() => tab.nerve.to("/api/v0/assets/asset-1").length === 1, "fresh asset authorization");
  expect(posts()).toHaveLength(0);
  await enter();
  await act(async () => until(() => posts().length === 1, "explicit submit"));
  expect(posts()[0]!.body).toEqual({ comment_html: "<p>keep this draft</p>", attachment_ids: ["asset-1"] });
});

it("the author can upload an ordinary attachment on an existing comment without PATCH or issue count changes", async () => {
  await render();
  const card = document.getElementById(`comment-${tab.comment.id}`)!;
  const trigger = [...card.querySelectorAll<HTMLButtonElement>("button")].find(
    (b) => b.textContent === "content_files.title"
  );
  expect(trigger).toBeDefined();
  await act(async () => trigger!.click());
  await waitForPickerDOM(() => !!card.querySelector('input[type="file"]'), "comment file picker");
  await tab.select([new File(["body"], "existing.txt", { type: "text/plain" })], card);
  await tab.click("content_files.upload", card);
  await act(async () => until(() => tab.puts.length === 1, "comment PUT"));
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 200 })));
  const path = `/api/v0/issue-comments/${tab.comment.id}/attachments`;
  await waitForPickerDOM(() => tab.nerve.to(path).length === 2, "comment attachment reload");
  expect(tab.nerve.calls.some((c) => c.method === "PATCH")).toBe(false);
  expect(posts()).toHaveLength(0);
  expect(tab.nerve.to(`/api/v0/issues/${tab.issue.id}`)).toHaveLength(0);
  expect(tab.app.issue.issues.getIssueById(tab.issue.id)?.attachment_count).toBe(tab.issue.attachment_count);
});

it("a known body rejection keeps the selected IDs and exposes the field error until explicit resubmission", async () => {
  await render();
  await fill("<p>keep after rejection</p>");
  await upload();
  await enter();
  await act(async () => until(() => posts().length === 1, "POST"));
  await act(async () =>
    posts()[0]!.answer(
      json(422, { status: 422, code: "validation_failed", errors: [{ field: "comment_html", code: "required" }] })
    )
  );
  await waitForPickerDOM(() => text()?.getAttribute("aria-invalid") === "true", "field refusal");
  expect(text()!.value).toBe("<p>keep after rejection</p>");
  expect(document.body.textContent).toContain("comment.txt");
  expect(posts()).toHaveLength(1);
  await fill("<p>corrected</p>");
  await enter();
  await act(async () => until(() => posts().length === 2, "explicit correction"));
  expect(posts()[1]!.body).toEqual({ comment_html: "<p>corrected</p>", attachment_ids: ["asset-1"] });
});
