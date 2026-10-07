/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { runInAction } from "mobx";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspaceMember } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import type { FakeNerve } from "@/lib/auth/fake-nerve";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { memberStore, membershipOf } from "@/store/member/workspace/fake-members";
import type { WorkspaceMemberStore } from "@/store/member/workspace/workspace-member.store";

// The members of a workspace (M3 design 7.3), against a fake nerve that answers each request when the test says.

// The account's store reads the tab's session as it fetches the account, which no test here does.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

const MEMBERS = "/api/v0/workspaces/acme/members";

const ann = membershipOf("ann", { role: 20 });
const bob = membershipOf("bob");
/** A member who was removed: nerve lists his membership, ended. */
const cat = membershipOf("cat", { role: 5, is_active: false });

/** The store fetches acme's members, and nerve lists these. */
async function load(nerve: FakeNerve, store: WorkspaceMemberStore, memberships: WorkspaceMember[]) {
  const at = nerve.calls.length;
  const fetched = store.fetchWorkspaceMembers("acme");
  await until(() => nerve.calls.length === at + 1, "the members");
  nerve.calls[at]?.answer(json(200, { data: memberships }));
  return settle(fetched, "the members");
}

/** A store whose members of acme nerve gave as ann, bob and cat. */
async function loaded() {
  const tab = memberStore();
  await load(tab.nerve, tab.store, [ann, bob, cat]);
  return tab;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("WorkspaceMemberStore, the members", () => {
  it("keeps a workspace's memberships as nerve lists them, ended ones too, and shares the members' profiles", async () => {
    const { nerve, users, store } = memberStore();
    const fetched = await load(nerve, store, [ann, bob, cat]);
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: MEMBERS });
    expect(fetched.value).toEqual([ann, bob, cat]);
    expect(store.workspaceMemberMap).toEqual({ acme: { "u-ann": ann, "u-bob": bob, "u-cat": cat } });
    expect(users).toEqual({ "u-ann": ann.member, "u-bob": bob.member, "u-cat": cat.member });
    expect(store.getWorkspaceMemberDetails("u-bob")).toEqual(bob);
    expect(store.getWorkspaceMemberDetails("u-zed")).toBeNull();
    expect(store.isUserSuspended("u-cat", "acme")).toBe(true);
    expect(store.isUserSuspended("u-bob", "acme")).toBe(false);

    // a fetch again holds the list as nerve has it now
    const promoted = membershipOf("bob", { role: 20 });
    await load(nerve, store, [ann, promoted]);
    expect(store.workspaceMemberMap).toEqual({ acme: { "u-ann": ann, "u-bob": promoted } });
  });

  it("lists the caller first, then the others by display name", async () => {
    const { user, store } = await loaded();
    expect(store.getWorkspaceMemberIds("acme")).toEqual(["u-ann", "u-bob", "u-cat"]);
    runInAction(() => {
      user.data = {
        ...bob.member,
        email: "bob@example.com",
        user_timezone: "UTC",
        cover_image_url: null,
        created_at: "2026-09-01T09:00:00Z",
      };
    });
    expect(store.getWorkspaceMemberIds("acme")).toEqual(["u-bob", "u-ann", "u-cat"]);
  });

  it("fails when nerve cannot list them, keeping the members it had", async () => {
    const { nerve, store } = memberStore();
    const first = track(store.fetchWorkspaceMembers("acme"));
    await until(() => nerve.calls.length === 1, "the members");
    nerve.calls[0]?.answer(problem(404, "workspace.not_found"));
    await until(() => first.settled, "the failure");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(store.workspaceMemberMap).toEqual({});

    await load(nerve, store, [ann, bob]);
    const again = track(store.fetchWorkspaceMembers("acme"));
    await until(() => nerve.calls.length === 3, "the refetch");
    nerve.calls[2]?.fail();
    await until(() => again.settled, "the failure");
    // The failure is the caller's to handle (SWR's error), not an unhandled rejection.
    expect(again.error).toBeInstanceOf(TypeError);
    expect(store.workspaceMemberMap).toEqual({ acme: { "u-ann": ann, "u-bob": bob } });
  });

  it("gives nothing, and does not fail, when the session changes as it fetches", async () => {
    const { nerve, store } = memberStore((fake) => fake.replacedSessionClient());
    const fetched = await settle(store.fetchWorkspaceMembers("acme"), "the fetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toEqual([]);
    expect(store.workspaceMemberMap).toEqual({});
  });
});

describe("WorkspaceMemberStore, the changes", () => {
  it("changes a member's role to nerve's answer, once nerve gives it", async () => {
    const { nerve, store } = await loaded();
    const demoted = membershipOf("bob", { role: 5 });
    const updated = track(store.updateMember("acme", "u-bob", { role: 5 }));
    await until(() => nerve.calls.length === 2, "the change");
    expect(nerve.calls[1]).toMatchObject({
      method: "PATCH",
      path: "/api/v0/workspace-members/m-bob",
      body: { role: 5 },
    });
    // until nerve answers, the role is as it was
    expect(store.workspaceMemberMap.acme?.["u-bob"]).toEqual(bob);
    nerve.calls[1]?.answer(json(200, demoted));
    await until(() => updated.settled, "the answer");
    expect(updated.value).toEqual(demoted);
    expect(store.workspaceMemberMap.acme).toEqual({ "u-ann": ann, "u-bob": demoted, "u-cat": cat });
  });

  it("keeps a removed member's membership, ended, as nerve lists it", async () => {
    const { nerve, store } = await loaded();
    const removed = track(store.removeMemberFromWorkspace("acme", "u-bob"));
    await until(() => nerve.calls.length === 2, "the removal");
    expect(nerve.calls[1]).toMatchObject({ method: "DELETE", path: "/api/v0/workspace-members/m-bob" });
    expect(store.isUserSuspended("u-bob", "acme")).toBe(false);
    nerve.calls[1]?.answer(noContent());
    await until(() => removed.settled, "the removal");
    expect(removed.error).toBeUndefined();
    expect(store.workspaceMemberMap.acme?.["u-bob"]).toEqual({ ...bob, is_active: false });
    expect(store.isUserSuspended("u-bob", "acme")).toBe(true);
  });

  const refusals: { change: string; send: (store: WorkspaceMemberStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "a role change",
      send: (store) => store.updateMember("acme", "u-ann", { role: 15 }),
      refusal: problem(409, "workspace.own_membership"),
    },
    {
      change: "a removal",
      send: (store) => store.removeMemberFromWorkspace("acme", "u-bob"),
      refusal: problem(409, "project.sole_admin"),
    },
  ];
  it.each(refusals)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, store } = await loaded();
    const sent = track(send(store));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(refusal);
    await until(() => sent.settled, "the refusal");
    expect(sent.error).toBeInstanceOf(ApiError);
    expect(store.workspaceMemberMap.acme).toEqual({ "u-ann": ann, "u-bob": bob, "u-cat": cat });
  });

  it.each(refusals)("fails, asking nerve nothing, for $change of a member it has not listed", async ({ send }) => {
    const { nerve, store } = memberStore();
    const sent = await settle(send(store), "the change");
    expect(sent).toMatchObject({ settled: true, error: new Error("Member not found") });
    expect(nerve.calls).toEqual([]);
  });

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const { nerve, store } = await loaded();
    const demoted = membershipOf("bob", { role: 5 });
    const promoted = track(store.updateMember("acme", "u-bob", { role: 20 }));
    const changed = track(store.updateMember("acme", "u-bob", { role: 5 }));
    const removed = track(store.removeMemberFromWorkspace("acme", "u-bob"));
    await inTurn(nerve, 1, ["PATCH", "/api/v0/workspace-members/m-bob"], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", "/api/v0/workspace-members/m-bob"], json(200, demoted));
    await inTurn(nerve, 3, ["DELETE", "/api/v0/workspace-members/m-bob"], noContent());
    await until(() => removed.settled, "the last change");
    expect(promoted.error).toBeInstanceOf(ApiError);
    expect(changed.value).toEqual(demoted);
    expect(store.workspaceMemberMap.acme?.["u-bob"]).toEqual({ ...demoted, is_active: false });
  });

  it("fetches the members while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      () => store.updateMember("acme", "u-bob", { role: 5 }),
      () => store.fetchWorkspaceMembers("acme"),
      ["GET", MEMBERS],
      json(200, { data: [ann, bob] })
    );
    expect(store.workspaceMemberMap).toEqual({ acme: { "u-ann": ann, "u-bob": bob } });
  });
});
