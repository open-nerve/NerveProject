/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueLink } from "@nerve/api-client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until, track } from "@/lib/auth/fake-time";
import { heldChange } from "@/lib/auth/fake-tab";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { loadProjects } from "@/store/project/fake-projects";
import { issueDetailOf, issueLinkOf, issueStoreTab, requestedIssue } from "../fake-issues";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const link = issueLinkOf("link-93");
const source = { ...issueDetailOf("x"), link_count: 7 };
const store = () => tab.app.issue.issueDetail.link;
const read = () => store().fetchLinks("ws", "A", "x");
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  tab.app.issue.issues.acceptRead(source, tab.app.issue.captureRead());
});
afterEach(() => {
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
async function loaded() {
  const sent = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/x/links"]);
  sent.call.answer(json(200, { data: [link] }));
  await sent.promise;
}
it("reads a native collection with nullable title and metadata without manufacturing fields", async () => {
  await loaded();
  expect(store().getLinksByIssueId("x")).toEqual([link.id]);
  expect(store().getLinkById("x", link.id)).toEqual(link);
  expect(store().getLinkById("other", link.id)).toBeUndefined();
});
it("an ordinary read refusal preserves the complete collection and exposes the error", async () => {
  await loaded();
  const next = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/x/links"]);
  next.call.answer(problem(500, "unavailable"));
  await expect(next.promise).rejects.toMatchObject({ status: 500 });
  expect(store().getLinksByIssueId("x")).toEqual([link.id]);
  expect(store().getError("x")).toMatchObject({ status: 500 });
});
const kinds = ["create", "update", "delete"] as const;
function operation(kind: (typeof kinds)[number]): Promise<IssueLink | void> {
  if (kind === "create") return store().createLink("ws", "A", "x", { url: link.url, title: "New" });
  if (kind === "update") return store().updateLink("ws", "A", "x", link.id, { title: "Edited" });
  return store().removeLink("ws", "A", "x", link.id);
}
const endpoint = (kind: (typeof kinds)[number]): ["POST" | "PATCH" | "DELETE", string] =>
  kind === "create"
    ? ["POST", "/api/v0/issues/x/links"]
    : [kind === "update" ? "PATCH" : "DELETE", `/api/v0/issue-links/${link.id}`];
it.each(kinds)("%s waits for confirmation and a refusal leaves links and counts unchanged", async (kind) => {
  await loaded();
  const at = tab.nerve.calls.length;
  const sent = await requestedIssue(tab.nerve, () => operation(kind), endpoint(kind));
  expect(store().getLinkById("x", link.id)).toEqual(link);
  expect(tab.app.issue.issues.getIssueById("x")?.link_count).toBe(7);
  sent.call.answer(problem(403, "forbidden"));
  await expect(sent.promise).rejects.toMatchObject({ status: 403 });
  expect(store().getLinkById("x", link.id)).toEqual(link);
  expect(tab.app.issue.issues.getIssueById("x")?.link_count).toBe(7);
  expect(tab.nerve.calls).toHaveLength(at + 1);
});
it.each(kinds)("%s confirms the native answer, verifies the count and survives an older GET", async (kind) => {
  await loaded();
  const old = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/x/links"]);
  const sent = await requestedIssue(tab.nerve, () => operation(kind), endpoint(kind));
  const answer = {
    ...link,
    id: kind === "create" ? "created" : link.id,
    title: kind === "update" ? "Edited" : "New",
    metadata: { error: "failed" as const },
  };
  expect(sent.call.body).toEqual(
    kind === "create" ? { url: link.url, title: "New" } : kind === "update" ? { title: "Edited" } : undefined
  );
  sent.call.answer(kind === "delete" ? noContent() : json(200, answer));
  await until(() => tab.nerve.to("/api/v0/issues/x").length > 0 || sent.result.settled, "authoritative link count");
  expect(tab.nerve.to("/api/v0/issues/x")).toHaveLength(1);
  tab.nerve
    .to("/api/v0/issues/x")[0]!
    .answer(json(200, { ...source, link_count: 9, updated_at: "2026-10-30T00:00:00Z" }));
  await sent.promise;
  old.call.answer(json(200, { data: [link] }));
  await old.promise;
  expect(tab.app.issue.issues.getIssueById("x")?.link_count).toBe(9);
  expect(store().getLinksByIssueId("x")).toEqual(
    kind === "delete" ? [] : kind === "create" ? [link.id, "created"] : [link.id]
  );
  if (kind !== "delete") expect(store().getLinkById("x", answer.id)).toEqual(answer);
});
it.each(["blocked", "failed"])(
  "metadata %s is a successful link, and post-write refusal retries only reads",
  async (error) => {
    const sent = await requestedIssue(tab.nerve, () => operation("create"), endpoint("create"));
    const answer = { ...link, metadata: { error } };
    sent.call.answer(json(201, answer));
    await until(() => tab.nerve.to("/api/v0/issues/x").length > 0 || sent.result.settled, "post-write read");
    expect(tab.nerve.to("/api/v0/issues/x")).toHaveLength(1);
    tab.nerve.to("/api/v0/issues/x")[0]!.answer(problem(500, "unavailable"));
    expect(await sent.promise).toEqual(answer);
    expect(store().getLinkById("x", link.id)).toEqual(answer);
    expect(store().getError("x")).toMatchObject({ status: 500 });
    const retry = store().retry("ws", "A", "x");
    await until(() => tab.nerve.to("/api/v0/issues/x").length === 2, "read retry");
    tab.nerve
      .to("/api/v0/issues/x")[1]!
      .answer(json(200, { ...source, link_count: 11, updated_at: "2026-10-30T00:00:00Z" }));
    await retry;
    expect(store().getError("x")).toBeUndefined();
    expect(tab.nerve.calls.filter((call) => call.method === "POST")).toHaveLength(1);
  }
);
it("revokes held and delayed links across permission ABA", async () => {
  await loaded();
  const old = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/x/links"]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  expect(store().getLinksByIssueId("x")).toBeUndefined();
  expect(store().getLinkById("x", link.id)).toBeUndefined();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  old.call.answer(json(200, { data: [link] }));
  await expect(old.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(store().getLinksByIssueId("x")).toBeUndefined();
});
it("rejects a link belonging to another issue without publishing it", async () => {
  const sent = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/x/links"]);
  sent.call.answer(json(200, { data: [{ ...link, issue_id: "wrong" }] }));
  await expect(sent.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(store().getLinksByIssueId("x")).toBeUndefined();
});
it("shares the core issue queue and rechecks scope after waiting", async () => {
  const root = tab.app.issue;
  const gate = heldChange<void>();
  const holding = root.enqueueIssueChanges(["x"], () => gate.sent);
  const writing = track(operation("create"));
  await vi.advanceTimersByTimeAsync(10);
  expect(tab.nerve.calls).toHaveLength(0);
  root.retire();
  gate.answer();
  await holding;
  await until(() => writing.settled, "retired queued link write");
  expect(writing.error).toBeInstanceOf(SessionChangedError);
  expect(tab.nerve.calls).toHaveLength(0);
});
