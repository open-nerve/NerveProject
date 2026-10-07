/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspaceInvitation, WorkspaceInvitationsCreate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { invitationOf, memberStore } from "@/store/member/workspace/fake-members";
import type { WorkspaceMemberStore } from "@/store/member/workspace/workspace-member.store";

// A workspace's invitations, as its admins manage them (M3 design 7.3), against a fake nerve that answers each
// request when the test says. Who may fetch them is use-members-settings-fetch.test.ts.

// The account's store reads the tab's session as it fetches the account, which no test here does.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

const INVITATIONS = "/api/v0/workspaces/acme/invitations";

const dan = invitationOf("dan");
/** An invitation its address declined: nerve lists it until an admin deletes it. */
const eve = invitationOf("eve", { role: 5, responded_at: "2026-10-03T09:00:00Z" });
/** Dan's invitation as nerve answers its change to an admin: the account that invited him is gone meanwhile. */
const promoted = invitationOf("dan", { role: 20, created_by_id: null });

/** The store fetches the invitations of the workspace slug names (acme unless it says), and nerve lists these. */
function load(nerve: FakeNerve, store: WorkspaceMemberStore, invitations: WorkspaceInvitation[], slug = "acme") {
  const fetch = () => store.fetchWorkspaceMemberInvitations(slug);
  const listed = { data: invitations };
  return answered(nerve, fetch, ["GET", `/api/v0/workspaces/${slug}/invitations`], listed, "the invitations");
}

/** A store whose invitations of acme nerve gave as dan's and eve's. */
async function loaded() {
  const tab = memberStore();
  await load(tab.nerve, tab.store, [dan, eve]);
  return tab;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("WorkspaceMemberStore, the invitations", () => {
  it("keeps a workspace's invitations as nerve lists them to an admin, declined ones too", async () => {
    const { nerve, store } = memberStore();
    const fetched = await load(nerve, store, [dan, eve]);
    expect(fetched.value).toEqual([dan, eve]);
    expect(store.workspaceMemberInvitations).toEqual({ acme: [dan, eve] });
    expect(store.workspaceMemberInvitationIds).toEqual(["i-dan", "i-eve"]);
    expect(store.getWorkspaceInvitationDetails("i-eve")).toEqual(eve);
    expect(store.getSearchedWorkspaceInvitationIds("EVE@")).toEqual(["i-eve"]);
  });

  it("fails when nerve refuses the list, keeping the invitations it had", async () => {
    const { nerve, store } = memberStore();
    const refused = track(store.fetchWorkspaceMemberInvitations("acme"));
    await until(() => nerve.calls.length === 1, "the invitations");
    nerve.calls[0]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.workspaceMemberInvitations).toEqual({});

    await load(nerve, store, [dan, eve]);
    const again = track(store.fetchWorkspaceMemberInvitations("acme"));
    await until(() => nerve.calls.length === 3, "the refetch");
    nerve.calls[2]?.answer(problem(403, "forbidden"));
    await until(() => again.settled, "the refusal");
    expect(again.error).toBeInstanceOf(ApiError);
    expect(store.workspaceMemberInvitations).toEqual({ acme: [dan, eve] });
  });

  it("gives nothing, and does not fail, when the session changes as it fetches", async () => {
    const { nerve, store } = memberStore((fake) => fake.replacedSessionClient());
    const fetched = await settle(store.fetchWorkspaceMemberInvitations("acme"), "the fetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toEqual([]);
    expect(store.workspaceMemberInvitations).toEqual({});
  });

  it("keeps the invitations it had when the session changes as it fetches them again", async () => {
    const { nerve, api, store } = await loaded();
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchWorkspaceMemberInvitations("acme"), "the refetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(store.workspaceMemberInvitations).toEqual({ acme: [dan, eve] });
  });

  it("puts the new invitations first in the list it has, once nerve gives them, and in none it has not fetched", async () => {
    const { nerve, store } = await loaded();
    const body: WorkspaceInvitationsCreate = {
      invitations: [
        { email: "fay@example.com", role: 15 },
        { email: "gus@example.com", role: 5 },
      ],
    };
    const fay = invitationOf("fay");
    const gus = invitationOf("gus", { role: 5 });
    const invited = track(store.inviteMembersToWorkspace("acme", body));
    await until(() => nerve.calls.length === 2, "the invitations");
    expect(nerve.calls[1]).toMatchObject({ method: "POST", path: INVITATIONS, body });
    expect(store.workspaceMemberInvitations.acme).toEqual([dan, eve]);
    nerve.calls[1]?.answer(json(201, { data: [fay, gus] }));
    await until(() => invited.settled, "the new invitations");
    expect(invited.value).toEqual([fay, gus]);
    expect(store.workspaceMemberInvitations.acme).toEqual([fay, gus, dan, eve]);
    // nerve's answer is all the store takes: it does not fetch the list again
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(2);

    const fresh = memberStore();
    const alone = track(fresh.store.inviteMembersToWorkspace("acme", body));
    await until(() => fresh.nerve.calls.length === 1, "the invitations");
    fresh.nerve.calls[0]?.answer(json(201, { data: [fay, gus] }));
    await until(() => alone.settled, "the new invitations");
    // a list of the new invitations alone would show as the whole list
    expect(fresh.store.workspaceMemberInvitations).toEqual({});
  });

  it("changes an invitation's role to nerve's answer, and takes a deleted one off the list", async () => {
    const { nerve, store } = await loaded();
    const changed = track(store.updateMemberInvitation("acme", "i-dan", { role: 20 }));
    await until(() => nerve.calls.length === 2, "the change");
    expect(nerve.calls[1]).toMatchObject({
      method: "PATCH",
      path: "/api/v0/workspace-invitations/i-dan",
      body: { role: 20 },
    });
    expect(store.workspaceMemberInvitations.acme).toEqual([dan, eve]);
    nerve.calls[1]?.answer(json(200, promoted));
    await until(() => changed.settled, "the answer");
    expect(changed.value).toEqual(promoted);
    // the answer, not the request: the inviter's account is gone, which no request says
    expect(store.workspaceMemberInvitations.acme).toEqual([promoted, eve]);

    const deleted = track(store.deleteMemberInvitation("acme", "i-eve"));
    await until(() => nerve.calls.length === 3, "the deletion");
    expect(nerve.calls[2]).toMatchObject({ method: "DELETE", path: "/api/v0/workspace-invitations/i-eve" });
    expect(store.workspaceMemberInvitations.acme).toEqual([promoted, eve]);
    nerve.calls[2]?.answer(noContent());
    await until(() => deleted.settled, "the deletion");
    expect(deleted.error).toBeUndefined();
    expect(store.workspaceMemberInvitations.acme).toEqual([promoted]);
  });

  it("changes the invitations of the workspace it is given, not of the one the address names", async () => {
    // the tab's address names acme
    const { nerve, store } = await loaded();
    const gil = invitationOf("gil", {}, "globex");
    const hal = invitationOf("hal", {}, "globex");
    await load(nerve, store, [gil, hal], "globex");
    const ivy = invitationOf("ivy", {}, "globex");
    const invited = track(
      store.inviteMembersToWorkspace("globex", { invitations: [{ email: "ivy@example.com", role: 15 }] })
    );
    await inTurn(nerve, 2, ["POST", "/api/v0/workspaces/globex/invitations"], json(201, { data: [ivy] }));
    await until(() => invited.settled, "the new invitation");
    const admin = invitationOf("gil", { role: 20 }, "globex");
    const changed = track(store.updateMemberInvitation("globex", "i-gil", { role: 20 }));
    await inTurn(nerve, 3, ["PATCH", "/api/v0/workspace-invitations/i-gil"], json(200, admin));
    await until(() => changed.settled, "the answer");
    const deleted = track(store.deleteMemberInvitation("globex", "i-hal"));
    await inTurn(nerve, 4, ["DELETE", "/api/v0/workspace-invitations/i-hal"], noContent());
    await until(() => deleted.settled, "the deletion");
    expect(store.workspaceMemberInvitations).toEqual({ acme: [dan, eve], globex: [ivy, admin] });
  });

  const refusals: { change: string; send: (store: WorkspaceMemberStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "invitations",
      send: (store) =>
        store.inviteMembersToWorkspace("acme", { invitations: [{ email: "dan@example.com", role: 15 }] }),
      refusal: problem(422, "validation_failed"),
    },
    {
      change: "a role change",
      send: (store) => store.updateMemberInvitation("acme", "i-eve", { role: 15 }),
      refusal: problem(409, "workspace.invitation_responded"),
    },
    {
      change: "a deletion",
      send: (store) => store.deleteMemberInvitation("acme", "i-dan"),
      refusal: problem(404, "workspace.invitation_not_found"),
    },
  ];
  it.each(refusals)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, store } = await loaded();
    const sent = track(send(store));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(refusal);
    await until(() => sent.settled, "the refusal");
    expect(sent.error).toBeInstanceOf(ApiError);
    expect(store.workspaceMemberInvitations.acme).toEqual([dan, eve]);
  });

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const { nerve, store } = await loaded();
    const invited = track(
      store.inviteMembersToWorkspace("acme", { invitations: [{ email: "fay@example.com", role: 15 }] })
    );
    const changed = track(store.updateMemberInvitation("acme", "i-dan", { role: 20 }));
    const deleted = track(store.deleteMemberInvitation("acme", "i-eve"));
    await inTurn(nerve, 1, ["POST", INVITATIONS], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", "/api/v0/workspace-invitations/i-dan"], json(200, promoted));
    await inTurn(nerve, 3, ["DELETE", "/api/v0/workspace-invitations/i-eve"], noContent());
    await until(() => deleted.settled, "the last change");
    expect(invited.error).toBeInstanceOf(ApiError);
    expect(changed.value).toEqual(promoted);
    expect(store.workspaceMemberInvitations.acme).toEqual([promoted]);
  });

  it("fetches the invitations while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      () => store.updateMemberInvitation("acme", "i-dan", { role: 20 }),
      () => store.fetchWorkspaceMemberInvitations("acme"),
      ["GET", INVITATIONS],
      json(200, { data: [eve] })
    );
    expect(store.workspaceMemberInvitations).toEqual({ acme: [eve] });
  });
});
