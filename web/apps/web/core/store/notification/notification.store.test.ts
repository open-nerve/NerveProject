/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { NotificationType } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, json, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
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
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// The caller's notifications and unread counts in each of his workspaces (P4 spec 2.3, 2.4, R5–R7, R21), against a
// fake nerve that answers each request when the test says. The tab's caller has acme and beta; the store keeps each
// workspace's values by its id and its readers name the workspace by the address's slug, as the caller's workspaces
// list it now.

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("NotificationStore, the unread counts", () => {
  it("keeps each workspace's counts apart, by its id, and gives them by the address's slug", async () => {
    const { nerve, store } = await notificationTab();
    expect(store.getUnreadCount(acme.slug)).toBeUndefined();
    const fetched = await loadUnreadCount(nerve, store, acme, countOf(3, 1));
    expect(nerve.calls[0]?.query).toEqual({});
    expect(fetched.value).toEqual(countOf(3, 1));
    expect(store.getUnreadCount(acme.slug)).toEqual(countOf(3, 1));
    expect(store.getUnreadCount(beta.slug)).toBeUndefined();

    await loadUnreadCount(nerve, store, beta, countOf(0, 2));
    expect(store.getUnreadCount(beta.slug)).toEqual(countOf(0, 2));
    expect(store.getUnreadCount(acme.slug)).toEqual(countOf(3, 1));
    // acme's again: its own count changes, beta's stays
    await loadUnreadCount(nerve, store, acme, countOf(0, 0));
    expect(store.getUnreadCount(acme.slug)).toEqual(countOf(0, 0));
    expect(store.getUnreadCount(beta.slug)).toEqual(countOf(0, 2));
  });

  it("gives nothing for a slug the caller's workspaces do not name now: never his, left, or another workspace's since", async () => {
    const { nerve, store, workspaceRoot } = await notificationTab();
    await loadUnreadCount(nerve, store, acme, countOf(3, 1));
    expect(store.getUnreadCount("gamma")).toBeUndefined();

    // he leaves acme: the count the store holds of it no longer shows
    await loadWorkspaces(nerve, workspaceRoot, [beta]);
    expect(store.getUnreadCount(acme.slug)).toBeUndefined();

    // a new workspace takes the slug acme: it is another workspace, of no count fetched yet
    const newAcme = workspaceOf("acme", { id: "id-acme-new" });
    await loadWorkspaces(nerve, workspaceRoot, [newAcme, beta]);
    expect(store.getUnreadCount("acme")).toBeUndefined();
    await loadUnreadCount(nerve, store, newAcme, countOf(5, 0));
    expect(store.getUnreadCount("acme")).toEqual(countOf(5, 0));
  });

  it("writes no answer of a fetch a newer one overtook, whichever nerve answers first", async () => {
    const { nerve, store } = await notificationTab();
    const path = notificationRequests.unreadCount(acme);
    const older = track(store.fetchUnreadCount(acme));
    const newer = track(store.fetchUnreadCount(acme));
    await until(() => nerve.calls.length === 2, "both fetches");
    expect(nerve.calls.map((call) => call.path)).toEqual([path, path]);
    nerve.calls[0]?.answer(json(200, countOf(9, 9)));
    await until(() => older.settled, "the older answer");
    expect(older.value).toBeUndefined();
    expect(store.getUnreadCount(acme.slug)).toBeUndefined();
    nerve.calls[1]?.answer(json(200, countOf(1, 0)));
    await until(() => newer.settled, "the newer answer");
    expect(newer.value).toEqual(countOf(1, 0));
    expect(store.getUnreadCount(acme.slug)).toEqual(countOf(1, 0));

    // the other order: the newer answers first, and the older, answered after it, writes nothing
    const late = track(store.fetchUnreadCount(acme));
    const latest = track(store.fetchUnreadCount(acme));
    await until(() => nerve.calls.length === 4, "both fetches again");
    nerve.calls[3]?.answer(json(200, countOf(2, 2)));
    await until(() => latest.settled, "the newest answer");
    nerve.calls[2]?.answer(json(200, countOf(7, 7)));
    await until(() => late.settled, "the late answer");
    expect(late.value).toBeUndefined();
    expect(store.getUnreadCount(acme.slug)).toEqual(countOf(2, 2));
  });

  it("hands nerve's refusal to the caller, the count as it was; gives nothing once the session changed", async () => {
    const { nerve, api, store } = await notificationTab();
    await loadUnreadCount(nerve, store, acme, countOf(3, 1));
    const refused = track(store.fetchUnreadCount(acme));
    await until(() => nerve.calls.length === 2, "the fetch");
    nerve.calls[1]?.answer(problem(404, "workspace.not_found"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(refused.error).toMatchObject({ status: 404, problem: { code: "workspace.not_found" } });
    expect(store.getUnreadCount(acme.slug)).toEqual(countOf(3, 1));

    // another session took the tab: the fetch is no failure, it gives nothing, and nothing changes
    FakeNerve.replaceSession(api);
    const cut = track(store.fetchUnreadCount(acme));
    await until(() => cut.settled, "the cut fetch");
    expect(cut).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(2);
    expect(store.getUnreadCount(acme.slug)).toEqual(countOf(3, 1));
  });
});

/** The page's first filters: all, no type, none of the three toggles. */
const first: NotificationFilters = { tab: "all", types: [], unread: false, archived: false, snoozed: false };
const unread: NotificationFilters = { ...first, unread: true };
const [one, two, three, four] = ["one", "two", "three", "four"].map((name) => notificationOf(name));

describe("NotificationStore, the lists", () => {
  it("keeps each first page by the workspace's id and the filters' key, and gives it by the address's slug", async () => {
    const { nerve, store } = await notificationTab();
    expect(store.getList(acme.slug, first)).toBeUndefined();
    const page = pageOf([one, two], "c-1");
    const fetched = await loadList(nerve, store, acme, first, page);
    expect(nerve.calls[0]?.query).toEqual({ tab: "all", archived: "false", snoozed: "false", limit: "100" });
    expect(fetched.value).toEqual(page);
    expect(store.getList(acme.slug, first)).toEqual(page);
    expect(store.getList(acme.slug, unread)).toBeUndefined();
    expect(store.getList(beta.slug, first)).toBeUndefined();

    // the key names what is fetched: types in another order name the same list, and nerve is asked for them in order
    const assigned = pageOf([notificationOf("assigned", { sender: "assigned" })]);
    const types: NotificationType[] = ["created", "assigned"];
    await loadList(nerve, store, acme, { ...first, types }, assigned);
    expect(nerve.calls[1]?.query).toMatchObject({ type: "assigned,created" });
    expect(store.getList(acme.slug, { ...first, types: ["assigned", "created"] })).toEqual(assigned);
    // the mentions tab, with the all tab's types kept: no type asked, the same list as with none
    const mentions = pageOf([notificationOf("mention", { sender: "mentioned" })]);
    await loadList(nerve, store, acme, { ...first, tab: "mentions", types }, mentions);
    expect(nerve.calls[2]?.query).toEqual({ tab: "mentions", archived: "false", snoozed: "false", limit: "100" });
    expect(store.getList(acme.slug, { ...first, tab: "mentions" })).toEqual(mentions);
    expect(store.getList(acme.slug, first)).toEqual(page);
  });

  it("gives no list for a slug the caller's workspaces do not name now: left, or another workspace's since", async () => {
    const { nerve, store, workspaceRoot } = await notificationTab();
    await loadList(nerve, store, acme, first, pageOf([one]));
    await loadWorkspaces(nerve, workspaceRoot, [beta]);
    expect(store.getList(acme.slug, first)).toBeUndefined();

    // a new workspace takes the slug acme: it is another workspace, of no list fetched yet
    const newAcme = workspaceOf("acme", { id: "id-acme-new" });
    await loadWorkspaces(nerve, workspaceRoot, [newAcme, beta]);
    expect(store.getList("acme", first)).toBeUndefined();
    const theirs = pageOf([notificationOf("new", { workspace_id: newAcme.id })]);
    await loadList(nerve, store, newAcme, first, theirs);
    expect(store.getList("acme", first)).toEqual(theirs);
  });

  it("reads the page after the held page's cursor: adds the rows it lacks, takes the answer's cursor, stops at the last", async () => {
    const { nerve, store } = await notificationTab();
    const path = notificationRequests.list(acme);
    await loadList(nerve, store, acme, unread, pageOf([one, two], "c-1"));
    // two moved onto the next page, as a late notification lands on a page read already: it shows once
    const more = await answered(
      nerve,
      () => store.fetchMore(acme, unread),
      ["GET", path],
      pageOf([two, three], "c-2"),
      "the next page"
    );
    expect(nerve.calls[1]?.query).toEqual({
      tab: "all",
      read: "false",
      archived: "false",
      snoozed: "false",
      limit: "100",
      cursor: "c-1",
    });
    expect(more.value).toEqual(pageOf([one, two, three], "c-2"));
    expect(store.getList(acme.slug, unread)).toEqual(pageOf([one, two, three], "c-2"));

    await answered(nerve, () => store.fetchMore(acme, unread), ["GET", path], pageOf([four]), "the last page");
    expect(nerve.calls[2]?.query.cursor).toBe("c-2");
    expect(store.getList(acme.slug, unread)).toEqual(pageOf([one, two, three, four]));
    // the last page held: there is nothing more to ask
    const past = await settle(store.fetchMore(acme, unread), "past the last");
    expect(past).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(3);
  });

  it("reads on from a page of no rows that has a cursor, and asks nothing for a list not fetched (re-93 m6)", async () => {
    const { nerve, store } = await notificationTab();
    const none = await settle(store.fetchMore(acme, first), "a list not fetched");
    expect(none).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(0);

    await loadList(nerve, store, acme, first, pageOf([], "c-1"));
    const path = notificationRequests.list(acme);
    await answered(nerve, () => store.fetchMore(acme, first), ["GET", path], pageOf([one]), "the next page");
    expect(nerve.calls[1]?.query.cursor).toBe("c-1");
    expect(store.getList(acme.slug, first)).toEqual(pageOf([one]));
  });

  it("a refresh reads the first page in place of every page held; a next page out writes nothing (R7, W4)", async () => {
    const { nerve, store } = await notificationTab();
    const path = notificationRequests.list(acme);
    await loadList(nerve, store, acme, first, pageOf([one, two], "c-1"));
    await answered(nerve, () => store.fetchMore(acme, first), ["GET", path], pageOf([three], "c-2"), "the next page");
    const more = track(store.fetchMore(acme, first));
    await until(() => nerve.calls.length === 3, "the page after");
    expect(nerve.calls[2]?.query.cursor).toBe("c-2");

    // two was archived in another tab: the refresh no longer lists it, nor any row of the pages after the first
    const refreshed = pageOf([four, one], "c-9");
    const refresh = track(store.fetchList(acme, first));
    await until(() => nerve.calls.length === 4, "the refresh");
    nerve.calls[3]?.answer(json(200, refreshed));
    await until(() => refresh.settled, "the refresh");
    expect(store.getList(acme.slug, first)).toEqual(refreshed);
    nerve.calls[2]?.answer(json(200, pageOf([notificationOf("five")], "c-3")));
    await until(() => more.settled, "the page the refresh superseded");
    expect(more.value).toBeUndefined();
    expect(store.getList(acme.slug, first)).toEqual(refreshed);
  });

  it("a next page sent while a refresh is out follows nothing once the refresh ends elsewhere (whole-branch I1)", async () => {
    const { nerve, store } = await notificationTab();
    const path = notificationRequests.list(acme);
    await loadList(nerve, store, acme, first, pageOf([one, two], "c-1"));
    await answered(nerve, () => store.fetchMore(acme, first), ["GET", path], pageOf([three], "c-2"), "the next page");

    // the refresh is out as the caller asks for more: that read names the cursor of the pages the refresh replaces
    const refresh = track(store.fetchList(acme, first));
    await until(() => nerve.calls.length === 3, "the refresh");
    const more = track(store.fetchMore(acme, first));
    await until(() => nerve.calls.length === 4, "the page after the held pages");
    expect(nerve.calls[3]?.query.cursor).toBe("c-2");

    const refreshed = pageOf([four, one], "c-9");
    nerve.calls[2]?.answer(json(200, refreshed));
    await until(() => refresh.settled, "the refresh");
    nerve.calls[3]?.answer(json(200, pageOf([notificationOf("five")], "c-3")));
    await until(() => more.settled, "the page after the pages the refresh replaced");
    // the refreshed first page ends at c-9: the page after c-2 is not its next, so the list skips nothing
    expect(store.getList(acme.slug, first)).toEqual(refreshed);
  });

  it("writes a workspace's late answers into its own list alone: another's rows and cursor stay (W5)", async () => {
    const { nerve, store } = await notificationTab();
    const theirs = pageOf([notificationOf("b-one", { workspace_id: beta.id })], "cb-1");
    const ours = pageOf([notificationOf("a-one")], "ca-1");
    // acme's first page is out as the caller moves to beta, whose page nerve answers first
    const acmeList = track(store.fetchList(acme, first));
    await until(() => nerve.calls.length === 1, "acme's list");
    await loadList(nerve, store, beta, first, theirs);
    nerve.calls[0]?.answer(json(200, ours));
    await until(() => acmeList.settled, "acme's late list");
    expect(store.getList(beta.slug, first)).toEqual(theirs);
    expect(store.getList(acme.slug, first)).toEqual(ours);

    // acme's next page, out as the caller moves to beta again: it lands in acme's list alone
    const acmeMore = track(store.fetchMore(acme, first));
    await until(() => nerve.calls.length === 3, "acme's next page");
    expect(nerve.calls[2]).toMatchObject({ path: notificationRequests.list(acme), query: { cursor: "ca-1" } });
    nerve.calls[2]?.answer(json(200, pageOf([notificationOf("a-two")])));
    await until(() => acmeMore.settled, "acme's late page");
    expect(store.getList(beta.slug, first)).toEqual(theirs);
    expect(store.getList(acme.slug, first)).toEqual(pageOf([notificationOf("a-one"), notificationOf("a-two")]));
  });

  it.each<[string, (store: INotificationStore) => Promise<unknown>]>([
    ["the first page", (store) => store.fetchList(acme, first)],
    ["the next page", (store) => store.fetchMore(acme, first)],
  ])(
    "hands nerve's refusal of %s to the caller, the list as it was; nothing once the session changed",
    async (_, read) => {
      const { nerve, api, store } = await notificationTab();
      const page = pageOf([one], "c-1");
      await loadList(nerve, store, acme, first, page);
      const refused = track(read(store));
      await until(() => nerve.calls.length === 2, "the read");
      nerve.calls[1]?.answer(problem(404, "workspace.not_found"));
      await until(() => refused.settled, "the refusal");
      expect(refused.error).toBeInstanceOf(ApiError);
      expect(store.getList(acme.slug, first)).toEqual(page);

      // another session took the tab: the read is no failure, it gives nothing, and nothing changes
      FakeNerve.replaceSession(api);
      expect(await settle(read(store), "the cut read")).toEqual({ settled: true, value: undefined });
      expect(nerve.calls).toHaveLength(2);
      expect(store.getList(acme.slug, first)).toEqual(page);
    }
  );
});
