/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { runInAction } from "mobx";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspaceMember } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { memberStore, membershipOf } from "@/store/member/workspace/fake-members";
import type { WorkspaceMemberStore } from "@/store/member/workspace/workspace-member.store";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The members of a workspace (M3 design 7.3), against a fake nerve that answers each request when the test says.
// That a workspace's members are kept by its id, so that a slug deleted and made again shows none of the old one's,
// is root.store.test.ts.

// The account's store reads the tab's session as it fetches the account, which no test here does.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

const MEMBERS = "/api/v0/workspaces/acme/members";

/** The membership as nerve lists it once its member has changed his display name. */
function renamed(membership: WorkspaceMember, displayName: string): WorkspaceMember {
  return { ...membership, member: { ...membership.member, display_name: displayName } };
}

const ann = membershipOf("ann", { role: 20 });
const bob = membershipOf("bob");
/** A member who was removed: nerve lists his membership, ended. */
const cat = membershipOf("cat", { role: 5, is_active: false });
/** Bob's membership as nerve answers his change to a guest: he renamed himself meanwhile, which no request says. */
const demoted = renamed(membershipOf("bob", { role: 5 }), "Robert");

/** The store fetches the members of the workspace slug names (acme unless it says), and nerve lists these. */
function load(nerve: FakeNerve, store: WorkspaceMemberStore, memberships: WorkspaceMember[], slug = "acme") {
  const fetch = () => store.fetchWorkspaceMembers(workspaceOf(slug));
  return answered(nerve, fetch, ["GET", `/api/v0/workspaces/${slug}/members`], { data: memberships }, "the members");
}

/** A store whose members of acme nerve gave as ann, bob and cat. */
async function loaded() {
  const tab = await memberStore();
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
    const { nerve, users, store } = await memberStore();
    const fetched = await load(nerve, store, [ann, bob, cat]);
    expect(fetched.value).toEqual({ "u-ann": ann, "u-bob": bob, "u-cat": cat });
    expect(store.getMemberships("acme")).toEqual({ "u-ann": ann, "u-bob": bob, "u-cat": cat });
    expect(users).toEqual({ "u-ann": ann.member, "u-bob": bob.member, "u-cat": cat.member });
    expect(store.getWorkspaceMemberDetails("u-bob")).toEqual(bob);
    expect(store.getWorkspaceMemberDetails("u-zed")).toBeNull();
    expect(store.isUserSuspended("u-cat", "acme")).toBe(true);
    expect(store.isUserSuspended("u-bob", "acme")).toBe(false);

    // each workspace's list is its own; a fetch again holds acme's as nerve has it now, the profiles too
    const elsewhere = membershipOf("ann", {}, "globex");
    await load(nerve, store, [elsewhere], "globex");
    const promoted = renamed(membershipOf("bob", { role: 20 }), "Robert");
    await load(nerve, store, [ann, promoted]);
    expect(store.getMemberships("acme")).toEqual({ "u-ann": ann, "u-bob": promoted });
    expect(store.getMemberships("globex")).toEqual({ "u-ann": elsewhere });
    expect(users["u-bob"]).toEqual(promoted.member);
  });

  it("lists the caller first, then the others by display name", async () => {
    const { nerve, user, store } = await memberStore();
    /** A display name with a capital: the order is the names', whatever their case. */
    const dee = membershipOf("Dee");
    // nerve's order is not the names'
    await load(nerve, store, [dee, cat, bob, ann]);
    expect(store.getWorkspaceMemberIds("acme")).toEqual(["u-ann", "u-bob", "u-cat", "u-Dee"]);
    runInAction(() => {
      user.data = {
        ...bob.member,
        email: "bob@example.com",
        user_timezone: "UTC",
        cover_image_url: null,
        created_at: "2026-09-01T09:00:00Z",
      };
    });
    expect(store.getWorkspaceMemberIds("acme")).toEqual(["u-bob", "u-ann", "u-cat", "u-Dee"]);
  });

  it("fails when nerve cannot list them, keeping the members it had", async () => {
    const { nerve, store } = await memberStore();
    const first = track(store.fetchWorkspaceMembers(workspaceOf("acme")));
    await until(() => nerve.calls.length === 1, "the members");
    nerve.calls[0]?.answer(problem(404, "workspace.not_found"));
    await until(() => first.settled, "the failure");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(store.getMemberships("acme")).toBeUndefined();

    await load(nerve, store, [ann, bob]);
    const again = track(store.fetchWorkspaceMembers(workspaceOf("acme")));
    await until(() => nerve.calls.length === 3, "the refetch");
    nerve.calls[2]?.fail();
    await until(() => again.settled, "the failure");
    // The failure is the caller's to handle (SWR's error), not an unhandled rejection.
    expect(again.error).toBeInstanceOf(TypeError);
    expect(store.getMemberships("acme")).toEqual({ "u-ann": ann, "u-bob": bob });
  });

  it("keeps the members it had, gives nothing and does not fail, when the session changes as it fetches them again", async () => {
    const { nerve, api, users, store } = await loaded();
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchWorkspaceMembers(workspaceOf("acme")), "the refetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(store.getMemberships("acme")).toEqual({ "u-ann": ann, "u-bob": bob, "u-cat": cat });
    expect(users).toEqual({ "u-ann": ann.member, "u-bob": bob.member, "u-cat": cat.member });
  });
});

describe("WorkspaceMemberStore, the changes", () => {
  it("changes a member's role to nerve's answer, once nerve gives it, and his profile with it", async () => {
    const { nerve, users, store } = await loaded();
    const updated = track(store.updateMember("acme", "u-bob", { role: 5 }));
    await until(() => nerve.calls.length === 2, "the change");
    expect(nerve.calls[1]).toMatchObject({
      method: "PATCH",
      path: "/api/v0/workspace-members/m-acme-bob",
      body: { role: 5 },
    });
    // until nerve answers, the role is as it was
    expect(store.getMemberships("acme")?.["u-bob"]).toEqual(bob);
    nerve.calls[1]?.answer(json(200, demoted));
    await until(() => updated.settled, "the answer");
    expect(updated.value).toEqual(demoted);
    expect(store.getMemberships("acme")).toEqual({ "u-ann": ann, "u-bob": demoted, "u-cat": cat });
    // the profile the other stores read is nerve's answer's too
    expect(users["u-bob"]).toEqual(demoted.member);
  });

  it("keeps a removed member's membership, ended, as nerve lists it", async () => {
    const { nerve, store } = await loaded();
    const removed = track(store.removeMemberFromWorkspace("acme", "u-bob"));
    await until(() => nerve.calls.length === 2, "the removal");
    expect(nerve.calls[1]).toMatchObject({ method: "DELETE", path: "/api/v0/workspace-members/m-acme-bob" });
    expect(store.isUserSuspended("u-bob", "acme")).toBe(false);
    nerve.calls[1]?.answer(noContent());
    await until(() => removed.settled, "the removal");
    expect(removed.error).toBeUndefined();
    expect(store.getMemberships("acme")?.["u-bob"]).toEqual({ ...bob, is_active: false });
    expect(store.isUserSuspended("u-bob", "acme")).toBe(true);
  });

  it("changes the membership of the workspace it is given, not of the one the address names", async () => {
    // the tab's address names acme
    const { nerve, store } = await loaded();
    const elsewhere = membershipOf("bob", {}, "globex");
    await load(nerve, store, [elsewhere], "globex");
    const guest = membershipOf("bob", { role: 5 }, "globex");
    const updated = track(store.updateMember("globex", "u-bob", { role: 5 }));
    await inTurn(nerve, 2, ["PATCH", "/api/v0/workspace-members/m-globex-bob"], json(200, guest));
    await until(() => updated.settled, "the answer");
    const removed = track(store.removeMemberFromWorkspace("globex", "u-bob"));
    await inTurn(nerve, 3, ["DELETE", "/api/v0/workspace-members/m-globex-bob"], noContent());
    await until(() => removed.settled, "the removal");
    expect(store.getMemberships("acme")).toEqual({ "u-ann": ann, "u-bob": bob, "u-cat": cat });
    expect(store.getMemberships("globex")).toEqual({ "u-bob": { ...guest, is_active: false } });
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
    expect(store.getMemberships("acme")).toEqual({ "u-ann": ann, "u-bob": bob, "u-cat": cat });
  });

  it.each(refusals)("fails, asking nerve nothing, for $change of a member it has not listed", async ({ send }) => {
    const { nerve, store } = await memberStore();
    const sent = await settle(send(store), "the change");
    expect(sent).toMatchObject({ settled: true, error: new Error("Member not found") });
    expect(nerve.calls).toEqual([]);
  });

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const { nerve, store } = await loaded();
    const promoted = track(store.updateMember("acme", "u-bob", { role: 20 }));
    const changed = track(store.updateMember("acme", "u-bob", { role: 5 }));
    const removed = track(store.removeMemberFromWorkspace("acme", "u-bob"));
    await inTurn(nerve, 1, ["PATCH", "/api/v0/workspace-members/m-acme-bob"], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", "/api/v0/workspace-members/m-acme-bob"], json(200, demoted));
    await inTurn(nerve, 3, ["DELETE", "/api/v0/workspace-members/m-acme-bob"], noContent());
    await until(() => removed.settled, "the last change");
    expect(promoted.error).toBeInstanceOf(ApiError);
    expect(changed.value).toEqual(demoted);
    expect(store.getMemberships("acme")?.["u-bob"]).toEqual({ ...demoted, is_active: false });
  });

  it("fetches the members while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    // nerve lists fewer than were loaded: the store holds the fetch's answer, which the refused change leaves alone
    await fetchedWhileChangeIsOut(
      nerve,
      {
        send: () => store.updateMember("acme", "u-bob", { role: 5 }),
        request: ["PATCH", "/api/v0/workspace-members/m-acme-bob"],
      },
      { send: () => store.fetchWorkspaceMembers(workspaceOf("acme")), request: ["GET", MEMBERS], body: { data: [ann] } }
    );
    expect(store.getMemberships("acme")).toEqual({ "u-ann": ann });
  });
});

// A fetch's answer may be older than a change nerve confirmed while it was out: what the fetch shows carries the
// change (the order is forced: the fetch waits until the test answers it, after the change has finished). Of two
// fetches of one workspace's members, only the newer writes. No membership is created here (one is made by accepting
// an invitation, WorkspaceRootStore), so a fetch cannot list one that a change also adds.
describe("WorkspaceMemberStore, while a fetch is out", () => {
  it("keeps a role change and a removal nerve confirmed during a refetch, and the changed member's profile", async () => {
    const { nerve, users, store } = await loaded();
    const refetched = track(store.fetchWorkspaceMembers(workspaceOf("acme")));
    await until(() => nerve.calls.length === 2, "the refetch");
    const updated = track(store.updateMember("acme", "u-bob", { role: 5 }));
    await inTurn(nerve, 2, ["PATCH", "/api/v0/workspace-members/m-acme-bob"], json(200, demoted));
    const removed = track(store.removeMemberFromWorkspace("acme", "u-ann"));
    await inTurn(nerve, 3, ["DELETE", "/api/v0/workspace-members/m-acme-ann"], noContent());
    await until(() => updated.settled && removed.settled, "the changes");
    // the list was read before both: bob a member still, by his old name, and ann's membership not ended
    nerve.calls[1]?.answer(json(200, { data: [ann, bob, cat] }));
    await until(() => refetched.settled, "the refetch");

    const shown = { "u-ann": { ...ann, is_active: false }, "u-bob": demoted, "u-cat": cat };
    expect(store.getMemberships("acme")).toEqual(shown);
    expect(refetched.value).toEqual(shown);
    expect(users["u-bob"]).toEqual(demoted.member);
  });

  it("lets each workspace's newer fetch write: an older one answering last writes nothing, profiles neither", async () => {
    const { nerve, users, store } = await memberStore();
    const older = track(store.fetchWorkspaceMembers(workspaceOf("acme")));
    await until(() => nerve.calls.length === 1, "the older members");
    const globex = track(store.fetchWorkspaceMembers(workspaceOf("globex")));
    await until(() => nerve.calls.length === 2, "globex's members");
    const newer = track(store.fetchWorkspaceMembers(workspaceOf("acme")));
    await until(() => nerve.calls.length === 3, "the newer members");
    nerve.calls[2]?.answer(json(200, { data: [ann, demoted] }));
    await until(() => newer.settled, "the newer members");
    // a newer fetch of acme's members does not overtake one of globex's
    const elsewhere = membershipOf("dee", {}, "globex");
    nerve.calls[1]?.answer(json(200, { data: [elsewhere] }));
    await until(() => globex.settled, "globex's members");
    // read before bob renamed himself
    nerve.calls[0]?.answer(json(200, { data: [ann, bob] }));
    await until(() => older.settled, "the older members");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.getMemberships("acme")).toEqual({ "u-ann": ann, "u-bob": demoted });
    expect(store.getMemberships("globex")).toEqual({ "u-dee": elsewhere });
    expect(users["u-bob"]).toEqual(demoted.member);
  });
});
