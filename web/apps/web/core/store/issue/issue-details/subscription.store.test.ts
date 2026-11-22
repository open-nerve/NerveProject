/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueDetail } from "@nerve/api-client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/api-error";
import type { Call } from "@/lib/auth/fake-nerve";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { heldChange } from "@/lib/auth/fake-tab";
import { track, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { accountOf } from "@/store/member/fake-users";
import { seedAccount } from "@/store/user/fake-account";
import { issueDetailOf, issueStoreTab } from "../fake-issues";

// The caller's subscriptions to work items (P4 spec 2.3, R13): never optimistic. A change goes through the work item's
// queue, after the writes of it already out; whether nerve made it or refused it, the work item is read again in the
// same turn, and is_subscribed is that read's, never set here; a refusal then fails the change, a failure of the read
// after nerve made it is the work item's own (its read's retry). While a change is queued, out or being read again, it
// is pending. The stores and HTTP are real.

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const store = () => tab.app.issue.issueDetail.subscription;
const detail = (subscribed: boolean, changes: Partial<IssueDetail> = {}): IssueDetail => ({
  ...issueDetailOf("x"),
  is_subscribed: subscribed,
  ...changes,
});
const SUBSCRIBE = "/api/v0/issues/x/subscribe";
const UNSUBSCRIBE = "/api/v0/issues/x/unsubscribe";
const READ = "/api/v0/issues/x";
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  await seedAccount(tab.nerve, tab.app.user, accountOf("actor"));
  tab.app.issue.issues.acceptRead(detail(false), tab.app.issue.captureRead());
  tab.nerve.calls.length = 0;
});
afterEach(() => {
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

/** The request to path nerve receives once it does. */
async function sent(path: string): Promise<Call> {
  await until(() => tab.nerve.to(path).length > 0, path);
  return tab.nerve.to(path)[0]!;
}
const subscribed = () => tab.app.issue.issues.getIssueDetailById("x")?.is_subscribed;

it("subscribes in the work item's queue, after the change of it already out, then reads it again", async () => {
  const gate = heldChange<void>();
  const holding = tab.app.issue.writes.enqueueIssueChanges(["x"], () => gate.sent);
  const subscribing = track(store().subscribe("ws", "A", "x"));
  await vi.advanceTimersByTimeAsync(50);
  expect(tab.nerve.calls).toHaveLength(0);
  expect(store().isPending("x")).toBe(true);
  gate.answer();
  await holding;
  const post = await sent(SUBSCRIBE);
  expect([post.method, post.body]).toEqual(["POST", undefined]);
  post.answer(noContent());
  (await sent(READ)).answer(json(200, detail(true, { updated_at: "2026-10-30T00:00:00Z" })));
  await until(() => subscribing.settled, "the subscription");
  expect(subscribing.error).toBeUndefined();
  expect(subscribed()).toBe(true);
  expect(store().isPending("x")).toBe(false);
});

it.each([
  ["subscribe", SUBSCRIBE, false],
  ["unsubscribe", UNSUBSCRIBE, true],
] as const)(
  "%s takes is_subscribed from the read again, never its own: pending while that read is held",
  async (kind, path, before) => {
    tab.app.issue.issues.acceptRead(detail(before), tab.app.issue.captureRead());
    const changing = track(store()[kind]("ws", "A", "x"));
    expect(store().isPending("x")).toBe(true);
    (await sent(path)).answer(noContent());
    const read = await sent(READ);
    await vi.advanceTimersByTimeAsync(50);
    expect(changing.settled).toBe(false);
    expect(store().isPending("x")).toBe(true);
    expect(tab.app.issue.issues.held("x")).toMatchObject({ is_subscribed: before });
    // a subscription changes nothing of the work item: nerve reads it again as of the same moment
    read.answer(json(200, detail(!before)));
    await until(() => changing.settled, "the change");
    expect(subscribed()).toBe(!before);
    expect(store().isPending("x")).toBe(false);
  }
);

it.each([
  [404, "issue.not_found"],
  [403, "forbidden"],
  [503, "server_busy"],
])("reads the work item again after nerve refuses (%s %s), then fails with the refusal", async (status, code) => {
  const subscribing = track(store().subscribe("ws", "A", "x"));
  (await sent(SUBSCRIBE)).answer(problem(status, code));
  const read = await sent(READ);
  expect(subscribing.settled).toBe(false);
  expect(store().isPending("x")).toBe(true);
  read.answer(status === 404 ? problem(404, "issue.not_found") : json(200, detail(false)));
  await until(() => subscribing.settled, "the refusal");
  expect(subscribing.error).toBeInstanceOf(ApiError);
  expect(subscribing.error).toMatchObject({ status, problem: { code } });
  expect(store().isPending("x")).toBe(false);
  expect(tab.app.issue.issues.held("x")).toMatchObject({ is_subscribed: false });
});

it("is made once nerve made it, though the read again fails: that failure is the work item's own, for its retry", async () => {
  const subscribing = track(store().subscribe("ws", "A", "x"));
  (await sent(SUBSCRIBE)).answer(noContent());
  (await sent(READ)).answer(problem(503, "server_busy"));
  await until(() => subscribing.settled, "the subscription");
  expect(subscribing.error).toBeUndefined();
  expect(tab.app.issue.verification.getError("x")).toMatchObject({ status: 503 });
  expect(store().isPending("x")).toBe(false);
});

it("sends nothing once the work item's issues have retired while it waited in the queue", async () => {
  const gate = heldChange<void>();
  const holding = tab.app.issue.writes.enqueueIssueChanges(["x"], () => gate.sent);
  const subscribing = track(store().subscribe("ws", "A", "x"));
  await vi.advanceTimersByTimeAsync(10);
  tab.app.issue.retire();
  gate.answer();
  await holding;
  await until(() => subscribing.settled, "the retired subscription");
  expect(subscribing.error).toBeInstanceOf(SessionChangedError);
  expect(tab.nerve.calls).toHaveLength(0);
  expect(store().isPending("x")).toBe(false);
});
