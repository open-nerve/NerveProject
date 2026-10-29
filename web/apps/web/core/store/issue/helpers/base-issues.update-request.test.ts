/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { answered, json } from "@/lib/auth/fake-nerve";
import { issueDetailOf, issueOf, issuePageOf, issueStoreTab, requestedIssue } from "../fake-issues";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  binding.app = tab.app;
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
async function initial(html?: string) {
  const { app, nerve } = tab;
  const issue = { ...issueOf("x"), ...(html === undefined ? {} : { description_html: html }) };
  await answered(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf([issue]),
    "list"
  );
  return issue;
}
it.each(["<p>before</p>", "", undefined])("pendingAndRefusalKeepExactHTMLPresence %s", async (html) => {
  const before = await initial(html);
  const { app, nerve } = tab;
  const request = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.updateIssue("ws", "A", "x", { description_html: "<p>new</p>", skip_activity: true }),
    ["PATCH", "/api/v0/issues/x"]
  );
  expect(request.call.body).toEqual({ description_html: "<p>new</p>", skip_activity: true });
  expect(app.issue.issues.getIssueById("x")).toEqual(before);
  expect(app.issue.projectIssues.total).toBe(1);
  const rejected = expect(request.promise).rejects.toBeDefined();
  request.call.answer(json(500, { code: "internal_error" }));
  await rejected;
  expect(app.issue.issues.getIssueById("x")).toEqual(before);
  expect(Object.hasOwn(app.issue.issues.getIssueById("x")!, "description_html")).toBe(html !== undefined);
});
it("confirmedServerHTMLAndVersionReplaceInputWithoutRequestContext", async () => {
  await initial("before");
  const { app, nerve } = tab;
  const answer = { ...issueDetailOf("x"), description_html: "", updated_at: "2026-10-29T00:00:00.000001Z" };
  await answered(
    nerve,
    () => app.issue.projectIssues.updateIssue("ws", "A", "x", { description_html: "client", skip_activity: true }),
    ["PATCH", "/api/v0/issues/x"],
    answer,
    "patch"
  );
  expect(app.issue.issues.getIssueDetailById("x")).toEqual(answer);
  expect(app.issue.issues.getIssueById("x")).not.toHaveProperty("skip_activity");
  expect(app.issue.projectIssues.total).toBe(1);
});
it.each(["removeIssue", "archiveIssue"] as const)("detail %s returnsRootRejectionToCaller", async (method) => {
  await initial();
  const { app, nerve } = tab;
  const request = await requestedIssue(
    nerve,
    () => app.issue.issueDetail[method]("ws", "A", "x"),
    method === "removeIssue" ? ["DELETE", "/api/v0/issues/x"] : ["POST", "/api/v0/issues/x/archive"]
  );
  const rejected = expect(request.promise).rejects.toBeDefined();
  request.call.answer(json(403, { code: "forbidden" }));
  await rejected;
  expect(app.issue.issues.getIssueById("x")).toBeDefined();
  expect(app.issue.projectIssues.total).toBe(1);
});
