/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import {
  acme,
  beta,
  countOf,
  loadUnreadCount,
  notificationRequests,
  notificationTab,
} from "@/store/notification/fake-notifications";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// The caller's unread notifications in each of his workspaces (P4 spec 2.3, R5, R6), against a fake nerve that answers
// each request when the test says. The tab's caller has acme and beta; the store keeps each workspace's count by its
// id and its readers name the workspace by the address's slug, as the caller's workspaces list it now.

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
