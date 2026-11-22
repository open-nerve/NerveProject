/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Notification, NotificationUnreadCount } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import type { Endpoint } from "@/lib/auth/fake-nerve";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { inTurn, sent } from "@/store/fake-queue";
import {
  acme,
  beta,
  countOf,
  loadList,
  loadUnreadCount,
  notificationOf,
  notificationRequests,
  notificationTab,
  pageOf,
} from "@/store/notification/fake-notifications";
import type { NotificationFilters } from "@/store/notification/filters";
import type { INotificationStore } from "@/store/notification/notification.store";

// The changes of the caller's notifications (P4 spec 2.3, R6, R8, R9; re-93 m2, m4; v0 design 7.7), against a fake
// nerve that answers each request when the test says. They go one at a time. Once nerve confirms a change, its answer
// is written into every list the store holds of the workspace: in its place where the list still lists it (its read
// state never moves it), out of a list it no longer matches, into none it newly would; then the workspace's unread
// count is fetched again, as only nerve counts it. Mark all read changes no row here: the filters' first page and the
// count are fetched again, whether nerve marked them or refused.

const { list, unreadCount, markAllRead, at } = notificationRequests;

/** The moment of the tests; nerve's answers are of moments after it. */
const NOW = new Date("2026-11-10T09:00:00Z");
const LATER = "2026-11-10T09:30:00Z";
const TOMORROW = "2026-11-11T09:00:00Z";

beforeEach(() => {
  vi.useFakeTimers({ now: NOW });
});
afterEach(() => {
  vi.useRealTimers();
});

const first: NotificationFilters = { tab: "all", types: [], unread: false, archived: false, snoozed: false };
const unread: NotificationFilters = { ...first, unread: true };
const archive: NotificationFilters = { ...first, archived: true };
const snoozes: NotificationFilters = { ...first, snoozed: true };

const [one, two, three, four, five, six] = ["one", "two", "three", "four", "five", "six"].map((name) =>
  notificationOf(name)
);
const shelved = notificationOf("shelved", { read_at: "2026-11-01T10:00:00Z", archived_at: "2026-11-02T10:00:00Z" });
const dozing = notificationOf("dozing", { snoozed_till: TOMORROW });
const theirs = notificationOf("theirs", { workspace_id: beta.id });

/** A tab whose store holds acme's first, unread, archived and snoozed lists and its count, and beta's first list. */
async function heldLists() {
  const tab = await notificationTab();
  const { nerve, store } = tab;
  await loadList(nerve, store, acme, first, pageOf([one, two, three], "c-1"));
  await loadList(nerve, store, acme, unread, pageOf([one, two, three]));
  await loadList(nerve, store, acme, archive, pageOf([shelved]));
  await loadList(nerve, store, acme, snoozes, pageOf([dozing]));
  await loadList(nerve, store, beta, first, pageOf([theirs]));
  await loadUnreadCount(nerve, store, acme, countOf(3, 0));
  return tab;
}

/** Every value the store holds: acme's four lists and its count, beta's list. */
const held = (store: INotificationStore) => ({
  first: store.getList(acme.slug, first),
  unread: store.getList(acme.slug, unread),
  archive: store.getList(acme.slug, archive),
  snoozes: store.getList(acme.slug, snoozes),
  count: store.getUnreadCount(acme.slug),
  theirs: store.getList(beta.slug, first),
});

/** Whether the store shows acme's count as count. */
const counted = (store: INotificationStore, count: NotificationUnreadCount) => () =>
  store.getUnreadCount(acme.slug)?.all === count.all && store.getUnreadCount(acme.slug)?.mentions === count.mentions;

/** A change of acme's that nerve confirms with reply; then acme's count, fetched again, which nerve gives as count. */
async function confirmed(
  nerve: FakeNerve,
  store: INotificationStore,
  send: () => Promise<unknown>,
  request: Endpoint,
  reply: Response,
  count: NotificationUnreadCount
) {
  const k = nerve.calls.length;
  const change = await sent(nerve, send, request, reply);
  await inTurn(nerve, k + 1, ["GET", unreadCount(acme)], json(200, count));
  await until(counted(store, count), "the count again");
  return change;
}

describe("NotificationStore, the changes", () => {
  it("reads a notification: nerve's answer in its place in every list of the workspace, the unread one too (R8)", async () => {
    const { nerve, store } = await heldLists();
    const others = store.getList(beta.slug, first);
    const read: Notification = { ...one, read_at: LATER };
    const k = nerve.calls.length;
    const send = () => store.update(acme, one.id, { read: true });
    const reading = await confirmed(nerve, store, send, ["PATCH", at(one)], json(200, read), countOf(2, 0));
    expect(nerve.calls[k]?.body).toEqual({ read: true });
    expect(reading.value).toEqual(read);
    expect(held(store)).toEqual({
      first: pageOf([read, two, three], "c-1"),
      unread: pageOf([read, two, three]),
      archive: pageOf([shelved]),
      snoozes: pageOf([dozing]),
      count: countOf(2, 0),
      theirs: pageOf([theirs]),
    });
    // another workspace's list is not so much as written
    expect(store.getList(beta.slug, first)).toBe(others);

    // unread again: in its place again
    const unreadAgain = () => store.update(acme, one.id, { read: false });
    await confirmed(nerve, store, unreadAgain, ["PATCH", at(one)], json(200, one), countOf(3, 0));
    expect(held(store)).toMatchObject({ first: pageOf([one, two, three], "c-1"), unread: pageOf([one, two, three]) });
  });

  it("archives, snoozes, ends a snooze, unarchives: out of each list that no longer lists it, into none (R8)", async () => {
    const { nerve, store } = await heldLists();
    const archived: Notification = { ...two, archived_at: LATER };
    const archiving = () => store.update(acme, two.id, { archived: true });
    await confirmed(nerve, store, archiving, ["PATCH", at(two)], json(200, archived), countOf(2, 0));
    expect(held(store)).toMatchObject({
      first: pageOf([one, three], "c-1"),
      unread: pageOf([one, three]),
      archive: pageOf([shelved]),
    });

    const snoozed: Notification = { ...three, snoozed_till: TOMORROW };
    const snoozing = () => store.update(acme, three.id, { snoozed_till: TOMORROW });
    await confirmed(nerve, store, snoozing, ["PATCH", at(three)], json(200, snoozed), countOf(1, 0));
    expect(nerve.calls.at(-2)?.body).toEqual({ snoozed_till: TOMORROW });
    expect(held(store)).toMatchObject({
      first: pageOf([one], "c-1"),
      unread: pageOf([one]),
      snoozes: pageOf([dozing]),
    });

    // each undone in its own view: an end of a snooze sends null (T1)
    const woken: Notification = { ...dozing, snoozed_till: null };
    const waking = () => store.update(acme, dozing.id, { snoozed_till: null });
    await confirmed(nerve, store, waking, ["PATCH", at(dozing)], json(200, woken), countOf(2, 0));
    expect(nerve.calls.at(-2)?.body).toEqual({ snoozed_till: null });
    const unshelved: Notification = { ...shelved, archived_at: null };
    const unarchiving = () => store.update(acme, shelved.id, { archived: false });
    await confirmed(nerve, store, unarchiving, ["PATCH", at(shelved)], json(200, unshelved), countOf(2, 0));
    expect(held(store)).toEqual({
      first: pageOf([one], "c-1"),
      unread: pageOf([one]),
      archive: pageOf([]),
      snoozes: pageOf([]),
      count: countOf(2, 0),
      theirs: pageOf([theirs]),
    });
  });

  it("drops a notification nerve no longer gives from every list of the workspace, fetches the count, hands the refusal on", async () => {
    const { nerve, store } = await heldLists();
    const others = store.getList(beta.slug, first);
    const k = nerve.calls.length;
    const gone = track(store.update(acme, two.id, { read: true }));
    await inTurn(nerve, k, ["PATCH", at(two)], problem(404, "notification.not_found"));
    await inTurn(nerve, k + 1, ["GET", unreadCount(acme)], json(200, countOf(2, 0)));
    await until(() => gone.settled && counted(store, countOf(2, 0))(), "the refusal and the count");
    expect(gone.error).toBeInstanceOf(ApiError);
    expect(gone.error).toMatchObject({ status: 404, problem: { code: "notification.not_found" } });
    expect(held(store)).toEqual({
      first: pageOf([one, three], "c-1"),
      unread: pageOf([one, three]),
      archive: pageOf([shelved]),
      snoozes: pageOf([dozing]),
      count: countOf(2, 0),
      theirs: pageOf([theirs]),
    });
    expect(store.getList(beta.slug, first)).toBe(others);
  });

  it.each<[string, Response]>([
    ["busy", problem(503, "server_busy")],
    ["a snooze not after now", problem(422, "validation_failed")],
  ])("leaves every value as it was when nerve refuses a change (%s), and hands the refusal on", async (_, refusal) => {
    const { nerve, store } = await heldLists();
    const before = held(store);
    const k = nerve.calls.length;
    const refused = track(store.update(acme, one.id, { snoozed_till: "2026-11-10T08:00:00Z" }));
    await inTurn(nerve, k, ["PATCH", at(one)], refusal);
    await until(() => refused.settled, "the refusal");
    await vi.advanceTimersByTimeAsync(1_000);
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(held(store)).toEqual(before);
    expect(nerve.calls).toHaveLength(k + 1);
  });

  it("sends the changes one at a time, each once nerve has answered the one before", async () => {
    const { nerve, store } = await heldLists();
    // nerve answers at once what the changes fetch again
    nerve.respond = (call) => {
      if (call.method === "GET") call.answer(json(200, call.path === list(acme) ? pageOf([]) : countOf(1, 0)));
    };
    const changes = () => nerve.calls.filter((call) => call.method !== "GET");
    const sentSoFar = () => changes().map(({ method, path }) => [method, path]);
    const reading = track(store.update(acme, one.id, { read: true }));
    const archiving = track(store.update(acme, two.id, { archived: true }));
    const marking = track(store.markAllRead(acme, first));
    await until(() => changes().length === 1, "the first change");
    await vi.advanceTimersByTimeAsync(1_000);
    expect(sentSoFar()).toEqual([["PATCH", at(one)]]);
    changes()[0]?.answer(json(200, { ...one, read_at: LATER }));
    await until(() => changes().length === 2, "the second change");
    await vi.advanceTimersByTimeAsync(1_000);
    expect(sentSoFar()).toEqual([
      ["PATCH", at(one)],
      ["PATCH", at(two)],
    ]);
    changes()[1]?.answer(json(200, { ...two, archived_at: LATER }));
    await until(() => changes().length === 3, "mark all read");
    expect(changes()[2]).toMatchObject({
      method: "POST",
      path: markAllRead(acme),
      body: { tab: "all", archived: false, snoozed: false },
    });
    changes()[2]?.answer(json(200, { marked: 1 }));
    await until(() => [reading, archiving, marking].every((change) => change.settled), "the changes");
    expect(marking.value).toEqual({ marked: 1 });
  });

  it("marks all read with the page's filters and changes no row here; their first page and the count come again (R9)", async () => {
    const { nerve, store } = await heldLists();
    const typed: NotificationFilters = { ...first, types: ["subscribed", "assigned"] };
    await loadList(nerve, store, acme, typed, pageOf([one, two]));
    const before = held(store);
    const k = nerve.calls.length;
    const marking = track(store.markAllRead(acme, typed));
    await inTurn(nerve, k, ["POST", markAllRead(acme)], json(200, { marked: 2 }));
    const body = { tab: "all", type: ["assigned", "subscribed"], archived: false, snoozed: false };
    expect(nerve.calls[k]?.body).toEqual(body);
    await until(() => marking.settled && nerve.calls.length === k + 3, "the marking and what it fetches again");
    expect(marking.value).toEqual({ marked: 2 });
    // no row is marked read here, nor any count zeroed: each stays as nerve gave it until nerve gives it again
    expect(held(store)).toEqual(before);
    expect(store.getList(acme.slug, typed)).toEqual(pageOf([one, two]));

    const again = nerve.calls.slice(k + 1);
    const page = again.find((call) => call.path === list(acme));
    expect(page?.query).toEqual({
      tab: "all",
      type: "assigned,subscribed",
      archived: "false",
      snoozed: "false",
      limit: "100",
    });
    const marked = pageOf([
      { ...one, read_at: LATER },
      { ...two, read_at: LATER },
    ]);
    page?.answer(json(200, marked));
    again.find((call) => call.path === unreadCount(acme))?.answer(json(200, countOf(1, 0)));
    await until(counted(store, countOf(1, 0)), "the count again");
    await until(() => store.getList(acme.slug, typed)?.data[0]?.read_at === LATER, "the page again");
    expect(store.getList(acme.slug, typed)).toEqual(marked);
  });

  it("fetches the first page and the count again when nerve refuses too: the batches before stay marked (re-93 m2)", async () => {
    const { nerve, store } = await heldLists();
    const before = held(store);
    const k = nerve.calls.length;
    const marking = track(store.markAllRead(acme, first));
    await inTurn(nerve, k, ["POST", markAllRead(acme)], problem(500, "internal_error"));
    await until(() => marking.settled && nerve.calls.length === k + 3, "the refusal and what it fetches again");
    expect(marking.error).toBeInstanceOf(ApiError);
    expect(marking.error).toMatchObject({ status: 500, problem: { code: "internal_error" } });
    expect(held(store)).toEqual(before);

    // nerve marked a batch before the one that failed: the page and the count, fetched again, show it
    const partly = pageOf([{ ...one, read_at: LATER }, two, three], "c-1");
    const again = nerve.calls.slice(k + 1);
    again.find((call) => call.path === list(acme))?.answer(json(200, partly));
    again.find((call) => call.path === unreadCount(acme))?.answer(json(200, countOf(2, 0)));
    await until(counted(store, countOf(2, 0)), "the count again");
    await until(() => store.getList(acme.slug, first)?.data[0]?.read_at === LATER, "the page again");
    expect(store.getList(acme.slug, first)).toEqual(partly);
  });

  it("makes a change confirmed while a fetch is out again on its answer: a refresh's, a next page's (v0 7.7)", async () => {
    const { nerve, store } = await heldLists();
    // a refresh is out, read before two is archived; nerve lists a new notification on it, and two still
    const k = nerve.calls.length;
    const refresh = track(store.fetchList(acme, first));
    await until(() => nerve.calls.length === k + 1, "the refresh");
    const archiving = () => store.update(acme, two.id, { archived: true });
    const archived = json(200, { ...two, archived_at: LATER });
    await confirmed(nerve, store, archiving, ["PATCH", at(two)], archived, countOf(2, 0));
    nerve.calls[k]?.answer(json(200, pageOf([four, one, two, three], "c-1")));
    await until(() => refresh.settled, "the refresh");
    expect(store.getList(acme.slug, first)).toEqual(pageOf([four, one, three], "c-1"));

    // the next page is out, read before five, on it, is archived
    const m = nerve.calls.length;
    const more = track(store.fetchMore(acme, first));
    await until(() => nerve.calls.length === m + 1, "the next page");
    const shelving = () => store.update(acme, five.id, { archived: true });
    await confirmed(
      nerve,
      store,
      shelving,
      ["PATCH", at(five)],
      json(200, { ...five, archived_at: LATER }),
      countOf(2, 0)
    );
    nerve.calls[m]?.answer(json(200, pageOf([five, six])));
    await until(() => more.settled, "the next page");
    expect(store.getList(acme.slug, first)).toEqual(pageOf([four, one, three, six]));
  });

  it("changes nothing of this generation once the session has changed: each change fails before it is sent", async () => {
    const { nerve, api, store } = await heldLists();
    const before = held(store);
    const k = nerve.calls.length;
    FakeNerve.replaceSession(api);
    const cut = [track(store.update(acme, one.id, { read: true })), track(store.markAllRead(acme, first))];
    await until(() => cut.every((change) => change.settled), "the changes");
    await vi.advanceTimersByTimeAsync(1_000);
    expect(cut.map((change) => change.error)).toEqual([
      expect.any(SessionChangedError),
      expect.any(SessionChangedError),
    ]);
    expect(nerve.calls).toHaveLength(k);
    expect(held(store)).toEqual(before);
  });
});
