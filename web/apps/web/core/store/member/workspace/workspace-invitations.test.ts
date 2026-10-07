/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspaceInvitation, WorkspaceInvitationsCreate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import type { FakeNerve } from "@/lib/auth/fake-nerve";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { inTurn } from "@/store/fake-queue";
import { memberStore } from "@/store/member/workspace/fake-members";
import type { WorkspaceMemberStore } from "@/store/member/workspace/workspace-member.store";

// A workspace's invitations, as its admins manage them (M3 design 7.3), against a fake nerve that answers each
// request when the test says. Who may fetch them is use-members-settings-fetch.test.ts.

// The account's store reads the tab's session as it fetches the account, which no test here does.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

const INVITATIONS = "/api/v0/workspaces/acme/invitations";

/** An invitation of acme as nerve lists it to an admin: the name names the address; pending, unless fields say not. */
function invitationOf(name: string, fields: Partial<WorkspaceInvitation> = {}): WorkspaceInvitation {
  return {
    id: `i-${name}`,
    workspace_id: "id-acme",
    email: `${name}@example.com`,
    role: 15,
    accepted: false,
    responded_at: null,
    created_at: "2026-10-02T09:00:00Z",
    created_by_id: "u-ann",
    token: `nrv_inv_${name}`,
    ...fields,
  };
}
const dan = invitationOf("dan");
/** An invitation its address declined: nerve lists it until an admin deletes it. */
const eve = invitationOf("eve", { role: 5, responded_at: "2026-10-03T09:00:00Z" });

/** The store fetches acme's invitations, and nerve lists these. */
async function load(nerve: FakeNerve, store: WorkspaceMemberStore, invitations: WorkspaceInvitation[]) {
  const at = nerve.calls.length;
  const fetched = store.fetchWorkspaceMemberInvitations("acme");
  await until(() => nerve.calls.length === at + 1, "the invitations");
  nerve.calls[at]?.answer(json(200, { data: invitations }));
  return settle(fetched, "the invitations");
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
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: INVITATIONS });
    expect(fetched.value).toEqual([dan, eve]);
    expect(store.workspaceMemberInvitations).toEqual({ acme: [dan, eve] });
    expect(store.workspaceMemberInvitationIds).toEqual(["i-dan", "i-eve"]);
    expect(store.getWorkspaceInvitationDetails("i-eve")).toEqual(eve);
    expect(store.getSearchedWorkspaceInvitationIds("EVE@")).toEqual(["i-eve"]);
  });

  it("fails when nerve refuses the list, keeping none", async () => {
    const { nerve, store } = memberStore();
    const refused = track(store.fetchWorkspaceMemberInvitations("acme"));
    await until(() => nerve.calls.length === 1, "the invitations");
    nerve.calls[0]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.workspaceMemberInvitations).toEqual({});
  });

  it("gives nothing, and does not fail, when the session changes as it fetches", async () => {
    const { nerve, store } = memberStore((fake) => fake.replacedSessionClient());
    const fetched = await settle(store.fetchWorkspaceMemberInvitations("acme"), "the fetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toEqual([]);
    expect(store.workspaceMemberInvitations).toEqual({});
  });

  it("puts the new invitations first in the list it has, once nerve gives them, and in none it has not fetched", async () => {
    const { nerve, store } = await loaded();
    const body: WorkspaceInvitationsCreate = { invitations: [{ email: "fay@example.com", role: 15 }] };
    const fay = invitationOf("fay");
    const invited = track(store.inviteMembersToWorkspace("acme", body));
    await until(() => nerve.calls.length === 2, "the invitation");
    expect(nerve.calls[1]).toMatchObject({ method: "POST", path: INVITATIONS, body });
    expect(store.workspaceMemberInvitations.acme).toEqual([dan, eve]);
    nerve.calls[1]?.answer(json(201, { data: [fay] }));
    await until(() => invited.settled, "the new invitations");
    expect(invited.value).toEqual([fay]);
    expect(store.workspaceMemberInvitations.acme).toEqual([fay, dan, eve]);

    const fresh = memberStore();
    const alone = track(fresh.store.inviteMembersToWorkspace("acme", body));
    await until(() => fresh.nerve.calls.length === 1, "the invitation");
    fresh.nerve.calls[0]?.answer(json(201, { data: [fay] }));
    await until(() => alone.settled, "the new invitations");
    // a list of the new invitations alone would show as the whole list
    expect(fresh.store.workspaceMemberInvitations).toEqual({});
  });

  it("changes an invitation's role to nerve's answer, and takes a deleted one off the list", async () => {
    const { nerve, store } = await loaded();
    const promoted = invitationOf("dan", { role: 20 });
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
    const promoted = invitationOf("dan", { role: 20 });
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
});
