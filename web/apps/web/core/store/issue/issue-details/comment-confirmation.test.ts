/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { contentAssetTab } from "@/store/asset/fake-assets";
import { json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { loadProjects } from "@/store/project/fake-projects";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
let tab: Awaited<ReturnType<typeof contentAssetTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await contentAssetTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
it.each(["create", "update"] as const)(
  "%s publishes its confirmed comment before canonical verification can unmount its form",
  async (kind) => {
    const store = tab.app.issue.issueDetail.comment;
    const confirmed = vi.fn(() => expect(tab.app.issue.issues.getIssueDetailById(tab.issue.id)).toBeDefined());
    const result = track(
      kind === "create"
        ? store.createComment("ws", "A", tab.issue.id, { comment_html: "<p>saved</p>" }, confirmed)
        : store.updateComment("ws", "A", tab.issue.id, tab.comment.id, { comment_html: "<p>saved</p>" }, confirmed)
    );
    const path =
      kind === "create" ? `/api/v0/issues/${tab.issue.id}/comments` : `/api/v0/issue-comments/${tab.comment.id}`;
    await until(() => tab.nerve.to(path).length === 1, "comment write");
    const answer = { ...tab.comment, comment_html: "<p>saved</p>" };
    tab.nerve.to(path)[0]!.answer(json(kind === "create" ? 201 : 200, answer));
    await until(() => tab.nerve.to(`/api/v0/issues/${tab.issue.id}`).length === 1, "owner verification");
    expect(confirmed).toHaveBeenCalledExactlyOnceWith(answer);
    expect(tab.app.issue.issues.getIssueDetailById(tab.issue.id)).toBeUndefined();
    tab.nerve.to(`/api/v0/issues/${tab.issue.id}`)[0]!.answer(problem(503, "server_busy"));
    await until(() => result.settled, "confirmed result after failed read");
    expect(result.error).toBeUndefined();
    expect(result.value).toEqual(answer);
  }
);
it("permission ABA during a write suppresses its old confirmation callback", async () => {
  const confirmed = vi.fn();
  const result = track(
    tab.app.issue.issueDetail.comment.createComment("ws", "A", tab.issue.id, { comment_html: "<p>old</p>" }, confirmed)
  );
  const path = `/api/v0/issues/${tab.issue.id}/comments`;
  await until(() => tab.nerve.to(path).length === 1, "old write");
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  tab.nerve.to(path)[0]!.answer(json(201, tab.comment));
  await until(() => result.settled, "old result");
  expect(confirmed).not.toHaveBeenCalled();
  expect(result.error).toBeDefined();
});
